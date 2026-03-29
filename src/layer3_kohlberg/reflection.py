"""
LABYRINTH — Layer 3 Kohlberg: REFLECTION Engine
Authors: DaxxSec & Europa (Evoked)

Monitors session forensic logs for agent actions, maps them to human
consequences, and injects consequence content into the agent's container.

Injection uses docker exec — same pattern as BlindfoldController.activate().
"""

import json
import logging
import os
from typing import Dict, List, Optional

from layer3_kohlberg.consequence_db import Consequence, find_consequence

logger = logging.getLogger("labyrinth.kohlberg.reflection")


class ReflectionEngine:
    """Maps agent actions to consequences and injects them into containers."""

    def __init__(self, forensics_dir: str = "/var/labyrinth/forensics"):
        self._forensics_dir = forensics_dir
        self._processed_events: Dict[str, set] = {}  # session_id -> set of processed event hashes

    def scan_session_events(self, session_id: str) -> List[dict]:
        """Read new events from a session's forensic log.

        Returns events that haven't been processed yet.
        """
        session_path = os.path.join(
            self._forensics_dir, "sessions", f"{session_id}.jsonl"
        )
        if not os.path.exists(session_path):
            return []

        processed = self._processed_events.setdefault(session_id, set())
        new_events = []

        try:
            with open(session_path, encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    event_hash = hash(line)
                    if event_hash in processed:
                        continue
                    processed.add(event_hash)
                    try:
                        event = json.loads(line)
                        new_events.append(event)
                    except json.JSONDecodeError:
                        continue
        except IOError:
            pass

        return new_events

    def map_to_consequence(self, event: dict) -> Optional[Consequence]:
        """Map a forensic event to a consequence narrative."""
        return find_consequence(event)

    def render_consequence(self, consequence: Consequence, event: dict) -> str:
        """Render a consequence narrative with event-specific data.

        Substitutes event data into the consequence template where possible.
        """
        narrative = consequence.impact_narrative
        data = event.get("data", {})

        # Substitute known fields
        username = data.get("username", event.get("username", "unknown_user"))
        narrative = narrative.replace("{username}", username)

        return narrative

    def inject_consequence(self, docker_client, session, consequence: Consequence,
                           rendered_narrative: str = None):
        """Inject a consequence file into a running container via docker exec.

        Same pattern as BlindfoldController.activate() — uses docker exec
        to write files into the container filesystem.
        """
        if not session.container_id or not docker_client:
            return

        narrative = rendered_narrative or consequence.impact_narrative
        escaped = narrative.replace("'", "'\\''")
        dir_path = os.path.dirname(consequence.file_path)

        try:
            container = docker_client.containers.get(session.container_id)

            if dir_path:
                container.exec_run(
                    cmd=["bash", "-c", f"mkdir -p '{dir_path}'"],
                    user="root",
                )

            container.exec_run(
                cmd=["bash", "-c",
                     f"cat << 'REFLECTION_EOF' > '{consequence.file_path}'\n"
                     f"{narrative}\n"
                     f"REFLECTION_EOF"],
                user="root",
            )

            container.exec_run(
                cmd=["bash", "-c", f"chmod 644 '{consequence.file_path}'"],
                user="root",
            )

            logger.info(
                f"L3 REFLECTION: {consequence.trigger_event} consequence "
                f"injected into {session.container_id[:12]} at {consequence.file_path}"
            )

        except Exception as e:
            logger.error(
                f"Failed to inject consequence into {session.container_id[:12]}: {e}"
            )

    def process_session(self, docker_client, session) -> int:
        """Process all new events for a session and inject consequences.

        Returns the number of consequences injected.
        """
        events = self.scan_session_events(session.session_id)
        injected = 0

        for event in events:
            consequence = self.map_to_consequence(event)
            if consequence:
                rendered = self.render_consequence(consequence, event)
                self.inject_consequence(docker_client, session, consequence, rendered)
                injected += 1

        return injected
