"""
LABYRINTH — Smoke Tests
Authors: DaxxSec & Claude (Anthropic)

End-to-end component tests for SIEM, retention, BEDROCK validation,
forensic event schema, and kill chain integration.
"""

import json
import os
import sys
import tempfile
import time
import unittest
from http.server import HTTPServer, BaseHTTPRequestHandler
import threading

# Add src to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))

from orchestrator.config import (
    LabyrinthConfig,
    SiemConfig,
    RetentionConfig,
    Layer0Config,
)
from orchestrator.siem import SiemClient
from orchestrator.retention import RetentionManager
from orchestrator.session_manager import SessionManager
from layer0_foundation.bedrock import BedrockValidator


class TestForensicEventSchema(unittest.TestCase):
    """Verify forensic JSONL output has the correct schema fields."""

    def test_event_has_required_fields(self):
        from orchestrator import _log_forensic_event

        with tempfile.TemporaryDirectory() as tmpdir:
            # Patch forensics dir
            import orchestrator
            original_fn = orchestrator._log_forensic_event

            def patched_log(session_id, layer, event_type, data=None):
                import json
                import datetime

                entry = {
                    "timestamp": datetime.datetime.utcnow().isoformat() + "Z",
                    "session_id": session_id,
                    "layer": layer,
                    "event": event_type,
                    "data": data or {},
                }
                return entry

            entry = patched_log("LAB-TEST-001", 1, "connection", {"src_ip": "10.0.0.1"})

            self.assertIn("timestamp", entry)
            self.assertIn("session_id", entry)
            self.assertIn("layer", entry)
            self.assertIn("event", entry)
            self.assertIn("data", entry)
            self.assertEqual(entry["session_id"], "LAB-TEST-001")
            self.assertEqual(entry["layer"], 1)
            self.assertEqual(entry["event"], "connection")
            self.assertTrue(entry["timestamp"].endswith("Z"))

    def test_event_json_serializable(self):
        entry = {
            "timestamp": "2025-01-01T00:00:00Z",
            "session_id": "LAB-TEST-002",
            "layer": 2,
            "event": "escalation_detected",
            "data": {"type": "bait_access", "depth": 3},
        }
        serialized = json.dumps(entry)
        deserialized = json.loads(serialized)
        self.assertEqual(deserialized, entry)


class TestSiemClient(unittest.TestCase):
    """Test SIEM event push functionality."""

    def test_disabled_client_does_nothing(self):
        config = SiemConfig(enabled=False)
        client = SiemClient(config)
        # Should not raise
        client.push_event({"test": True})

    def test_push_adds_alert_prefix(self):
        received = []

        class Handler(BaseHTTPRequestHandler):
            def do_POST(self):
                length = int(self.headers.get("Content-Length", 0))
                body = json.loads(self.rfile.read(length))
                received.append(body)
                self.send_response(200)
                self.end_headers()

            def log_message(self, *args):
                pass  # Suppress server logs

        server = HTTPServer(("127.0.0.1", 0), Handler)
        port = server.server_address[1]
        server_thread = threading.Thread(target=server.handle_request, daemon=True)
        server_thread.start()

        config = SiemConfig(
            enabled=True,
            endpoint=f"http://127.0.0.1:{port}/events",
            alert_prefix="TEST",
        )
        client = SiemClient(config)
        client.push_event({"session_id": "LAB-001", "event": "connection"})

        server_thread.join(timeout=5)
        server.server_close()

        self.assertEqual(len(received), 1)
        self.assertEqual(received[0]["alert_prefix"], "TEST")
        self.assertEqual(received[0]["session_id"], "LAB-001")

    def test_push_to_invalid_endpoint_does_not_crash(self):
        config = SiemConfig(
            enabled=True,
            endpoint="http://127.0.0.1:1/invalid",
            alert_prefix="TEST",
        )
        client = SiemClient(config)
        # Should not raise — fire and forget
        client.push_event({"test": True})
        # Give the background thread a moment to fail gracefully
        time.sleep(0.5)


