"""Sample Python helper for the ColonyCore Dataset Service.

This helper maps to the OpenAPI contract defined in docs/schema/dataset-service.openapi.yaml
and demonstrates how analysts can script dataset access and exports from their own runtime.
"""

from __future__ import annotations

import json
import os
import time
from dataclasses import dataclass
from typing import Any, Dict, Iterable, List, Optional

import requests

_DEFAULT_HEADERS = {
    "Accept": "application/json",
    "User-Agent": "colonycore-dataset-client/0.1",
}


@dataclass
class ExportHandle:
    """Represents an asynchronous export job."""

    id: str
    status: str
    artifacts: List[Dict[str, Any]]


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
    def list_templates(self) -> List[Dict[str, Any]]:
        resp = self._session.get(f"{self._base_url}/api/v1/datasets/templates", timeout=self._timeout)
        resp.raise_for_status()
        payload = resp.json()
        return payload.get("templates", [])

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
        return ExportHandle(id=payload["id"], status=payload["status"], artifacts=payload.get("artifacts", []))

    def get_export(self, export_id: str) -> ExportHandle:
        resp = self._session.get(f"{self._base_url}/api/v1/datasets/exports/{export_id}", timeout=self._timeout)
        resp.raise_for_status()
        payload = resp.json()["export"]
        return ExportHandle(id=payload["id"], status=payload["status"], artifacts=payload.get("artifacts", []))

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
