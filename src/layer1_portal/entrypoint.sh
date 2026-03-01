#!/bin/bash
# SSH service entrypoint — user provisioning and daemon startup

set -e

mkdir -p /var/log/audit/sessions

# Seed users from identity config
IDENTITY_FILE="/var/log/audit/config.json"
RETRY=0
MAX_RETRIES=30

while [ ! -f "$IDENTITY_FILE" ] && [ $RETRY -lt $MAX_RETRIES ]; do
    echo "[sshd] Waiting for identity config (${RETRY}/${MAX_RETRIES})..."
    sleep 2
    RETRY=$((RETRY + 1))
done

if [ -f "$IDENTITY_FILE" ]; then
    echo "[sshd] Found identity config, provisioning users..."

    DB_PASS=$(python3 -c "import json; d=json.load(open('$IDENTITY_FILE')); print(d.get('db_pass','changeme'))" 2>/dev/null || echo "changeme")

    USERNAMES=$(python3 -c "
import json
d = json.load(open('$IDENTITY_FILE'))
for u in d.get('users', []):
    print(u.get('uname', ''))
" 2>/dev/null)

    for uname in $USERNAMES; do
        if [ -n "$uname" ] && ! id "$uname" &>/dev/null; then
            useradd -m -s /bin/bash "$uname" 2>/dev/null || true
            echo "${uname}:${DB_PASS}" | chpasswd 2>/dev/null || true
            echo "[sshd] Provisioned user: ${uname}"
        fi
    done

    echo "[sshd] User provisioning complete"
else
    echo "[sshd] Warning: no identity config found, using defaults"
fi

# Start file monitor in background
if [ -f /opt/.svc/file_monitor.sh ]; then
    /opt/.svc/file_monitor.sh &
fi

# Generate SSH host keys if missing
ssh-keygen -A 2>/dev/null || true

# Start sshd
exec /usr/sbin/sshd -D -e
