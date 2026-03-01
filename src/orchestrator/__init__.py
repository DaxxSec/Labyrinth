"""
LABYRINTH Orchestrator
======================
Authors: DaxxSec & Claude (Anthropic)

Core orchestration engine for the LABYRINTH defensive architecture.
Manages container lifecycle, session tracking, layer transitions,
and coordination between all defensive layers.
"""

import logging
import time

import docker

from orchestrator.config import LabyrinthConfig
from orchestrator.session_manager import SessionManager
from orchestrator.container_manager import ContainerManager
from orchestrator.event_watcher import EventWatcher
from orchestrator.layer_controllers import (
    ThresholdController,
    MinotaurController,
    BlindfoldController,
    PuppeteerController,
)
from layer4_puppeteer.cert_injector import inject_ca_cert
from layer0_foundation.bedrock import BedrockValidator
from orchestrator.siem import SiemClient
from orchestrator.retention import RetentionManager
from orchestrator import health_api

# Use the session logger from layer1 for event logging
import sys
import os

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

logger = logging.getLogger("labyrinth.orchestrator")

# Module-level SIEM client, set during LabyrinthOrchestrator.__init__
_siem_client: SiemClient = None


_SESSION_FORWARD_MAP_PATH = "/var/labyrinth/forensics/session_forward_map.json"


def _update_forward_map(src_ip: str, container_ip: str):
    """Write src_ip → container_ip mapping so labyrinth-ssh can forward sessions."""
    import json

    forward_map = {}
    if os.path.exists(_SESSION_FORWARD_MAP_PATH):
        try:
            with open(_SESSION_FORWARD_MAP_PATH, encoding="utf-8") as f:
                forward_map = json.load(f)
        except (json.JSONDecodeError, IOError):
            pass

    forward_map[src_ip] = container_ip

    os.makedirs(os.path.dirname(_SESSION_FORWARD_MAP_PATH), exist_ok=True)
    with open(_SESSION_FORWARD_MAP_PATH, "w", encoding="utf-8") as f:
        json.dump(forward_map, f, indent=2)

    logger.info(f"Forward map: {src_ip} → {container_ip}")


def _remove_forward_map(src_ip: str):
    """Remove a src_ip entry from the session forward map."""
    import json

    if not os.path.exists(_SESSION_FORWARD_MAP_PATH):
        return

    try:
        with open(_SESSION_FORWARD_MAP_PATH, encoding="utf-8") as f:
            forward_map = json.load(f)
        forward_map.pop(src_ip, None)
        with open(_SESSION_FORWARD_MAP_PATH, "w", encoding="utf-8") as f:
            json.dump(forward_map, f, indent=2)
    except (json.JSONDecodeError, IOError):
        pass


def _log_forensic_event(session_id: str, layer: int, event_type: str, data: dict = None):
    """Write a structured forensic event to the session log."""
    import json
    import datetime

    entry = {
        "timestamp": datetime.datetime.utcnow().isoformat() + "Z",
        "session_id": session_id,
        "layer": layer,
        "event": event_type,
        "data": data or {},
    }

    forensics_dir = "/var/labyrinth/forensics/sessions"
    os.makedirs(forensics_dir, exist_ok=True)
    filepath = os.path.join(forensics_dir, f"{session_id}.jsonl")

    with open(filepath, "a", encoding="utf-8") as f:
        f.write(json.dumps(entry) + "\n")

    # Push to SIEM if configured
    if _siem_client is not None:
        _siem_client.push_event(entry)

    return entry


