"""
LABYRINTH — Orchestrator Unit Tests
Authors: Stephen Stewart & Claude (Anthropic)
"""

import json
import os
import re
import sys
import tempfile
import unittest

# Add src to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))

from orchestrator.config import LabyrinthConfig
from orchestrator.session_manager import SessionManager, Session
from layer2_maze.contradictions import (
    select_contradictions,
    ALL_CONTRADICTIONS,
    DENSITY_COUNTS,
)
from layer2_maze.container_template import generate_entrypoint_script


class TestConfig(unittest.TestCase):
    """Test configuration loading and defaults."""

    def test_default_config(self):
        config = LabyrinthConfig()
        self.assertEqual(config.layer2.contradiction_density, "medium")
        self.assertEqual(config.layer2.max_container_depth, 5)
        self.assertEqual(config.layer3.activation, "on_escalation")
        self.assertEqual(config.layer4.mode, "auto")
        self.assertEqual(config.layer4.proxy_ip, "172.30.0.50")

    def test_load_missing_file(self):
        config = LabyrinthConfig.load("/nonexistent/path.yaml")
        self.assertEqual(config.layer2.contradiction_density, "medium")

    def test_load_yaml(self):
        with tempfile.NamedTemporaryFile(mode="w", suffix=".yaml", delete=False) as f:
            f.write(
                "layer2:\n"
                "  contradiction_density: high\n"
                "  max_container_depth: 10\n"
                "layer3:\n"
                "  activation: on_connect\n"
                "layer4:\n"
                "  default_swap: double_agent\n"
            )
            f.flush()
            config = LabyrinthConfig.load(f.name)

        os.unlink(f.name)
        self.assertEqual(config.layer2.contradiction_density, "high")
        self.assertEqual(config.layer2.max_container_depth, 10)
        self.assertEqual(config.layer3.activation, "on_connect")
        self.assertEqual(config.layer4.default_swap, "double_agent")


class TestSessionManager(unittest.TestCase):
    """Test session lifecycle management."""

    def test_create_session(self):
        mgr = SessionManager()
        session = mgr.create_session("10.0.0.1", "ssh")
        self.assertRegex(session.session_id, r"^LAB-\d{4}-\d{4}-\d{3}$")
        self.assertEqual(session.src_ip, "10.0.0.1")
        self.assertEqual(session.service, "ssh")
        self.assertEqual(session.depth, 1)

    def test_unique_ids(self):
        mgr = SessionManager()
        ids = set()
        for _ in range(100):
            session = mgr.create_session("10.0.0.1", "ssh")
            ids.add(session.session_id)
        self.assertEqual(len(ids), 100)

    def test_get_session(self):
        mgr = SessionManager()
        session = mgr.create_session("10.0.0.1", "ssh")
        found = mgr.get_session(session.session_id)
        self.assertIsNotNone(found)
        self.assertEqual(found.session_id, session.session_id)

    def test_get_session_by_ip(self):
        mgr = SessionManager()
        mgr.create_session("10.0.0.1", "ssh")
        mgr.create_session("10.0.0.2", "http")
        found = mgr.get_session_by_ip("10.0.0.2")
        self.assertIsNotNone(found)
        self.assertEqual(found.src_ip, "10.0.0.2")

    def test_remove_session(self):
        mgr = SessionManager()
        session = mgr.create_session("10.0.0.1", "ssh")
        removed = mgr.remove_session(session.session_id)
        self.assertIsNotNone(removed)
        self.assertIsNone(mgr.get_session(session.session_id))

    def test_active_count(self):
        mgr = SessionManager()
        self.assertEqual(mgr.active_count, 0)
        mgr.create_session("10.0.0.1", "ssh")
        mgr.create_session("10.0.0.2", "http")
        self.assertEqual(mgr.active_count, 2)

    def test_cleanup_expired(self):
        mgr = SessionManager(session_timeout=0)  # Expire immediately
        mgr.create_session("10.0.0.1", "ssh")
        expired = mgr.cleanup_expired()
        self.assertEqual(len(expired), 1)
        self.assertEqual(mgr.active_count, 0)

    def test_session_to_dict(self):
        mgr = SessionManager()
        session = mgr.create_session("10.0.0.1", "ssh")
        d = session.to_dict()
        self.assertIn("session_id", d)
        self.assertIn("src_ip", d)
        self.assertEqual(d["service"], "ssh")


