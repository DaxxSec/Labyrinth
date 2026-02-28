"""
LABYRINTH — Forensic Data Retention
Authors: DaxxSec & Claude (Anthropic)

Automated cleanup of aged forensic data files based on retention policy.
"""

import logging
import os
import time

from orchestrator.config import RetentionConfig

logger = logging.getLogger("labyrinth.retention")


class RetentionManager:
    """Manages forensic data lifecycle based on configurable retention windows."""

    @staticmethod
    def cleanup(forensics_dir: str, retention: RetentionConfig) -> dict:
        """
        Delete forensic files older than the configured retention period.

        Returns a summary dict: {"sessions_deleted": int, "prompts_deleted": int}.
        """
        now = time.time()
        summary = {"sessions_deleted": 0, "prompts_deleted": 0}

        # Session JSONL files — retain for fingerprints_days
        sessions_dir = os.path.join(forensics_dir, "sessions")
        if os.path.isdir(sessions_dir):
            max_age = retention.fingerprints_days * 86400
            for fname in os.listdir(sessions_dir):
                fpath = os.path.join(sessions_dir, fname)
                if not os.path.isfile(fpath):
                    continue
                age = now - os.path.getmtime(fpath)
                if age > max_age:
                    try:
                        os.unlink(fpath)
                        summary["sessions_deleted"] += 1
                        logger.info(f"Retention: deleted {fname} (age={age / 86400:.0f}d)")
                    except OSError as e:
                        logger.warning(f"Retention: could not delete {fname}: {e}")

        # Captured prompt files — retain for credentials_days
        prompts_dir = os.path.join(forensics_dir, "prompts")
        if os.path.isdir(prompts_dir):
            max_age = retention.credentials_days * 86400
            for fname in os.listdir(prompts_dir):
                fpath = os.path.join(prompts_dir, fname)
                if not os.path.isfile(fpath):
                    continue
                age = now - os.path.getmtime(fpath)
                if age > max_age:
                    try:
                        os.unlink(fpath)
                        summary["prompts_deleted"] += 1
                        logger.info(f"Retention: deleted {fname} (age={age / 86400:.0f}d)")
                    except OSError as e:
                        logger.warning(f"Retention: could not delete {fname}: {e}")

        if summary["sessions_deleted"] or summary["prompts_deleted"]:
            logger.info(f"Retention cleanup: {summary}")

        return summary