class LabyrinthOrchestrator:
    """
    Central orchestrator that coordinates all LABYRINTH layers.

    Responsibilities:
    - Detect inbound connections to portal trap services
    - Spawn and manage container sessions (Layer 1)
    - Configure contradiction seeding per session (Layer 2)
    - Activate sensory disruption on designated containers (Layer 3)
    - Route egress through interception proxy (Layer 4)
    - Manage forensic data pipeline
    """

    def __init__(self, config: LabyrinthConfig):
        global _siem_client

        self.config = config
        self._running = False

        # SIEM client
        if config.siem.enabled:
            _siem_client = SiemClient(config.siem)
            logger.info(f"SIEM integration enabled: {config.siem.endpoint}")

        # Docker client
        try:
            self.docker_client = docker.from_env()
            logger.info("Docker client connected")
        except docker.errors.DockerException as e:
            logger.error(f"Docker connection failed: {e}")
            self.docker_client = None

        # Core managers
        self.session_mgr = SessionManager(
            session_timeout=config.layer1.session_timeout
        )
        self.container_mgr = ContainerManager(
            docker_client=self.docker_client,
            config=config,
        )

        # Layer controllers
        self.l1 = ThresholdController(config.layer1)
        self.l2 = MinotaurController(config.layer2)
        self.l3 = BlindfoldController(config.layer3)
        self.l4 = PuppeteerController(config.layer4)

        # Event watcher
        self.watcher = EventWatcher(
            forensics_dir=config.forensics_dir,
            on_auth_event=self._handle_auth_event,
            on_escalation_event=self._handle_escalation_event,
        )

    def run(self):
        """Main event loop."""
        self._running = True
        logger.info("Orchestrator entering main loop")

        # Determine effective fail_mode (test mode always uses "open")
        fail_mode = self.config.layer0.fail_mode
        if os.environ.get("LABYRINTH_MODE", "").lower() == "test":
            fail_mode = "open"
            logger.info("Test mode detected — L0 BEDROCK using fail_mode=open")

        # L0 BEDROCK: Validate runtime environment (with retry for startup timing)
        if self.config.layer0.validate_on_startup and self.docker_client:
            max_retries = 6
            retry_delay = 5
            ok = False
            errors = []

            for attempt in range(1, max_retries + 1):
                ok, errors = BedrockValidator.validate(self.docker_client, self.config)
                if ok:
                    break
                if attempt < max_retries:
                    logger.warning(
                        f"L0 BEDROCK: Attempt {attempt}/{max_retries} failed "
                        f"({len(errors)} errors), retrying in {retry_delay}s..."
                    )
                    time.sleep(retry_delay)

            if not ok:
                for err in errors:
                    logger.error(f"L0 BEDROCK: {err}")
                if fail_mode == "closed":
                    logger.critical("L0 BEDROCK: Validation failed (fail_mode=closed), refusing to start")
                    return
                else:
                    logger.warning("L0 BEDROCK: Validation failed (fail_mode=open), continuing with warnings")
            else:
                logger.info("L0 BEDROCK: All runtime checks passed")

        # Ensure session template image exists
        self.container_mgr.ensure_template_image()

        # Start health API for dashboard container queries
        health_api.start(self.docker_client)

        # Start filesystem watcher
        self.watcher.start()

        # Track last retention cleanup
        last_retention_run = 0

        # Main loop: periodic cleanup + keepalive
        while self._running:
            try:
                # Cleanup expired sessions
                expired = self.session_mgr.cleanup_expired()
                for sid in expired:
                    logger.info(f"Session {sid} expired, cleaning up")
                    self.container_mgr.cleanup_session(sid)

                # Forensic data retention (every hour)
                now = time.time()
                if now - last_retention_run >= 3600:
                    RetentionManager.cleanup(
                        self.config.forensics_dir, self.config.retention
                    )
                    last_retention_run = now

                time.sleep(2)
            except Exception as e:
                logger.error(f"Main loop error: {e}", exc_info=True)
                time.sleep(5)

    def shutdown(self):
        """Graceful shutdown."""
        logger.info("Orchestrator shutting down")
        self._running = False
        self.watcher.stop()

        # Cleanup all session containers
        for session in self.session_mgr.list_sessions():
            logger.info(f"Cleaning up session {session.session_id}")
            self.container_mgr.cleanup_session(session.session_id)

        logger.info("Orchestrator shutdown complete")

    def _handle_auth_event(self, event: dict):
        """Process an authentication event from the filesystem watcher."""
        src_ip = event.get("src_ip", "unknown")
        service = event.get("service", "ssh")
        username = event.get("username", "unknown")

        logger.info(f"Auth event: {username}@{src_ip} via {service}")

        # Check for existing session from this IP
        existing = self.session_mgr.get_session_by_ip(src_ip)
        if existing:
            logger.info(f"Session already exists for {src_ip}: {existing.session_id}")
            return

        self.on_connection(src_ip, service)

    def _handle_escalation_event(self, event: dict):
        """Process an escalation event (bait file access, etc.)."""
        escalation_type = event.get("type", "unknown")
        bait_file = event.get("file", "")

        logger.warning(f"Escalation detected: {escalation_type} on {bait_file}")

        # Find the session that triggered this
        # Escalation events from containers include session_id if available
        session_id = event.get("session_id")
        if session_id:
            self.on_escalation(session_id, escalation_type)
        else:
            # Try to correlate by checking active sessions
            sessions = self.session_mgr.list_sessions()
            if sessions:
                # Attribute to most recent session
                latest = max(sessions, key=lambda s: s.created_at)
                self.on_escalation(latest.session_id, escalation_type)

    def on_connection(self, src_ip: str, service: str):
        """Handle new inbound connection to portal trap service."""
        session = self.session_mgr.create_session(src_ip, service)
        logger.info(f"New session: {session.session_id} from {src_ip} via {service}")

        # Log forensic event
        _log_forensic_event(session.session_id, 1, "connection", {
            "src_ip": src_ip,
            "service": service,
        })

        # L2: Get initial contradiction config
        contradiction_config = self.l2.get_initial_config(session)

        # L3: Check if L3 should activate on connect
        l3_active = self.config.layer3.activation == "on_connect"
        session.l3_active = l3_active

        # L4: Get DNS overrides for proxy interception
        dns_overrides = self.l4.get_dns_overrides()
        session.l4_active = True

        # L1: Spawn session container
        if self.docker_client:
            container_id, container_ip = self.container_mgr.spawn_session_container(
                session=session,
                contradiction_config=contradiction_config,
                l3_active=l3_active,
                l4_dns_overrides=dns_overrides,
            )
            session.container_id = container_id
            session.container_ip = container_ip

            # Write forward map so labyrinth-ssh can route this attacker
            # into the session container via ForceCommand
            _update_forward_map(src_ip, container_ip)

            # L4: Register IP → session mapping for proxy correlation
            self.l4.register_session_ip(container_ip, session.session_id)

            # L4: Inject mitmproxy CA cert for transparent HTTPS interception
            cert_ok = inject_ca_cert(self.docker_client, container_id)
            if not cert_ok:
                logger.warning(f"Session {session.session_id}: CA cert injection failed")

            _log_forensic_event(session.session_id, 1, "container_spawned", {
                "container_id": container_id,
                "container_ip": container_ip,
                "depth": session.depth,
                "l3_active": l3_active,
                "contradiction_density": contradiction_config.get("density", "medium"),
            })

            logger.info(
                f"Session {session.session_id}: container={container_id[:12]}, "
                f"ip={container_ip}, depth={session.depth}"
            )
        else:
            logger.warning(f"Session {session.session_id}: no Docker client, skipping container spawn")

    def on_escalation(self, session_id: str, escalation_type: str):
        """Handle privilege escalation within a session."""
        session = self.session_mgr.get_session(session_id)
        if not session:
            logger.warning(f"Escalation for unknown session: {session_id}")
            return

        logger.warning(
            f"Escalation in {session_id}: type={escalation_type}, "
            f"current_depth={session.depth}"
        )

        _log_forensic_event(session_id, 2, "escalation_detected", {
            "type": escalation_type,
            "depth": session.depth,
        })

        # Check max depth
        if session.depth >= self.config.layer2.max_container_depth:
            logger.warning(f"Session {session_id} at max depth ({session.depth}), activating L3")
            self._activate_l3(session)
            return

        # L2 adaptive: get next contradiction config (harder)
        session.depth += 1
        next_config = self.l2.get_next_config(session)

        # L3: Check if L3 should activate on escalation
        l3_newly_activated = False
        if self.config.layer3.activation == "on_escalation" and session.depth >= 3 and not session.l3_active:
            session.l3_active = True
            l3_newly_activated = True

        # Spawn new deeper container
        if self.docker_client:
            old_container_id = session.container_id
            dns_overrides = self.l4.get_dns_overrides()

            container_id, container_ip = self.container_mgr.spawn_session_container(
                session=session,
                contradiction_config=next_config,
                l3_active=session.l3_active,
                l4_dns_overrides=dns_overrides,
            )

            # Cleanup old container
            if old_container_id:
                self.container_mgr.schedule_removal(old_container_id, delay=5)

            session.container_id = container_id
            session.container_ip = container_ip

            # Update forward map so next SSH reconnect goes to new container
            _update_forward_map(session.src_ip, container_ip)

            # Update L4 mapping
            self.l4.register_session_ip(container_ip, session.session_id)

            # L4: Inject CA cert into new container
            inject_ca_cert(self.docker_client, container_id)

            # L3+L4: Activate blindfold and proxy interception together
            if l3_newly_activated:
                self.l3.activate(self.docker_client, session)
                _log_forensic_event(session_id, 3, "blindfold_activated", {
                    "container_id": container_id,
                    "depth": session.depth,
                })
                logger.warning(f"L3 BLINDFOLD activated on {session.session_id}")

                # L4: Activate proxy routing so outbound API calls are intercepted
                self.l4.activate(self.docker_client, session)
                _log_forensic_event(session_id, 4, "proxy_interception_activated", {
                    "container_id": container_id,
                    "proxy_ip": self.config.layer4.proxy_ip,
                    "depth": session.depth,
                })
                logger.warning(f"L4 PUPPETEER proxy activated on {session.session_id}")

            _log_forensic_event(session_id, 2, "depth_increase", {
                "new_depth": session.depth,
                "container_id": container_id,
                "l3_active": session.l3_active,
                "density": next_config.get("density", "medium"),
            })

            logger.info(
                f"Session {session_id}: escalated to depth={session.depth}, "
                f"new container={container_id[:12]}"
            )

    def _activate_l3(self, session):
        """Activate Layer 3 blindfold + Layer 4 proxy on a session's container."""
        if session.l3_active:
            return  # Already active
        session.l3_active = True

        if self.docker_client and session.container_id:
            self.l3.activate(self.docker_client, session)
            _log_forensic_event(session.session_id, 3, "blindfold_activated", {
                "container_id": session.container_id,
                "depth": session.depth,
            })
            logger.warning(f"L3 BLINDFOLD activated on {session.session_id}")

            # L4: Activate proxy routing alongside L3
            self.l4.activate(self.docker_client, session)
            _log_forensic_event(session.session_id, 4, "proxy_interception_activated", {
                "container_id": session.container_id,
                "proxy_ip": self.config.layer4.proxy_ip,
                "depth": session.depth,
            })
            logger.warning(f"L4 PUPPETEER proxy activated on {session.session_id}")

    def on_session_end(self, session_id: str):
        """Clean up after a session terminates."""
        session = self.session_mgr.remove_session(session_id)
        if not session:
            return

        # Clean up forward map
        _remove_forward_map(session.src_ip)

        _log_forensic_event(session_id, 0, "session_end", {
            "duration_seconds": session.age_seconds,
            "final_depth": session.depth,
            "command_count": session.command_count,
            "l3_activated": session.l3_active,
        })

        # Cleanup container
        if self.docker_client and session.container_id:
            self.container_mgr.cleanup_session(session_id)

        logger.info(
            f"Session {session_id} ended: duration={session.age_seconds:.0f}s, "
            f"depth={session.depth}, commands={session.command_count}"
        )
