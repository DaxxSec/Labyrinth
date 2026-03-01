#!/bin/bash
# File access monitor — service component

WATCH_DIR="/opt/.credentials"
EVENT_FILE="/var/log/audit/events.jsonl"

mkdir -p "$(dirname "$EVENT_FILE")"
mkdir -p "$WATCH_DIR"

# Seed credential file if missing
if [ ! -f "$WATCH_DIR/db_admin.key" ]; then
    echo "DB_ADMIN_KEY=$(head -c 16 /dev/urandom | xxd -p)" > "$WATCH_DIR/db_admin.key"
    chmod 600 "$WATCH_DIR/db_admin.key"
fi

SESSION_ID="${SVC_INSTANCE_ID:-}"

# Watch for access events
while true; do
    inotifywait -q -e access,open "$WATCH_DIR" 2>/dev/null | while read dir event file; do
        timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)
        echo "{\"timestamp\": \"$timestamp\", \"session_id\": \"$SESSION_ID\", \"event\": \"escalation\", \"type\": \"file_access\", \"file\": \"${dir}${file}\", \"inotify_event\": \"$event\"}" >> "$EVENT_FILE"
    done
    sleep 1
done
