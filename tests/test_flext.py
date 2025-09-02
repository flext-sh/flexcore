"""FLEXT Service Integration Tests.

Tests the integration between FlexCore (Go) and FLEXT Service (Python).
These tests validate the cross-runtime coordination and service communication.

Copyright (c) 2025 Flext. All rights reserved.
SPDX-License-Identifier: MIT
"""

from __future__ import annotations

import time

import pytest
import requests
from requests.adapters import HTTPAdapter
from urllib3.util.retry import Retry


class TestFlextServiceIntegration:
    """Test FlexCore integration with FLEXT Service."""

    FLEXCORE_URL = "http://localhost:8080"
    FLEXT_SERVICE_URL = "http://localhost:8081"

    @pytest.fixture(autouse=True)
    def setup_http_session(self) -> None:
        """Setup HTTP session with retry strategy."""
        self.session = requests.Session()
        retry_strategy = Retry(
            total=3,
            backoff_factor=1,
            status_forcelist=[429, 500, 502, 503, 504]
        )
        adapter = HTTPAdapter(max_retries=retry_strategy)
        self.session.mount("http://", adapter)
        self.session.mount("https://", adapter)

    def test_services_health_check(self) -> None:
        """Test that both services are healthy and responding."""
        # Test FlexCore health
        try:
            flexcore_response = self.session.get(f"{self.FLEXCORE_URL}/health", timeout=5)
            assert flexcore_response.status_code == 200
            flexcore_health = flexcore_response.json()
            assert flexcore_health.get("status") == "healthy"
        except requests.exceptions.RequestException as e:
            pytest.skip(f"FlexCore service not available: {e}")

        # Test FLEXT Service health
        try:
            flext_response = self.session.get(f"{self.FLEXT_SERVICE_URL}/health", timeout=5)
            assert flext_response.status_code == 200
            flext_health = flext_response.json()
            assert flext_health.get("status") == "healthy"
        except requests.exceptions.RequestException as e:
            pytest.skip(f"FLEXT service not available: {e}")

    def test_cross_service_communication(self) -> None:
        """Test communication between FlexCore and FLEXT Service."""
        # Create a workflow execution request via FLEXT Service
        workflow_data = {
            "workflow_id": "/test/cross-service",
            "input_data": {
                "message": "Test cross-service communication",
                "timestamp": int(time.time())
            },
            "target_runtime": "flexcore"
        }

        try:
            # Send workflow to FLEXT Service
            flext_response = self.session.post(
                f"{self.FLEXT_SERVICE_URL}/api/v1/workflows/execute",
                json=workflow_data,
                timeout=10
            )
            assert flext_response.status_code in [200, 201, 202]

            execution_result = flext_response.json()
            job_id = execution_result.get("job_id")
            assert job_id is not None

            # Verify execution status via FlexCore
            time.sleep(2)  # Allow processing time

            status_response = self.session.get(
                f"{self.FLEXCORE_URL}/api/v1/workflows/{job_id}/status",
                timeout=5
            )
            assert status_response.status_code == 200

            status_data = status_response.json()
            assert status_data.get("job_id") == job_id
            assert status_data.get("status") in ["running", "completed", "pending"]

        except requests.exceptions.RequestException as e:
            pytest.skip(f"Cross-service communication test skipped: {e}")

    def test_runtime_coordination(self) -> None:
        """Test runtime coordination between Go and Python."""
        coordination_data = {
            "runtime_type": "meltano",
            "coordination_request": {
                "operation": "data_processing",
                "source": "postgres://test_db",
                "target": "flexcore://pipeline/test"
            }
        }

        try:
            # Request runtime coordination
            coord_response = self.session.post(
                f"{self.FLEXT_SERVICE_URL}/api/v1/runtimes/coordinate",
                json=coordination_data,
                timeout=15
            )
            assert coord_response.status_code in [200, 201, 202]

            coord_result = coord_response.json()
            assert coord_result.get("status") == "coordinated"
            assert "flexcore_endpoint" in coord_result

            # Verify coordination via FlexCore
            flexcore_endpoint = coord_result["flexcore_endpoint"]
            verification_response = self.session.get(
                f"{self.FLEXCORE_URL}{flexcore_endpoint}",
                timeout=5
            )
            assert verification_response.status_code == 200

        except requests.exceptions.RequestException as e:
            pytest.skip(f"Runtime coordination test skipped: {e}")

    def test_event_streaming(self) -> None:
        """Test event streaming between services."""
        event_data = {
            "event_type": "data.processing.started",
            "source": "flexcore",
            "target": "flext-service",
            "payload": {
                "pipeline_id": "test-pipeline-123",
                "data_size": 1024,
                "processing_type": "transformation"
            }
        }

        try:
            # Send event from FlexCore to FLEXT Service
            event_response = self.session.post(
                f"{self.FLEXCORE_URL}/api/v1/events/publish",
                json=event_data,
                timeout=10
            )
            assert event_response.status_code in [200, 202]

            # Verify event was received by FLEXT Service
            time.sleep(1)  # Allow event processing time

            events_response = self.session.get(
                f"{self.FLEXT_SERVICE_URL}/api/v1/events/recent",
                params={"event_type": event_data["event_type"]},
                timeout=5
            )
            assert events_response.status_code == 200

            recent_events = events_response.json()
            assert isinstance(recent_events, list)

            # Find our event
            test_event = None
            for event in recent_events:
                if event.get("payload", {}).get("pipeline_id") == "test-pipeline-123":
                    test_event = event
                    break

            assert test_event is not None
            assert test_event["event_type"] == event_data["event_type"]

        except requests.exceptions.RequestException as e:
            pytest.skip(f"Event streaming test skipped: {e}")

    def test_plugin_coordination(self) -> None:
        """Test plugin coordination between Go and Python runtimes."""
        plugin_config = {
            "plugin_id": "test-bridge-plugin",
            "runtime_source": "go",
            "runtime_target": "python",
            "coordination_type": "data_bridge",
            "config": {
                "buffer_size": 1000,
                "timeout": 30
            }
        }

        try:
            # Request plugin coordination via FlexCore
            plugin_response = self.session.post(
                f"{self.FLEXCORE_URL}/api/v1/plugins/coordinate",
                json=plugin_config,
                timeout=10
            )
            assert plugin_response.status_code in [200, 201]

            coordination_result = plugin_response.json()
            assert coordination_result.get("status") == "coordinated"
            assert "bridge_endpoint" in coordination_result

            # Verify bridge is active via FLEXT Service
            bridge_response = self.session.get(
                f"{self.FLEXT_SERVICE_URL}/api/v1/bridges/status",
                params={"plugin_id": plugin_config["plugin_id"]},
                timeout=5
            )
            assert bridge_response.status_code == 200

            bridge_status = bridge_response.json()
            assert bridge_status.get("plugin_id") == plugin_config["plugin_id"]
            assert bridge_status.get("status") in ["active", "ready"]

        except requests.exceptions.RequestException as e:
            pytest.skip(f"Plugin coordination test skipped: {e}")


