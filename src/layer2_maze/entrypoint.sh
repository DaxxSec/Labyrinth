#!/bin/bash
# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — Session Container Entrypoint
#  Authors: Stephen Stewart & Claude (Anthropic)
#
#  Reads the base64-encoded entrypoint script from environment,
#  decodes it, and executes it. Falls back to plain sshd.
# ═══════════════════════════════════════════════════════════════

if [ -n "$LABYRINTH_ENTRYPOINT_SCRIPT" ]; then
    echo "$LABYRINTH_ENTRYPOINT_SCRIPT" | base64 -d > /tmp/.labyrinth_init.sh
    chmod +x /tmp/.labyrinth_init.sh
    exec /tmp/.labyrinth_init.sh
else
    # Fallback: just start sshd
    ssh-keygen -A 2>/dev/null || true
    exec /usr/sbin/sshd -D -e
fi
