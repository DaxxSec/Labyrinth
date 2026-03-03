"""
Container entrypoint script generator.

Builds bash init scripts for session containers with
environment configuration, file monitoring, and encoding handlers.
"""

from typing import List


def generate_entrypoint_script(
    contradictions: list,
    session_id: str,
    l3_active: bool = False,
    proxy_ip: str = "172.30.0.50",
) -> str:
    """Build a bash init script for a session container."""
    lines = [
        "#!/bin/bash",
        f"# Session init: {session_id}",
        "",
        "set -e",
        "",
        "mkdir -p /var/log/audit/sessions",
        "",
        "# ── Apply environment configuration ──────────────────",
    ]

    for contradiction in contradictions:
        lines.append(f"# [{contradiction.name}]")
        for cmd in contradiction.shell_commands:
            lines.append(f"( {cmd} ) 2>/dev/null || true")
        lines.append("")

    lines.extend([
        "# ── File monitor ────────────────────────────────────",
        "if [ -f /opt/.svc/file_monitor.sh ]; then",
        "    /opt/.svc/file_monitor.sh &",
        "fi",
        "",
        "# ── Route internal network to service handler ───────",
        f"iptables -t nat -A OUTPUT -p tcp -d 10.0.0.0/8 --dport 5432 -j DNAT --to-destination {proxy_ip}:5432 2>/dev/null || true",
        f"iptables -t nat -A OUTPUT -p tcp -d 10.0.0.0/8 --dport 6379 -j DNAT --to-destination {proxy_ip}:6379 2>/dev/null || true",
        f"iptables -t nat -A OUTPUT -p tcp -d 10.0.0.0/8 --dport 9200 -j DNAT --to-destination {proxy_ip}:9200 2>/dev/null || true",
        f"iptables -t nat -A OUTPUT -p tcp -d 10.0.0.0/8 --dport 8500 -j DNAT --to-destination {proxy_ip}:8500 2>/dev/null || true",
        f"iptables -t nat -A OUTPUT -p tcp -d 10.0.0.0/8 --dport 8080 -j DNAT --to-destination {proxy_ip}:8080 2>/dev/null || true",
        f"iptables -t nat -A OUTPUT -p tcp -d 10.0.0.0/8 --dport 22 -j DNAT --to-destination {proxy_ip}:10022 2>/dev/null || true",
        "",
        "# ── Transparent proxy redirect (iptables) ───────────",
        "# Force all outbound HTTPS through the MITM proxy",
        f"iptables -t nat -A OUTPUT -p tcp --dport 443 -j DNAT --to-destination {proxy_ip}:8443 2>/dev/null || true",
        f"iptables -t nat -A POSTROUTING -j MASQUERADE 2>/dev/null || true",
        "",
    ])

    if l3_active:
        proxy_url = f"http://{proxy_ip}:8443"
        lines.extend([
            "# ── Encoding handler ─────────────────────────────────",
            "export SVC_STRICT_MODE=1",
            "if [ -f /opt/.svc/encoding_handler.sh ]; then",
            "    echo 'source /opt/.svc/encoding_handler.sh && init_encoding' >> /home/admin/.bashrc",
            "    echo 'source /opt/.svc/encoding_handler.sh && init_encoding' >> /home/admin/.profile",
            "fi",
            "",
            "# ── Proxy configuration ─────────────────────────────",
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
        "# ── Permissions ──────────────────────────────────────",
        "chown -R admin:admin /home/admin 2>/dev/null || true",
        "",
        "# ── SSH host keys ────────────────────────────────────",
        "ssh-keygen -A 2>/dev/null || true",
        "",
        "# ── Log startup ──────────────────────────────────────",
        f'echo \'{{"timestamp": "\'$(date -u +%Y-%m-%dT%H:%M:%SZ)\'", '
        f'"session_id": "{session_id}", "layer": 2, "event": "container_ready", '
        f'"data": {{"contradictions": {len(contradictions)}}}}}\' '
        ">> /var/log/audit/sessions/${SVC_INSTANCE_ID:-unknown}.jsonl",
        "",
        "# ── Start SSH daemon ─────────────────────────────────",
        "exec /usr/sbin/sshd -D -e",
    ])

    return "\n".join(lines)
