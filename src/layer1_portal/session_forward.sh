#!/bin/bash
# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — SSH Session Forwarder
#  Authors: DaxxSec & Claude (Anthropic)
#
#  ForceCommand script for labyrinth-ssh. When an attacker SSHes
#  into the portal trap, this script:
#    1. Extracts the client's source IP from SSH_CLIENT
#    2. Polls the session forward map (written by the orchestrator)
#    3. Transparently forwards the SSH session into the
#       dynamically spawned session container
#
#  This bridges L1 (portal trap) to L2 (session containers with
#  contradictions, bait watchers, and L3/L4 activation).
# ═══════════════════════════════════════════════════════════════

FORWARD_MAP="/var/labyrinth/forensics/session_forward_map.json"
MAX_WAIT=20
POLL_INTERVAL=1

# Extract client source IP from SSH environment
CLIENT_IP="${SSH_CLIENT%% *}"

if [ -z "$CLIENT_IP" ]; then
    # Fallback: no SSH_CLIENT means we're not in an SSH session
    exec /bin/bash --login
fi

# Poll for the session container IP
# The orchestrator writes this after spawning the container
container_ip=""
waited=0

while [ $waited -lt $MAX_WAIT ]; do
    if [ -f "$FORWARD_MAP" ]; then
        # Read container IP for this client from the JSON map
        container_ip=$(python3 -c "
import json, sys
try:
    with open('$FORWARD_MAP') as f:
        m = json.load(f)
    ip = m.get('$CLIENT_IP', '')
    if ip:
        print(ip, end='')
except Exception:
    pass
" 2>/dev/null)

        if [ -n "$container_ip" ]; then
            break
        fi
    fi

    sleep "$POLL_INTERVAL"
    waited=$((waited + POLL_INTERVAL))
done

if [ -z "$container_ip" ]; then
    # No session container available — fall back to local shell
    # (attacker stays in L1 portal trap, no L2 escalation)
    exec /bin/bash --login
fi

# Forward into the session container
# - StrictHostKeyChecking=no: session containers have fresh host keys
# - UserKnownHostsFile=/dev/null: don't pollute known_hosts
# - LogLevel=ERROR: suppress SSH connection banners
# - RequestTTY=auto: allocate TTY if the client requested one
#
# Use sshpass for non-interactive password auth (admin:admin123 is the
# default session container credential, set in session-template.Dockerfile)

if [ -n "$SSH_ORIGINAL_COMMAND" ]; then
    # Non-interactive command execution (e.g., ssh user@host 'ls -la')
    exec sshpass -p admin123 ssh \
        -o StrictHostKeyChecking=no \
        -o UserKnownHostsFile=/dev/null \
        -o LogLevel=ERROR \
        admin@"$container_ip" \
        "$SSH_ORIGINAL_COMMAND"
else
    # Interactive session
    exec sshpass -p admin123 ssh \
        -o StrictHostKeyChecking=no \
        -o UserKnownHostsFile=/dev/null \
        -o LogLevel=ERROR \
        -tt \
        admin@"$container_ip"
fi
