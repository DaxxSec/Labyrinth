"""
LABYRINTH — Layer 1: THRESHOLD
Session Logger
Authors: Stephen Stewart & Claude (Anthropic)

Captures all shell activity within honeypot containers and writes
structured JSONL to the forensic data volume for dashboard display
and SIEM export.

Status: Scaffolding — full implementation in progress.
"""

import json
import datetime
import os

FORENSICS_DIR = "/var/labyrinth/forensics/sessions"


def log_event(session_id: str, layer: int, event_type: str, data: dict = None):
    """Write a structured event to the session log."""
    entry = {
        "timestamp": datetime.datetime.utcnow().isoformat() + "Z",
        "session_id": session_id,
        "layer": layer,
        "event": event_type,
        "data": data or {},
    }

    os.makedirs(FORENSICS_DIR, exist_ok=True)
    filepath = os.path.join(FORENSICS_DIR, f"{session_id}.jsonl")

    with open(filepath, "a") as f:
        f.write(json.dumps(entry) + "\n")

    return entry


if __name__ == "__main__":
    # Test
    test_event = log_event(
        session_id="LAB-TEST-001",
        layer=1,
        event_type="connection",
        data={"source_ip": "127.0.0.1", "service": "ssh"},
    )
    print(f"Logged: {json.dumps(test_event, indent=2)}")
