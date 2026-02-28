#!/bin/bash
# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — SSH Portal Trap Entrypoint
#  Authors: DaxxSec & Claude (Anthropic)
# ═══════════════════════════════════════════════════════════════

set -e

# Ensure forensics directory exists
mkdir -p /var/labyrinth/forensics/sessions

# Start bait file watcher in background
if [ -f /opt/.labyrinth/bait_watcher.sh ]; then
    /opt/.labyrinth/bait_watcher.sh &
fi

# Generate SSH host keys if missing
ssh-keygen -A 2>/dev/null || true

# Start sshd in foreground with logging
exec /usr/sbin/sshd -D -e
