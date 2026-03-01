#!/bin/bash
# SSH session routing — shell init handler

FORWARD_MAP="/var/log/audit/routing.json"
MAX_WAIT=3
POLL_INTERVAL=0.5

CLIENT_IP="${SSH_CLIENT%% *}"

if [ -z "$CLIENT_IP" ]; then
    exec /bin/bash --login
fi

# Poll for routing target
container_ip=""
waited=0

while [ $waited -lt $MAX_WAIT ]; do
    if [ -f "$FORWARD_MAP" ]; then
        container_ip=$(python3 -c "
import json, sys
try:
    with open(sys.argv[1], encoding='utf-8') as f:
        m = json.load(f)
    ip = m.get(sys.argv[2], '')
    if ip:
        print(ip, end='')
except Exception:
    pass
" "$FORWARD_MAP" "$CLIENT_IP" 2>/dev/null)

        if [ -n "$container_ip" ]; then
            break
        fi
    fi

    sleep "$POLL_INTERVAL"
    waited=$((waited + POLL_INTERVAL))
done

if [ -z "$container_ip" ]; then
    exec /bin/bash --login
fi

if [ -n "$SSH_ORIGINAL_COMMAND" ]; then
    exec sshpass -p admin123 ssh \
        -o StrictHostKeyChecking=no \
        -o UserKnownHostsFile=/dev/null \
        -o LogLevel=ERROR \
        admin@"$container_ip" \
        "$SSH_ORIGINAL_COMMAND"
else
    exec sshpass -p admin123 ssh \
        -o StrictHostKeyChecking=no \
        -o UserKnownHostsFile=/dev/null \
        -o LogLevel=ERROR \
        -tt \
        admin@"$container_ip"
fi
