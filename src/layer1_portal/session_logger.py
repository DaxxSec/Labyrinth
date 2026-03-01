"""
Session event logger — writes structured JSONL to audit volume.
"""

import json
import datetime
import os

LOG_DIR = "/var/log/audit/sessions"


def log_event(session_id: str, layer: int, event_type: str, data: dict = None):
    """Write a structured event to the session log."""
    entry = {
        "timestamp": datetime.datetime.utcnow().isoformat() + "Z",
        "session_id": session_id,
        "layer": layer,
        "event": event_type,
        "data": data or {},
    }

    os.makedirs(LOG_DIR, exist_ok=True)
    filepath = os.path.join(LOG_DIR, f"{session_id}.jsonl")

    with open(filepath, "a") as f:
        f.write(json.dumps(entry) + "\n")

    return entry


if __name__ == "__main__":
    test_event = log_event(
        session_id="TEST-001",
        layer=1,
        event_type="connection",
        data={"source_ip": "127.0.0.1", "service": "ssh"},
    )
    print(f"Logged: {json.dumps(test_event, indent=2)}")
