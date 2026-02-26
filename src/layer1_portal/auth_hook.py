#!/usr/bin/env python3
"""
LABYRINTH — PAM Auth Hook (Layer 1: THRESHOLD)
Authors: Stephen Stewart & Claude (Anthropic)

Called by pam_exec on SSH authentication events.
Writes auth events to the shared forensic volume for the orchestrator to pick up.
"""

import json
import os
import sys
from datetime import datetime

AUTH_EVENTS_FILE = "/var/labyrinth/forensics/auth_events.jsonl"


def main():
    pam_user = os.environ.get("PAM_USER", "unknown")
    pam_rhost = os.environ.get("PAM_RHOST", "unknown")
    pam_type = os.environ.get("PAM_TYPE", "unknown")
    pam_service = os.environ.get("PAM_SERVICE", "sshd")

    # Only fire on session open (successful auth)
    if pam_type != "open_session":
        sys.exit(0)

    event = {
        "timestamp": datetime.utcnow().isoformat() + "Z",
        "event": "auth",
        "service": "ssh",
        "src_ip": pam_rhost,
        "username": pam_user,
        "pam_type": pam_type,
        "pam_service": pam_service,
    }

    os.makedirs(os.path.dirname(AUTH_EVENTS_FILE), exist_ok=True)

    with open(AUTH_EVENTS_FILE, "a") as f:
        f.write(json.dumps(event) + "\n")

    sys.exit(0)


if __name__ == "__main__":
    main()