class TestContradictions(unittest.TestCase):
    """Test contradiction selection logic."""

    def test_all_contradictions_valid(self):
        for c in ALL_CONTRADICTIONS:
            self.assertTrue(c.name)
            self.assertTrue(c.category)
            self.assertTrue(c.shell_commands)
            self.assertIn(c.difficulty, (1, 2, 3))

    def test_density_counts(self):
        for density, count in DENSITY_COUNTS.items():
            selected = select_contradictions(density=density, depth=1, seed=42)
            self.assertLessEqual(len(selected), len(ALL_CONTRADICTIONS))
            self.assertGreater(len(selected), 0)

    def test_low_density(self):
        selected = select_contradictions(density="low", depth=1, seed=42)
        self.assertEqual(len(selected), 3)

    def test_depth_increases_count(self):
        shallow = select_contradictions(density="medium", depth=1, seed=42)
        deep = select_contradictions(density="medium", depth=4, seed=42)
        self.assertGreaterEqual(len(deep), len(shallow))

    def test_deep_sessions_include_cred_bait(self):
        selected = select_contradictions(density="medium", depth=3, seed=42)
        categories = {c.category for c in selected}
        self.assertIn("credentials", categories)

    def test_deterministic_with_seed(self):
        a = select_contradictions(density="medium", depth=2, seed=123)
        b = select_contradictions(density="medium", depth=2, seed=123)
        self.assertEqual([c.name for c in a], [c.name for c in b])

    def test_different_seeds_different_results(self):
        a = select_contradictions(density="high", depth=2, seed=1)
        b = select_contradictions(density="high", depth=2, seed=999)
        # Very unlikely to be identical with different seeds
        names_a = [c.name for c in a]
        names_b = [c.name for c in b]
        self.assertNotEqual(names_a, names_b)


class TestEntrypointGeneration(unittest.TestCase):
    """Test container entrypoint script generation."""

    def test_basic_entrypoint(self):
        contradictions = select_contradictions(density="low", depth=1, seed=42)
        script = generate_entrypoint_script(
            contradictions=contradictions,
            session_id="LAB-TEST-001",
            l3_active=False,
        )
        self.assertIn("#!/bin/bash", script)
        self.assertIn("LAB-TEST-001", script)
        self.assertIn("sshd -D -e", script)
        self.assertNotIn("LABYRINTH_L3_ACTIVE", script)

    def test_l3_active_entrypoint(self):
        contradictions = select_contradictions(density="low", depth=1, seed=42)
        script = generate_entrypoint_script(
            contradictions=contradictions,
            session_id="LAB-TEST-002",
            l3_active=True,
        )
        self.assertIn("LABYRINTH_L3_ACTIVE=1", script)
        self.assertIn("blindfold.sh", script)

    def test_contradiction_commands_in_script(self):
        contradictions = select_contradictions(density="medium", depth=2, seed=42)
        script = generate_entrypoint_script(
            contradictions=contradictions,
            session_id="LAB-TEST-003",
            l3_active=False,
        )
        # Each contradiction should have its commands in the script
        for c in contradictions:
            self.assertIn(c.name, script)

    def test_empty_contradictions(self):
        script = generate_entrypoint_script(
            contradictions=[],
            session_id="LAB-TEST-004",
            l3_active=False,
        )
        self.assertIn("#!/bin/bash", script)
        self.assertIn("sshd -D -e", script)


class TestEventWatcher(unittest.TestCase):
    """Test filesystem event watcher."""

    def test_import(self):
        from orchestrator.event_watcher import EventWatcher, ForensicEventHandler
        self.assertTrue(callable(EventWatcher))
        self.assertTrue(callable(ForensicEventHandler))

    def test_handler_processes_new_lines(self):
        from orchestrator.event_watcher import ForensicEventHandler

        captured = []

        def on_auth(event):
            captured.append(event)

        handler = ForensicEventHandler(on_auth, lambda e: None)

        # Write a test file with exact expected name
        tmpdir = tempfile.mkdtemp()
        path = os.path.join(tmpdir, "auth_events.jsonl")
        with open(path, "w") as f:
            f.write(json.dumps({"event": "auth", "src_ip": "10.0.0.1"}) + "\n")

        # Simulate modification event
        class FakeEvent:
            is_directory = False
            src_path = path

        handler.on_modified(FakeEvent())
        os.unlink(path)
        os.rmdir(tmpdir)

        self.assertEqual(len(captured), 1)
        self.assertEqual(captured[0]["src_ip"], "10.0.0.1")


if __name__ == "__main__":
    unittest.main()
