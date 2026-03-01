#!/bin/bash
# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — PAM Auth Hook (Layer 1: THRESHOLD)
#  Authors: DaxxSec & Claude (Anthropic)
#
#  Validates SSH credentials against the bait identity planted
#  by the HTTP honeypot. Only accepts known bait passwords —
#  rejecting random guesses like a real server would.
#  Auto-creates bait users on first login.
# ═══════════════════════════════════════════════════════════════

PAM_USER="${PAM_USER:-}"
PAM_AUTHTOK="${PAM_AUTHTOK:-}"
IDENTITY_FILE="/var/labyrinth/forensics/bait_identity.json"

# Empty username — reject
if [ -z "$PAM_USER" ]; then
    exit 1
fi

# Let hardcoded admin through (validated by /etc/shadow via fallback)
if [ "$PAM_USER" = "admin" ] || [ "$PAM_USER" = "root" ]; then
    exit 1  # fall through to common-auth
fi

# No identity file — reject non-system users
if [ ! -f "$IDENTITY_FILE" ]; then
    exit 1
fi

# Validate password against bait identity using Python
exec /usr/bin/python3 -c "
import json, sys, os

user = os.environ.get('PAM_USER', '')
password = os.environ.get('PAM_AUTHTOK', '')
if not user or not password:
    sys.exit(1)

try:
    with open('$IDENTITY_FILE') as f:
        identity = json.load(f)
except:
    sys.exit(1)

# Check if this is a known bait user
bait_usernames = [u.get('uname', '') for u in identity.get('users', [])]
db_pass = identity.get('db_pass', '')

# Accept if: known bait user with the planted password (db_pass)
if user in bait_usernames and password == db_pass:
    # Auto-create if needed
    import subprocess
    try:
        subprocess.run(['id', user], capture_output=True, check=True)
    except subprocess.CalledProcessError:
        subprocess.run(['useradd', '-m', '-s', '/bin/bash', user],
                       capture_output=True)
    sys.exit(0)

# Reject — wrong password or unknown user
sys.exit(1)
"
