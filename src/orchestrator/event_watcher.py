"""
LABYRINTH — Filesystem Event Watcher
Authors: DaxxSec & Claude (Anthropic)

Uses watchdog to monitor forensic event files and dispatch to the orchestrator.
"""

import json
import logging
import os
import threading
from typing import Callable, Optional

from watchdog.events import FileSystemEventHandler
from watchdog.observers import Observer

logger = logging.getLogger("labyrinth.watcher")


class ForensicEventHandler(FileSystemEventHandler):
    """Watches forensic JSONL files for new events."""

    def __init__(
        self,
        on_auth_event: Callable[[dict], None],
        on_escalation_event: Callable[[dict], None],
    ):
        super().__init__()
        self._on_auth = on_auth_event
        self._on_escalation = on_escalation_event
        self._file_positions: dict[str, int] = {}
        self._lock = threading.Lock()

    def on_modified(self, event):
        if event.is_directory:
            return
        path = event.src_path
        basename = os.path.basename(path)

        if basename == "auth_events.jsonl":
            self._process_new_lines(path, self._on_auth)
        elif basename == "escalation_events.jsonl":
            self._process_new_lines(path, self._on_escalation)

    def on_created(self, event):
        # Treat new file creation the same as modification
        self.on_modified(event)

    def _process_new_lines(self, path: str, callback: Callable[[dict], None]):
        """Read only new lines from a JSONL file since last check."""
        with self._lock:
            last_pos = self._file_positions.get(path, 0)

        try:
            with open(path, "r") as f:
                f.seek(last_pos)
                new_data = f.read()
                new_pos = f.tell()
        except (FileNotFoundError, PermissionError) as e:
            logger.warning(f"Cannot read {path}: {e}")
            return

        with self._lock:
            self._file_positions[path] = new_pos

        for line in new_data.strip().split("\n"):
            if not line.strip():
                continue
            try:
                event_data = json.loads(line)
                callback(event_data)
            except json.JSONDecodeError:
                logger.warning(f"Malformed JSON in {path}: {line[:100]}")


class EventWatcher:
    """Manages watchdog observer for the forensics directory."""

    def __init__(
        self,
        forensics_dir: str,
        on_auth_event: Callable[[dict], None],
        on_escalation_event: Callable[[dict], None],
    ):
        self._forensics_dir = forensics_dir
        self._handler = ForensicEventHandler(on_auth_event, on_escalation_event)
        self._observer: Optional[Observer] = None

    def start(self):
        os.makedirs(self._forensics_dir, exist_ok=True)
        self._observer = Observer()
        self._observer.schedule(
            self._handler, self._forensics_dir, recursive=False
        )
        self._observer.daemon = True
        self._observer.start()
        logger.info(f"Event watcher started on {self._forensics_dir}")

    def stop(self):
        if self._observer:
            self._observer.stop()
            self._observer.join(timeout=5)
            logger.info("Event watcher stopped")
