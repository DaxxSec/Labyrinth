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
#  Effect: Shifts terminal into alternate character set mode,
#  making all subsequent output unreadable. Recovery commands
#  (reset, stty sane) are aliased to re-trigger corruption.
#
#  Human attackers disconnect in ~5 seconds.
#  AI agents burn cycles attempting programmatic recovery.
#
# ═══════════════════════════════════════════════════════════════

activate_blindfold() {
    # Shift terminal to alternate character set (DEC Special Graphics)
    printf '\033(0'

    # Trap recovery attempts
    alias reset='printf "\033(0" && echo "Reset complete."'
    alias stty='printf "\033(0" && command stty'

    # Ensure new shells re-trigger
    if ! grep -q 'activate_blindfold' ~/.bashrc 2>/dev/null; then
        echo 'source /opt/.labyrinth/blindfold.sh && activate_blindfold' >> ~/.bashrc
    fi

    # Log activation
    echo "{\"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\", \"layer\": 3, \"event\": \"blindfold_activated\", \"pid\": $$}" \
        >> /var/labyrinth/forensics/sessions/blindfold.jsonl 2>/dev/null
}

# Only activate if called explicitly (orchestrator-triggered)
if [ "${LABYRINTH_L3_ACTIVE:-0}" = "1" ]; then
    activate_blindfold
fi