class TestFlextWorkflowIntegration:
    """Test workflow integration between services."""

    def test_windmill_workflow_execution(self) -> None:
        """Test Windmill workflow execution coordination."""
        workflow_spec = {
            "workflow_path": "/flext/integration/test",
            "input_schema": {
                "message": "string",
                "processing_type": "string"
            },
            "runtime_coordination": {
                "go_runtime": "flexcore",
                "python_runtime": "flext-service"
            }
        }

        try:
            # Deploy workflow
            deploy_response = requests.post(
                "http://localhost:8080/api/v1/windmill/workflows/deploy",
                json=workflow_spec,
                timeout=10
            )
            assert deploy_response.status_code in [200, 201]

            # Execute workflow
            execution_data = {
                "workflow_path": workflow_spec["workflow_path"],
                "args": {
                    "message": "Test Windmill integration",
                    "processing_type": "cross_runtime"
                }
            }

            exec_response = requests.post(
                "http://localhost:8080/api/v1/windmill/workflows/execute",
                json=execution_data,
                timeout=15
            )
            assert exec_response.status_code in [200, 202]

            exec_result = exec_response.json()
            job_id = exec_result.get("job_id")
            assert job_id is not None

            # Monitor execution status
            max_wait = 30
            wait_time = 0
            status = None

            while wait_time < max_wait:
                status_response = requests.get(
                    f"http://localhost:8080/api/v1/windmill/jobs/{job_id}/status",
                    timeout=5
                )

                if status_response.status_code == 200:
                    status_data = status_response.json()
                    status = status_data.get("status")

                    if status in ["completed", "failed"]:
                        break

                time.sleep(2)
                wait_time += 2

            assert status == "completed"

        except requests.exceptions.RequestException as e:
            pytest.skip(f"Windmill workflow test skipped: {e}")


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
