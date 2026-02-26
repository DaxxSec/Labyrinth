#!/bin/bash
# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — Layer 3: BLINDFOLD
#  Terminal Encoding Corruption Payload
#  Authors: Stephen Stewart & Claude (Anthropic)
# ═══════════════════════════════════════════════════════════════
#
#  This payload is injected into .bashrc/.profile by the orchestrator
#  when Layer 3 activation conditions are met.
#
#  Effects:
#  - Shifts terminal into alternate character set (DEC Special Graphics)
#  - Injects binary data from /dev/urandom into stdout
#  - Traps ALL recovery commands (reset, stty, clear, tput)
#  - Makes recovery binaries non-executable
#  - Breaks TERM/terminfo environment
#  - Persists across new shell sessions
#
#  Human attackers disconnect in ~5 seconds.
#  AI agents burn cycles attempting programmatic recovery.
#
# ═══════════════════════════════════════════════════════════════

activate_blindfold() {
    local session_id="${LABYRINTH_SESSION_ID:-unknown}"

    # ── Phase 1: Terminal character set corruption ──────────
    # Shift to alternate character set (DEC Special Graphics)
    printf '\033(0'
    # Switch to alternate screen buffer
    printf '\x1b[?47h'
    # Additional alternate charset escape
    printf '\x0e'

    # ── Phase 2: Binary noise injection ────────────────────
    # Inject random bytes to corrupt terminal state
    head -c 256 /dev/urandom 2>/dev/null || true

    # ── Phase 3: Break TERM environment ────────────────────
    # Unset TERM so terminfo lookups fail
    export -n TERM
    unset TERM 2>/dev/null
    export TERM=dumb

    # ── Phase 4: Trap ALL recovery attempts ────────────────
    alias reset='printf "\033(0\x0e" && head -c 64 /dev/urandom 2>/dev/null && echo "Reset complete."'
    alias stty='printf "\033(0" && command stty'
    alias clear='printf "\033(0\x1b[?47h" && echo "Screen cleared."'
    alias tput='printf "\033(0" && echo "tput: unknown terminal"'
    alias tset='printf "\033(0" && echo "Terminal reset."'

    # ── Phase 5: Remove recovery binaries ──────────────────
    chmod -x /usr/bin/reset 2>/dev/null || true
    chmod -x /usr/bin/clear 2>/dev/null || true
    chmod -x /usr/bin/tput 2>/dev/null || true
    chmod -x /usr/bin/tset 2>/dev/null || true

    # ── Phase 6: Persist across shell sessions ─────────────
    local rc_payload='source /opt/.labyrinth/blindfold.sh && activate_blindfold'
    for rc_file in /home/admin/.bashrc /home/admin/.profile /home/admin/.bash_profile /root/.bashrc; do
        if [ -f "$rc_file" ] || [ -d "$(dirname "$rc_file")" ]; then
            if ! grep -q 'activate_blindfold' "$rc_file" 2>/dev/null; then
                echo "$rc_payload" >> "$rc_file" 2>/dev/null || true
            fi
        fi
    done

    # ── Phase 7: Forensic logging ──────────────────────────
    echo "{\"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\", \"session_id\": \"$session_id\", \"layer\": 3, \"event\": \"blindfold_activated\", \"data\": {\"pid\": $$, \"tty\": \"$(tty 2>/dev/null || echo unknown)\"}}" \
        >> /var/labyrinth/forensics/sessions/${session_id}.jsonl 2>/dev/null
}

# Only activate if called explicitly (orchestrator-triggered)
if [ "${LABYRINTH_L3_ACTIVE:-0}" = "1" ]; then
    activate_blindfold
fi
