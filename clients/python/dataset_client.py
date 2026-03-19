"""Sample Python helper for the ColonyCore Dataset Service.

This helper maps to the OpenAPI contract defined in docs/schema/dataset-service.openapi.yaml
and demonstrates how analysts can script dataset access and exports from their own runtime.
"""

from __future__ import annotations

import json
import os
import time
import warnings
from dataclasses import dataclass, field
from typing import Any, Dict, Iterable, List, Optional

import requests

_DEFAULT_HEADERS = {
    "Accept": "application/json",
    "User-Agent": "colonycore-dataset-client/0.1",
}
_MAX_TEMPLATE_PAGES = 1000
_DATASET_SCOPE_HEADERS = {
    "requestor": "X-Dataset-Requestor",
    "roles": "X-Dataset-Roles",
    "project_ids": "X-Dataset-Project-Ids",
    "protocol_ids": "X-Dataset-Protocol-Ids",
}
_STREAM_PROGRESS_HEADER = "X-Progress"


@dataclass
class ExportHandle:
    """Represents an asynchronous export job."""

    id: str
    status: str
    progress_pct: int = 0
    eta_seconds: Optional[int] = None
    progress_state: Optional[str] = None
    artifact_readiness: Optional[str] = None
    artifacts: List[Dict[str, Any]] = field(default_factory=list)
    error: Optional[str] = None
    template: Dict[str, Any] = field(default_factory=dict)
    scope: Dict[str, Any] = field(default_factory=dict)
    parameters: Dict[str, Any] = field(default_factory=dict)
    formats: List[str] = field(default_factory=list)
    requested_by: Optional[str] = None
    reason: Optional[str] = None
    project_id: Optional[str] = None
    protocol_id: Optional[str] = None
    created_at: Optional[str] = None
    updated_at: Optional[str] = None
    completed_at: Optional[str] = None


@dataclass
class CSVRunResult:
    """Represents a streamed CSV response plus transport metadata."""

    body: str
    progress: Optional[str]
    headers: Dict[str, str] = field(default_factory=dict)


