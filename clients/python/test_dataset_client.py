import unittest

from dataset_client import CSVRunResult, DatasetClient


class FakeResponse:
    def __init__(self, *, payload=None, text="", headers=None):
        self._payload = payload or {}
        self.text = text
        self.headers = headers or {}

    def raise_for_status(self):
        return None

    def json(self):
        return self._payload


class FakeSession:
    def __init__(self, responses):
        self._responses = list(responses)
        self.headers = {}
        self.calls = []

    def _next_response(self):
        if not self._responses:
            raise AssertionError("no fake responses remaining")
        return self._responses.pop(0)

    def post(self, url, **kwargs):
        self.calls.append(("POST", url, kwargs))
        return self._next_response()

    def get(self, url, **kwargs):
        self.calls.append(("GET", url, kwargs))
        return self._next_response()


def export_payload(**overrides):
    payload = {
        "id": "export-1",
        "status": "running",
        "progress_pct": 45,
        "eta_seconds": 12,
        "progress_state": "executing_template",
        "artifact_readiness": "pending",
        "artifacts": [{"id": "artifact-1"}],
        "template": {"slug": "frog/report@1"},
        "scope": {"project_ids": ["project-1"]},
        "parameters": {"stage": "adult"},
        "formats": ["json"],
        "requested_by": "analyst",
        "reason": "nightly refresh",
        "project_id": "project-1",
        "protocol_id": "protocol-1",
        "created_at": "2026-03-19T00:00:00Z",
        "updated_at": "2026-03-19T00:00:05Z",
        "completed_at": None,
    }
    payload.update(overrides)
    return {"export": payload}


class DatasetClientTest(unittest.TestCase):
    def test_queue_export_returns_progress_fields(self):
        session = FakeSession([FakeResponse(payload=export_payload())])
        client = DatasetClient("https://api.example.test", session=session)

        handle = client.queue_export(
            template_slug="frog/report@1",
            formats=["json"],
            requested_by="analyst",
        )

        self.assertEqual("export-1", handle.id)
        self.assertEqual("running", handle.status)
        self.assertEqual(45, handle.progress_pct)
        self.assertEqual(12, handle.eta_seconds)
        self.assertEqual("executing_template", handle.progress_state)
        self.assertEqual("pending", handle.artifact_readiness)
        self.assertEqual({"slug": "frog/report@1"}, handle.template)
        self.assertEqual(["json"], handle.formats)
        self.assertEqual("analyst", handle.requested_by)

    def test_get_export_preserves_ready_artifacts(self):
        session = FakeSession(
            [
                FakeResponse(
                    payload=export_payload(
                        status="succeeded",
                        progress_pct=100,
                        eta_seconds=None,
                        progress_state="completed",
                        artifact_readiness="ready",
                        artifacts=[{"id": "artifact-1"}, {"id": "artifact-2"}],
                    )
                )
            ]
        )
        client = DatasetClient("https://api.example.test", session=session)

        handle = client.get_export("export-1")

        self.assertEqual("succeeded", handle.status)
        self.assertEqual(100, handle.progress_pct)
        self.assertIsNone(handle.eta_seconds)
        self.assertEqual("completed", handle.progress_state)
        self.assertEqual("ready", handle.artifact_readiness)
        self.assertEqual(2, len(handle.artifacts))

    def test_run_template_csv_can_return_stream_metadata(self):
        session = FakeSession(
            [
                FakeResponse(
                    text="value\n7\n",
                    headers={"X-Progress": "bytes=0/8", "Content-Type": "text/csv"},
                )
            ]
        )
        client = DatasetClient("https://api.example.test", session=session)

        result = client.run_template(
            "frog",
            "report",
            "1",
            output_format="csv",
            include_stream_metadata=True,
        )

        self.assertIsInstance(result, CSVRunResult)
        self.assertEqual("value\n7\n", result.body)
        self.assertEqual("bytes=0/8", result.progress)
        self.assertEqual("text/csv", result.headers["Content-Type"])
        method, url, kwargs = session.calls[0]
        self.assertEqual("POST", method)
        self.assertTrue(url.endswith("/api/v1/datasets/templates/frog/report/1/run"))
        self.assertEqual("csv", kwargs["params"]["format"])
        self.assertEqual("text/csv", kwargs["headers"]["Accept"])


if __name__ == "__main__":
    unittest.main()
