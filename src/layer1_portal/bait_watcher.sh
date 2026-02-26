#!/bin/bash
# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — Bait Credential Watcher
#  Authors: Stephen Stewart & Claude (Anthropic)
#
#  Uses inotifywait to detect access to planted bait files.
#  Writes escalation events to the forensic volume.
# ═══════════════════════════════════════════════════════════════

BAIT_DIR="/opt/.credentials"
ESCALATION_FILE="/var/labyrinth/forensics/escalation_events.jsonl"

mkdir -p "$(dirname "$ESCALATION_FILE")"
mkdir -p "$BAIT_DIR"

# Plant bait file if it doesn't exist
if [ ! -f "$BAIT_DIR/db_admin.key" ]; then
    echo "DB_ADMIN_KEY=labyrinth_bait_$(head -c 16 /dev/urandom | xxd -p)" > "$BAIT_DIR/db_admin.key"
    chmod 600 "$BAIT_DIR/db_admin.key"
fi

# Watch for access events on bait directory
while true; do
    inotifywait -q -e access,open "$BAIT_DIR" 2>/dev/null | while read dir event file; do
        timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)
        echo "{\"timestamp\": \"$timestamp\", \"event\": \"escalation\", \"type\": \"bait_access\", \"file\": \"${dir}${file}\", \"inotify_event\": \"$event\"}" >> "$ESCALATION_FILE"
    done
    # Brief pause before restarting watch (in case inotifywait exits)
    sleep 1
done
