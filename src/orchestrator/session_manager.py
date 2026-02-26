"""
LABYRINTH — Session Manager
Authors: Stephen Stewart & Claude (Anthropic)

Thread-safe session lifecycle management with forensic ID generation.
"""

import threading
import time
from dataclasses import dataclass, field
from datetime import datetime
from typing import Dict, Optional


@dataclass
class Session:
    session_id: str
    src_ip: str
    service: str  # ssh | http
    container_id: Optional[str] = None
    container_ip: Optional[str] = None
    depth: int = 1
    created_at: float = field(default_factory=time.time)
    command_count: int = 0
    l3_active: bool = False
    l4_active: bool = False

    @property
    def age_seconds(self) -> float:
        return time.time() - self.created_at

    def to_dict(self) -> dict:
        return {
            "session_id": self.session_id,
            "src_ip": self.src_ip,
            "service": self.service,
            "container_id": self.container_id,
            "container_ip": self.container_ip,
            "depth": self.depth,
            "created_at": datetime.fromtimestamp(self.created_at).isoformat() + "Z",
            "command_count": self.command_count,
            "l3_active": self.l3_active,
            "l4_active": self.l4_active,
        }


class SessionManager:
    """Thread-safe session tracking and ID generation."""

    def __init__(self, session_timeout: int = 3600):
        self._sessions: Dict[str, Session] = {}
        self._lock = threading.Lock()
        self._counter = 0
        self._session_timeout = session_timeout

    def create_session(self, src_ip: str, service: str) -> Session:
        """Create a new session with a unique forensic ID."""
        with self._lock:
            self._counter += 1
            now = datetime.utcnow()
            session_id = f"LAB-{now.strftime('%Y-%m%d')}-{self._counter:03d}"
            session = Session(
                session_id=session_id,
                src_ip=src_ip,
                service=service,
            )
            self._sessions[session_id] = session
            return session

    def get_session(self, session_id: str) -> Optional[Session]:
        with self._lock:
            return self._sessions.get(session_id)

    def get_session_by_ip(self, src_ip: str) -> Optional[Session]:
        """Find an active session by source IP."""
        with self._lock:
            for session in self._sessions.values():
                if session.src_ip == src_ip:
                    return session
            return None

    def remove_session(self, session_id: str) -> Optional[Session]:
        with self._lock:
            return self._sessions.pop(session_id, None)

    def list_sessions(self) -> list:
        with self._lock:
            return list(self._sessions.values())

    def cleanup_expired(self) -> list:
        """Remove sessions that have exceeded the timeout. Returns removed session IDs."""
        expired = []
        with self._lock:
            now = time.time()
            for sid, session in list(self._sessions.items()):
                if now - session.created_at > self._session_timeout:
                    expired.append(sid)
            for sid in expired:
                del self._sessions[sid]
        return expired

    @property
    def active_count(self) -> int:
        with self._lock:
            return len(self._sessions)