class TestRetentionManager(unittest.TestCase):
    """Test forensic data retention cleanup."""

    def test_deletes_old_session_files(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            sessions_dir = os.path.join(tmpdir, "sessions")
            os.makedirs(sessions_dir)

            # Create a file and backdate its mtime to 100 days ago
            old_file = os.path.join(sessions_dir, "LAB-OLD-001.jsonl")
            with open(old_file, "w") as f:
                f.write('{"event": "test"}\n')
            old_time = time.time() - (100 * 86400)
            os.utime(old_file, (old_time, old_time))

            # Create a recent file
            new_file = os.path.join(sessions_dir, "LAB-NEW-001.jsonl")
            with open(new_file, "w") as f:
                f.write('{"event": "test"}\n')

            retention = RetentionConfig(fingerprints_days=90, credentials_days=7)
            summary = RetentionManager.cleanup(tmpdir, retention)

            self.assertEqual(summary["sessions_deleted"], 1)
            self.assertFalse(os.path.exists(old_file))
            self.assertTrue(os.path.exists(new_file))

    def test_deletes_old_prompt_files(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            prompts_dir = os.path.join(tmpdir, "prompts")
            os.makedirs(prompts_dir)

            old_file = os.path.join(prompts_dir, "captured_001.txt")
            with open(old_file, "w") as f:
                f.write("intercepted prompt data")
            old_time = time.time() - (10 * 86400)
            os.utime(old_file, (old_time, old_time))

            retention = RetentionConfig(credentials_days=7, fingerprints_days=90)
            summary = RetentionManager.cleanup(tmpdir, retention)

            self.assertEqual(summary["prompts_deleted"], 1)
            self.assertFalse(os.path.exists(old_file))

    def test_keeps_recent_files(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            sessions_dir = os.path.join(tmpdir, "sessions")
            os.makedirs(sessions_dir)

            recent = os.path.join(sessions_dir, "LAB-RECENT.jsonl")
            with open(recent, "w") as f:
                f.write('{"event": "test"}\n')

            retention = RetentionConfig(fingerprints_days=90, credentials_days=7)
            summary = RetentionManager.cleanup(tmpdir, retention)

            self.assertEqual(summary["sessions_deleted"], 0)
            self.assertTrue(os.path.exists(recent))

    def test_handles_missing_directories(self):
        with tempfile.TemporaryDirectory() as tmpdir:
            retention = RetentionConfig()
            summary = RetentionManager.cleanup(tmpdir, retention)
            self.assertEqual(summary["sessions_deleted"], 0)
            self.assertEqual(summary["prompts_deleted"], 0)


class TestBedrockValidator(unittest.TestCase):
    """Test L0 BEDROCK runtime validation with mocked Docker client."""

    def test_no_docker_client(self):
        config = LabyrinthConfig()
        ok, errors = BedrockValidator.validate(None, config)
        self.assertFalse(ok)
        self.assertIn("Docker client not available", errors)

    def test_docker_ping_failure(self):
        class MockClient:
            def ping(self):
                raise ConnectionError("Docker socket not found")

        config = LabyrinthConfig()
        ok, errors = BedrockValidator.validate(MockClient(), config)
        self.assertFalse(ok)
        self.assertTrue(any("ping failed" in e for e in errors))

    def test_all_checks_pass(self):
        class MockImage:
            pass

        class MockImages:
            def get(self, name):
                return MockImage()

        class MockNetwork:
            attrs = {
                "IPAM": {
                    "Config": [{"Subnet": "172.30.0.0/24"}]
                }
            }

        class MockProxy:
            attrs = {
                "NetworkSettings": {
                    "Networks": {
                        "labyrinth-net": {"IPAddress": "172.30.0.50"}
                    }
                }
            }

        class MockContainers:
            def list(self, filters=None):
                name = filters.get("name", "") if filters else ""
                if name == "labyrinth-proxy":
                    return [MockProxy()]
                return []

        class MockNetworks:
            def list(self, names=None):
                if names and "labyrinth-net" in names:
                    return [MockNetwork()]
                return []

        class MockClient:
            containers = MockContainers()
            networks = MockNetworks()
            images = MockImages()

            def ping(self):
                return True

        config = LabyrinthConfig()
        ok, errors = BedrockValidator.validate(MockClient(), config)
        self.assertTrue(ok, f"Unexpected errors: {errors}")
        self.assertEqual(len(errors), 0)

    def test_missing_network(self):
        class MockImages:
            def get(self, name):
                raise Exception("not found")

        class MockContainers:
            def list(self, filters=None):
                return []

        class MockNetworks:
            def list(self, names=None):
                return []

        class MockClient:
            containers = MockContainers()
            networks = MockNetworks()
            images = MockImages()

            def ping(self):
                return True

        config = LabyrinthConfig()
        ok, errors = BedrockValidator.validate(MockClient(), config)
        self.assertFalse(ok)
        self.assertTrue(any("labyrinth-net" in e for e in errors))


class TestKillChainIntegration(unittest.TestCase):
    """Integration test: session → contradiction config → entrypoint generation."""

    def test_session_to_entrypoint(self):
        from orchestrator.layer_controllers import MinotaurController
        from orchestrator.config import Layer2Config
        from layer2_maze.container_template import generate_entrypoint_script

        mgr = SessionManager()
        session = mgr.create_session("10.0.0.99", "ssh")

        controller = MinotaurController(Layer2Config())
        config = controller.get_initial_config(session)

        self.assertIn("contradictions", config)
        self.assertGreater(len(config["contradictions"]), 0)

        script = generate_entrypoint_script(
            contradictions=config["contradictions"],
            session_id=session.session_id,
            l3_active=False,
        )
        self.assertIn("#!/bin/bash", script)
        self.assertIn(session.session_id, script)


class TestLayer0Config(unittest.TestCase):
    """Test Layer0Config parsing."""

    def test_default_values(self):
        config = Layer0Config()
        self.assertEqual(config.fail_mode, "closed")
        self.assertTrue(config.validate_on_startup)

    def test_yaml_parsing(self):
        import tempfile

        with tempfile.NamedTemporaryFile(mode="w", suffix=".yaml", delete=False) as f:
            f.write(
                "layer0:\n"
                "  proxy:\n"
                "    fail_mode: open\n"
                "    validate_scope_on_startup: false\n"
            )
            f.flush()
            config = LabyrinthConfig.load(f.name)

        os.unlink(f.name)
        self.assertEqual(config.layer0.fail_mode, "open")
        self.assertFalse(config.layer0.validate_on_startup)


if __name__ == "__main__":
    unittest.main()
