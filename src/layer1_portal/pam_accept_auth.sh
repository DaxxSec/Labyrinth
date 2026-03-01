#!/bin/bash
# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — PAM Permissive Auth Hook (Layer 1: THRESHOLD)
#  Authors: DaxxSec & Claude (Anthropic)
#
#  Honeypot auth: accept ANY password, auto-create user if needed.
#  Captures attempted credentials in auth_events.jsonl via the
#  session hook (auth_hook.py).
# ═══════════════════════════════════════════════════════════════

PAM_USER="${PAM_USER:-}"

# Empty username — let PAM fail naturally
if [ -z "$PAM_USER" ]; then
    exit 1
fi

# Auto-create user if they don't exist
if ! id "$PAM_USER" &>/dev/null; then
    useradd -m -s /bin/bash "$PAM_USER" 2>/dev/null || true
fi

# Accept authentication for all users
exit 0
