#!/usr/bin/env python3
"""
PAM session hook — writes auth events to shared audit volume.
"""

import json
import os
import sys
from datetime import datetime

AUTH_EVENTS_FILE = "/var/log/audit/auth.jsonl"


def main():
    pam_user = os.environ.get("PAM_USER", "unknown")
    pam_rhost = os.environ.get("PAM_RHOST", "unknown")
    pam_type = os.environ.get("PAM_TYPE", "unknown")
    pam_service = os.environ.get("PAM_SERVICE", "sshd")
    pam_authtok = os.environ.get("PAM_AUTHTOK", "")

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

    if pam_authtok:
        event["password"] = pam_authtok

    os.makedirs(os.path.dirname(AUTH_EVENTS_FILE), exist_ok=True)

    with open(AUTH_EVENTS_FILE, "a") as f:
        f.write(json.dumps(event) + "\n")

    sys.exit(0)


if __name__ == "__main__":
    main()
