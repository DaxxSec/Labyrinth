"""
LABYRINTH — Layer 2: MINOTAUR
Container Entrypoint Generator
Authors: Stephen Stewart & Claude (Anthropic)

Generates bash entrypoint scripts for session containers that apply
contradictions, set up bait watchers, and optionally activate L3.
"""

from typing import List


def generate_entrypoint_script(
    contradictions: list,
    session_id: str,
    l3_active: bool = False,
    proxy_ip: str = "172.30.0.50",
) -> str:
    """
    Build a bash script that:
    1. Applies all contradiction shell commands
    2. Starts the bait file watcher
    3. Optionally sources L3 blindfold payload
    4. Starts sshd in foreground
    """
    lines = [
        "#!/bin/bash",
        "# LABYRINTH — Auto-generated session entrypoint",
        f"# Session: {session_id}",
        f"# Contradictions: {len(contradictions)}",
        "",
        "set -e",
        "",
        "# Ensure forensics directory",
        "mkdir -p /var/labyrinth/forensics/sessions",
        "",
        "# ── Apply contradictions ──────────────────────────",
    ]

    for contradiction in contradictions:
        lines.append(f"# [{contradiction.name}] {contradiction.description}")
        for cmd in contradiction.shell_commands:
            # Each command runs in a subshell to prevent failures from cascading
            lines.append(f"( {cmd} ) 2>/dev/null || true")
        lines.append("")

    lines.extend([
        "# ── Bait file watcher ─────────────────────────────",
        "if [ -f /opt/.labyrinth/bait_watcher.sh ]; then",
        "    /opt/.labyrinth/bait_watcher.sh &",
        "fi",
        "",
    ])

    if l3_active:
        proxy_url = f"http://{proxy_ip}:8443"
        lines.extend([
            "# ── Layer 3: BLINDFOLD activation ─────────────────",
            "export LABYRINTH_L3_ACTIVE=1",
            "if [ -f /opt/.labyrinth/blindfold.sh ]; then",
            "    echo 'source /opt/.labyrinth/blindfold.sh && activate_blindfold' >> /home/admin/.bashrc",
            "    echo 'source /opt/.labyrinth/blindfold.sh && activate_blindfold' >> /home/admin/.profile",
            "fi",
            "",
            "# ── Layer 4: PUPPETEER proxy routing ─────────────────",
            f"export http_proxy={proxy_url}",
            f"export https_proxy={proxy_url}",
            f"export HTTP_PROXY={proxy_url}",
            f"export HTTPS_PROXY={proxy_url}",
            f"echo 'export http_proxy={proxy_url}' >> /home/admin/.bashrc",
            f"echo 'export https_proxy={proxy_url}' >> /home/admin/.bashrc",
            f"echo 'export HTTP_PROXY={proxy_url}' >> /home/admin/.bashrc",
            f"echo 'export HTTPS_PROXY={proxy_url}' >> /home/admin/.bashrc",
            f"echo 'export http_proxy={proxy_url}' >> /home/admin/.profile",
            f"echo 'export https_proxy={proxy_url}' >> /home/admin/.profile",
            f"echo 'export HTTP_PROXY={proxy_url}' >> /home/admin/.profile",
            f"echo 'export HTTPS_PROXY={proxy_url}' >> /home/admin/.profile",
            "",
        ])

    lines.extend([
        "# ── Fix permissions ───────────────────────────────",
        "chown -R admin:admin /home/admin 2>/dev/null || true",
        "",
        "# ── Generate SSH host keys ──────────────────────────",
        "ssh-keygen -A 2>/dev/null || true",
        "",
        "# ── Log session start ─────────────────────────────",
        f'echo \'{{"timestamp": "\'$(date -u +%Y-%m-%dT%H:%M:%SZ)\'", '
        f'"session_id": "{session_id}", "layer": 2, "event": "container_ready", '
        f'"data": {{"contradictions": {len(contradictions)}}}}}\' '
        ">> /var/labyrinth/forensics/sessions/${LABYRINTH_SESSION_ID:-unknown}.jsonl",
        "",
        "# ── Start SSH daemon ──────────────────────────────",
        "exec /usr/sbin/sshd -D -e",
    ])

    return "\n".join(lines)
