"""
LABYRINTH — SIEM Integration
Authors: Stephen Stewart & Claude (Anthropic)

Fire-and-forget event push to external SIEM endpoints.
"""

import json
import logging
import threading
import urllib.request
import urllib.error

from orchestrator.config import SiemConfig

logger = logging.getLogger("labyrinth.siem")


class SiemClient:
    """Push forensic events to an external SIEM endpoint via HTTP POST."""

    def __init__(self, config: SiemConfig):
        self.enabled = config.enabled
        self.endpoint = config.endpoint
        self.alert_prefix = config.alert_prefix

    def push_event(self, event: dict):
        """
        Send event to SIEM endpoint in a background thread.

        Fire-and-forget: failures are logged but never crash the orchestrator.
        """
        if not self.enabled or not self.endpoint:
            return

        payload = dict(event)
        payload["alert_prefix"] = self.alert_prefix

        thread = threading.Thread(
            target=self._send, args=(payload,), daemon=True
        )
        thread.start()

    def _send(self, payload: dict):
        """HTTP POST to SIEM endpoint."""
        try:
            data = json.dumps(payload).encode("utf-8")
            req = urllib.request.Request(
                self.endpoint,
                data=data,
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            with urllib.request.urlopen(req, timeout=10) as resp:
                logger.debug(f"SIEM push: {resp.status}")
        except Exception as e:
            logger.warning(f"SIEM push failed: {e}")