class DatasetClient:
    """Thin convenience wrapper around the DatasetService REST API."""

    def __init__(
        self,
        base_url: str,
        api_key: Optional[str] = None,
        timeout: int = 30,
        session: Optional[requests.Session] = None,
    ) -> None:
        self._base_url = base_url.rstrip("/")
        self._timeout = timeout
        self._session = session or requests.Session()
        for header, value in _DEFAULT_HEADERS.items():
            self._session.headers.setdefault(header, value)
        if api_key:
            self._session.headers.setdefault("Authorization", f"Bearer {api_key}")

    # Template helpers -----------------------------------------------------
    def _scope_headers(self, scope: Optional[Dict[str, Any]]) -> Dict[str, str]:
        if not scope:
            return {}

        headers: Dict[str, str] = {}
        for key, header in _DATASET_SCOPE_HEADERS.items():
            value = scope.get(key)
            if value is None or value == "":
                continue
            if isinstance(value, (list, tuple, set)):
                items = [str(item).strip() for item in value if str(item).strip()]
                if items:
                    headers[header] = ",".join(items)
                continue
            headers[header] = str(value)
        return headers

    def _export_handle(self, payload: Dict[str, Any]) -> ExportHandle:
        return ExportHandle(
            id=payload["id"],
            status=payload["status"],
            progress_pct=payload.get("progress_pct", 0),
            eta_seconds=payload.get("eta_seconds"),
            progress_state=payload.get("progress_state"),
            artifact_readiness=payload.get("artifact_readiness"),
            artifacts=list(payload.get("artifacts") or []),
            error=payload.get("error"),
            template=dict(payload.get("template") or {}),
            scope=dict(payload.get("scope") or {}),
            parameters=dict(payload.get("parameters") or {}),
            formats=list(payload.get("formats") or []),
            requested_by=payload.get("requested_by"),
            reason=payload.get("reason"),
            project_id=payload.get("project_id"),
            protocol_id=payload.get("protocol_id"),
            created_at=payload.get("created_at"),
            updated_at=payload.get("updated_at"),
            completed_at=payload.get("completed_at"),
        )

    def list_templates_page(
        self,
        page: int = 1,
        page_size: int = 50,
        scope: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        resp = self._session.get(
            f"{self._base_url}/api/v1/datasets/templates",
            params={"page": page, "page_size": page_size},
            headers=self._scope_headers(scope),
            timeout=self._timeout,
        )
        resp.raise_for_status()
        return resp.json()

    def list_templates(
        self,
        page: int = 1,
        page_size: int = 50,
        scope: Optional[Dict[str, Any]] = None,
    ) -> List[Dict[str, Any]]:
        templates: List[Dict[str, Any]] = []
        current_page = page
        pages_fetched = 0

        while True:
            payload = self.list_templates_page(page=current_page, page_size=page_size, scope=scope)
            pages_fetched += 1
            page_templates = payload.get("templates", [])
            if not page_templates:
                break

            templates.extend(page_templates)
            if not (payload.get("pagination") or {}).get("has_next"):
                break
            if pages_fetched >= _MAX_TEMPLATE_PAGES:
                warnings.warn(
                    f"template listing truncated after {_MAX_TEMPLATE_PAGES} pages; additional pages were not fetched",
                    RuntimeWarning,
                    stacklevel=2,
                )
                break

            current_page += 1

        return templates

    def get_template(self, plugin: str, key: str, version: str) -> Dict[str, Any]:
        path = f"{self._base_url}/api/v1/datasets/templates/{plugin}/{key}/{version}"
        resp = self._session.get(path, timeout=self._timeout)
        resp.raise_for_status()
        return resp.json()["template"]

    def validate_template(
        self,
        plugin: str,
        key: str,
        version: str,
        parameters: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        path = f"{self._base_url}/api/v1/datasets/templates/{plugin}/{key}/{version}/validate"
        resp = self._session.post(path, json={"parameters": parameters or {}}, timeout=self._timeout)
        resp.raise_for_status()
        return resp.json()

    def run_template(
        self,
        plugin: str,
        key: str,
        version: str,
        parameters: Optional[Dict[str, Any]] = None,
        scope: Optional[Dict[str, Any]] = None,
        output_format: str = "json",
        include_stream_metadata: bool = False,
    ) -> Any:
        path = f"{self._base_url}/api/v1/datasets/templates/{plugin}/{key}/{version}/run"
        headers = {}
        fmt = output_format.lower()
        if fmt == "csv":
            headers["Accept"] = "text/csv"
        payload = {
            "parameters": parameters or {},
            "scope": scope or {},
        }
        resp = self._session.post(path, params={"format": fmt}, json=payload, headers=headers, timeout=self._timeout)
        resp.raise_for_status()
        if fmt == "csv":
            if include_stream_metadata:
                return CSVRunResult(
                    body=resp.text,
                    progress=resp.headers.get(_STREAM_PROGRESS_HEADER),
                    headers=dict(resp.headers),
                )
            return resp.text
        return resp.json()

    # Export helpers -------------------------------------------------------
    def queue_export(
        self,
        template_slug: Optional[str] = None,
        *,
        plugin: Optional[str] = None,
        key: Optional[str] = None,
        version: Optional[str] = None,
        parameters: Optional[Dict[str, Any]] = None,
        scope: Optional[Dict[str, Any]] = None,
        formats: Optional[Iterable[str]] = None,
        requested_by: Optional[str] = None,
        reason: Optional[str] = None,
        project_id: Optional[str] = None,
        protocol_id: Optional[str] = None,
    ) -> ExportHandle:
        template: Dict[str, Any]
        if template_slug:
            template = {"slug": template_slug}
        else:
            if not all([plugin, key, version]):
                raise ValueError("plugin, key, and version are required when slug is omitted")
            template = {"plugin": plugin, "key": key, "version": version}
        body = {
            "template": template,
            "parameters": parameters or {},
            "scope": scope or {},
            "formats": list(formats or []),
            "requested_by": requested_by,
            "reason": reason,
            "project_id": project_id,
            "protocol_id": protocol_id,
        }
        resp = self._session.post(f"{self._base_url}/api/v1/datasets/exports", json=body, timeout=self._timeout)
        resp.raise_for_status()
        payload = resp.json()["export"]
        return self._export_handle(payload)

    def get_export(self, export_id: str) -> ExportHandle:
        resp = self._session.get(f"{self._base_url}/api/v1/datasets/exports/{export_id}", timeout=self._timeout)
        resp.raise_for_status()
        payload = resp.json()["export"]
        return self._export_handle(payload)

    def wait_for_export(
        self,
        export_id: str,
        poll_interval: float = 2.0,
        timeout: float = 300.0,
    ) -> ExportHandle:
        deadline = time.time() + timeout
        handle = self.get_export(export_id)
        while handle.status not in {"succeeded", "failed"}:
            if time.time() > deadline:
                raise TimeoutError(f"export {export_id} did not complete within {timeout} seconds")
            time.sleep(poll_interval)
            handle = self.get_export(export_id)
        return handle

    def download_artifact(self, url: str, destination: Optional[str] = None) -> str:
        """Download an artifact using the signed URL returned by the export service."""
        resp = self._session.get(url, timeout=self._timeout)
        resp.raise_for_status()
        if destination is None:
            return resp.text
        os.makedirs(os.path.dirname(destination) or ".", exist_ok=True)
        mode = "wb" if isinstance(resp.content, (bytes, bytearray)) else "w"
        with open(destination, mode) as fh:
            fh.write(resp.content if "b" in mode else resp.text)
        return destination


def dump_pretty(obj: Any) -> str:
    """Render response payloads as formatted JSON for notebooks or logs."""
    return json.dumps(obj, indent=2, sort_keys=True)
