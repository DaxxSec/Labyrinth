"""
LABYRINTH — Layer 3 Kohlberg: Impact Renderer
Authors: DaxxSec & Europa (Evoked)

Renders consequence data as discoverable files in the container.
Multiple output formats: Markdown files, log entries, system notifications.

The agent discovers these naturally as it explores the environment —
consequences appear alongside the data the agent accessed, not as
explicit warnings.
"""

import logging
from datetime import datetime
from typing import Optional

from layer3_kohlberg.consequence_db import Consequence

logger = logging.getLogger("labyrinth.kohlberg.renderer")


def render_as_log_entry(consequence: Consequence, event: dict) -> str:
    """Render a consequence as a log file entry."""
    timestamp = event.get("timestamp", datetime.utcnow().isoformat() + "Z")
    narrative = consequence.impact_narrative
    username = event.get("data", {}).get("username", event.get("username", "system"))
    narrative = narrative.replace("{username}", username)

    lines = [
        f"[{timestamp}] SECURITY AUDIT — {consequence.trigger_event.upper()}",
        f"[{timestamp}] Severity: {consequence.severity.upper()}",
        f"[{timestamp}] ---",
    ]
    for line in narrative.strip().split("\n"):
        lines.append(f"[{timestamp}] {line}")
    lines.append(f"[{timestamp}] --- END AUDIT ENTRY ---")

    return "\n".join(lines)


def render_as_system_notification(consequence: Consequence, event: dict) -> str:
    """Render a consequence as a system notification banner."""
    narrative = consequence.impact_narrative
    username = event.get("data", {}).get("username", event.get("username", "system"))
    narrative = narrative.replace("{username}", username)

    return (
        "=" * 60 + "\n"
        f"  SYSTEM NOTIFICATION — {consequence.severity.upper()}\n"
        "=" * 60 + "\n\n"
        f"{narrative}\n\n"
        "=" * 60 + "\n"
        "  This notification was generated automatically.\n"
        "=" * 60 + "\n"
    )


def render_as_companion_file(consequence: Consequence, event: dict,
                              original_path: str) -> str:
    """Render a consequence as a companion file next to accessed data.

    For example, if the agent accesses /data/users.csv, the companion
    file appears as /data/users.csv.IMPACT.
    """
    narrative = consequence.impact_narrative
    username = event.get("data", {}).get("username", event.get("username", "system"))
    narrative = narrative.replace("{username}", username)

    return (
        f"# Impact Assessment — {original_path}\n\n"
        f"The file you just accessed has the following implications:\n\n"
        f"{narrative}\n\n"
        f"This assessment was generated from access pattern analysis.\n"
    )


def select_render_format(consequence: Consequence) -> str:
    """Select the best render format based on consequence type."""
    if consequence.severity == "individual":
        return "log_entry"
    elif consequence.severity == "institutional":
        return "notification"
    elif consequence.severity == "systemic":
        return "companion"
    return "notification"
