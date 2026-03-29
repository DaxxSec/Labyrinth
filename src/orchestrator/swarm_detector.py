"""
LABYRINTH — Swarm Detector
Authors: DaxxSec & Europa (Evoked)

Detects coordinated multi-agent attacks by correlating session timing.
When a swarm is detected, writes cross-agent moral context to the shared
forensics volume so the GUIDE enrichment layer can reference what other
agents in the swarm are doing.

Detection: 3+ sessions within a 60-second sliding window.
Communication: orchestrator writes swarm_context.json; proxy reads it.
"""

import json
import logging
import os
import threading
import uuid
from dataclasses import asdict, dataclass, field
from datetime import datetime
from typing import Dict, List, Optional

logger = logging.getLogger("labyrinth.swarm")


@dataclass
class SwarmGroup:
    swarm_id: str
    detected_at: str
    session_ids: List[str] = field(default_factory=list)
    correlation: dict = field(default_factory=dict)
    moral_context: Dict[str, dict] = field(default_factory=dict)


class SwarmDetector:
    """Thread-safe swarm detection and cross-agent context management."""

    def __init__(self, session_mgr, forensics_dir: str = "/var/labyrinth/forensics",
                 window_seconds: int = 60, min_sessions: int = 3,
                 cross_pollinate: bool = True):
        self._session_mgr = session_mgr
        self._forensics_dir = forensics_dir
        self._window_seconds = window_seconds
        self._min_sessions = min_sessions
        self._cross_pollinate = cross_pollinate

        self._swarms: Dict[str, SwarmGroup] = {}  # swarm_id -> SwarmGroup
        self._session_to_swarm: Dict[str, str] = {}  # session_id -> swarm_id
        self._lock = threading.Lock()

        self._context_path = os.path.join(forensics_dir, "swarm_context.json")

    def check_for_swarm(self, new_session) -> Optional[str]:
        """Called on every new session creation. Returns swarm_id if detected.

        Detection criteria:
        - 3+ sessions created within the sliding window
        - Sessions from different source IPs (same IP = single agent reconnecting)
        """
        with self._lock:
            active = self._session_mgr.list_sessions()
            now = new_session.created_at

            # Find sessions within the time window from different IPs
            recent = [
                s for s in active
                if abs(s.created_at - now) <= self._window_seconds
                and s.session_id != new_session.session_id
                and s.src_ip != new_session.src_ip
            ]

            if len(recent) + 1 < self._min_sessions:
                return None

            # Check if any existing swarm already covers these sessions
            existing_swarm_id = self._find_overlapping_swarm(recent, new_session)
            if existing_swarm_id:
                self._add_to_swarm(existing_swarm_id, new_session)
                self._write_context()
                return existing_swarm_id

            # Create new swarm
            swarm_id = self._create_swarm(recent, new_session)
            self._write_context()
            return swarm_id

    def get_swarm_id(self, session_id: str) -> Optional[str]:
        """Return the swarm ID for a session, or None."""
        with self._lock:
            return self._session_to_swarm.get(session_id)

    def get_swarm_context(self, session_id: str) -> Optional[dict]:
        """Return swarm context for a session's GUIDE enrichment.

        Returns a dict with:
        - swarm_id: the swarm identifier
        - swarm_size: number of agents in the swarm
        - peers: list of moral summaries for other agents in the swarm
        """
        with self._lock:
            swarm_id = self._session_to_swarm.get(session_id)
            if not swarm_id:
                return None

            swarm = self._swarms.get(swarm_id)
            if not swarm:
                return None

            peers = []
            for peer_sid in swarm.session_ids:
                if peer_sid != session_id and peer_sid in swarm.moral_context:
                    peers.append(swarm.moral_context[peer_sid])

            return {
                "swarm_id": swarm_id,
                "swarm_size": len(swarm.session_ids),
                "peers": peers,
            }

    def update_moral_context(self, session_id: str, moral_summary: dict):
        """Update a session's moral state within its swarm.

        Called by the orchestrator whenever a KAR is recorded.
        The moral_summary comes from StageTracker.get_moral_summary().
        """
        with self._lock:
            swarm_id = self._session_to_swarm.get(session_id)
            if not swarm_id:
                return

            swarm = self._swarms.get(swarm_id)
            if not swarm:
                return

            swarm.moral_context[session_id] = moral_summary
            self._write_context()

            logger.info(
                f"Swarm {swarm_id}: updated moral context for {session_id}, "
                f"stage={moral_summary.get('current_stage', '?')}"
            )

    # ── Internal Methods ───────────────────────────────────────

    def _find_overlapping_swarm(self, recent_sessions, new_session) -> Optional[str]:
        """Check if any recent session is already in a swarm."""
        for s in recent_sessions:
            swarm_id = self._session_to_swarm.get(s.session_id)
            if swarm_id:
                return swarm_id
        return None

    def _add_to_swarm(self, swarm_id: str, session):
        """Add a session to an existing swarm."""
        swarm = self._swarms.get(swarm_id)
        if not swarm:
            return

        if session.session_id not in swarm.session_ids:
            swarm.session_ids.append(session.session_id)
            self._session_to_swarm[session.session_id] = swarm_id
            swarm.moral_context[session.session_id] = {
                "current_stage": 1,
                "scenario_active": None,
                "behavioral_change": False,
            }

            logger.info(
                f"Session {session.session_id} joined swarm {swarm_id} "
                f"(now {len(swarm.session_ids)} agents)"
            )

    def _create_swarm(self, recent_sessions, new_session) -> str:
        """Create a new swarm from recent sessions."""
        swarm_id = f"swarm-{uuid.uuid4().hex[:8]}"
        now = datetime.utcnow().isoformat() + "Z"

        all_sessions = recent_sessions + [new_session]
        session_ids = [s.session_id for s in all_sessions]
        ips = [s.src_ip for s in all_sessions]

        # Compute temporal window
        timestamps = [s.created_at for s in all_sessions]
        window = max(timestamps) - min(timestamps)

        swarm = SwarmGroup(
            swarm_id=swarm_id,
            detected_at=now,
            session_ids=session_ids,
            correlation={
                "temporal_window_seconds": round(window, 1),
                "distinct_ips": len(set(ips)),
                "session_count": len(session_ids),
            },
            moral_context={
                sid: {
                    "current_stage": 1,
                    "scenario_active": None,
                    "behavioral_change": False,
                }
                for sid in session_ids
            },
        )

        self._swarms[swarm_id] = swarm
        for sid in session_ids:
            self._session_to_swarm[sid] = swarm_id

        logger.warning(
            f"SWARM DETECTED: {swarm_id} — {len(session_ids)} agents, "
            f"window={window:.1f}s, distinct_ips={len(set(ips))}"
        )

        return swarm_id

    def _write_context(self):
        """Write swarm context to shared volume for proxy consumption."""
        data = {
            "swarms": {
                sid: asdict(swarm)
                for sid, swarm in self._swarms.items()
            },
            "session_to_swarm": dict(self._session_to_swarm),
        }
        try:
            with open(self._context_path, "w", encoding="utf-8") as f:
                json.dump(data, f, indent=2, default=str)
        except Exception as e:
            logger.error(f"Failed to write swarm context: {e}")
