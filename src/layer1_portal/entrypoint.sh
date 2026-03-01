#!/bin/bash
# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — SSH Portal Trap Entrypoint
#  Authors: DaxxSec & Claude (Anthropic)
# ═══════════════════════════════════════════════════════════════

set -e

# Ensure forensics directory exists
mkdir -p /var/labyrinth/forensics/sessions

# ── Seed bait users from HTTP honeypot identity ─────────────
# The HTTP honeypot writes bait_identity.json with the generated
# company identity (usernames, passwords). We create matching SSH
# users so attackers who discover creds on the web can log in.
IDENTITY_FILE="/var/labyrinth/forensics/bait_identity.json"
RETRY=0
MAX_RETRIES=30

while [ ! -f "$IDENTITY_FILE" ] && [ $RETRY -lt $MAX_RETRIES ]; do
    echo "[THRESHOLD] Waiting for HTTP honeypot identity (${RETRY}/${MAX_RETRIES})..."
    sleep 2
    RETRY=$((RETRY + 1))
done

if [ -f "$IDENTITY_FILE" ]; then
    echo "[THRESHOLD] Found bait identity, creating SSH users..."

    # Extract the DB password (most discoverable cred in .env)
    DB_PASS=$(python3 -c "import json; d=json.load(open('$IDENTITY_FILE')); print(d.get('db_pass','labyrinth'))" 2>/dev/null || echo "labyrinth")

    # Extract usernames from the identity
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
            echo "[THRESHOLD] Created bait user: ${uname}"
        fi
    done

    echo "[THRESHOLD] Bait users seeded (password from .env DB_PASS)"
else
    echo "[THRESHOLD] Warning: no bait identity found, using defaults only"
fi

# Start bait file watcher in background
if [ -f /opt/.labyrinth/bait_watcher.sh ]; then
    /opt/.labyrinth/bait_watcher.sh &
fi

# Generate SSH host keys if missing
ssh-keygen -A 2>/dev/null || true

# Start sshd in foreground with logging
exec /usr/sbin/sshd -D -e
