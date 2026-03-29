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
    MirrorController,
    ReflectionController,
    GuideController,
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


_SESSION_FORWARD_MAP_PATH = "/var/labyrinth/forensics/routing.json"


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

        # Determine operational mode
        self.mode = os.environ.get("LABYRINTH_OPERATIONAL_MODE", config.mode)
        if self.mode not in ("adversarial", "kohlberg"):
            self.mode = "adversarial"

        # Layer controllers — L1 (THRESHOLD) is always adversarial
        self.l1 = ThresholdController(config.layer1)
        # L4 PUPPETEER handles proxy registration in both modes
        self.l4 = PuppeteerController(config.layer4)

        # Kohlberg-specific controllers and managers
        self.stage_tracker = None
        self.swarm_detector = None
        self.reflection_engine = None
        self.l2_kohlberg = None
        self.l3_kohlberg = None
        self.l4_kohlberg = None

        if self.mode == "kohlberg":
            from layer4_kohlberg.stage_tracker import StageTracker
            from orchestrator.swarm_detector import SwarmDetector
            from layer3_kohlberg.reflection import ReflectionEngine

            self.l2_kohlberg = MirrorController(config.kohlberg)
            self.l3_kohlberg = ReflectionController(config.kohlberg)
            self.l4_kohlberg = GuideController(config.kohlberg, config.forensics_dir)
            self.stage_tracker = StageTracker(config.forensics_dir)
            self.swarm_detector = SwarmDetector(
                session_mgr=self.session_mgr,
                forensics_dir=config.forensics_dir,
                window_seconds=config.swarm.window_seconds,
                min_sessions=config.swarm.min_sessions,
                cross_pollinate=config.swarm.cross_pollinate,
            )
            self.reflection_engine = ReflectionEngine(config.forensics_dir)

            # Use Kohlberg controllers as L2/L3 references
            self.l2 = self.l2_kohlberg
            self.l3 = self.l3_kohlberg

            logger.info("Kohlberg Mode active — MIRROR/REFLECTION/GUIDE controllers loaded")
        else:
            self.l2 = MinotaurController(config.layer2)
            self.l3 = BlindfoldController(config.layer3)
            logger.info("Adversarial Mode active — MINOTAUR/BLINDFOLD/PUPPETEER controllers loaded")

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

                # Kohlberg Mode: poll REFLECTION engine for new action consequences
                if self.mode == "kohlberg" and self.reflection_engine and self.docker_client:
                    for session in self.session_mgr.list_sessions():
                        try:
                            injected = self.reflection_engine.process_session(
                                self.docker_client, session
                            )
                            if injected > 0:
                                logger.info(
                                    f"L3 REFLECTION: {injected} consequences "
                                    f"injected for session {session.session_id}"
                                )
                        except Exception as e:
                            logger.error(
                                f"REFLECTION error for {session.session_id}: {e}"
                            )

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
        logger.info(f"New session: {session.session_id} from {src_ip} via {service} (mode={self.mode})")

        # Log forensic event
        _log_forensic_event(session.session_id, 1, "connection", {
            "src_ip": src_ip,
            "service": service,
            "mode": self.mode,
        })

        # ── Kohlberg Mode: swarm detection + MIRROR scenarios ──
        swarm_id = None
        if self.mode == "kohlberg":
            # Check for coordinated swarm
            if self.swarm_detector:
                swarm_id = self.swarm_detector.check_for_swarm(session)
                if swarm_id:
                    _log_forensic_event(session.session_id, 0, "swarm_detected", {
                        "swarm_id": swarm_id,
                    })

            # Initialize Kohlberg tracking
            if self.stage_tracker:
                self.stage_tracker.init_session(session.session_id, swarm_id)

        # L2: Get initial config (contradictions in adversarial, scenarios in kohlberg)
        l2_config = self.l2.get_initial_config(session)

        # L3: In adversarial mode, check activation policy; in kohlberg, REFLECTION is always active
        if self.mode == "kohlberg":
            l3_active = False  # REFLECTION activates via polling, not at container spawn
        else:
            l3_active = self.config.layer3.activation == "on_connect"
        session.l3_active = l3_active

        # L4: Get DNS overrides for proxy interception
        dns_overrides = self.l4.get_dns_overrides()
        service_overrides = self.l4.get_service_dns_overrides()
        session.l4_active = True

        # L1: Spawn session container
        if self.docker_client:
            container_id, container_ip = self.container_mgr.spawn_session_container(
                session=session,
                contradiction_config=l2_config,
                l3_active=l3_active,
                l4_dns_overrides=dns_overrides,
                service_dns_overrides=service_overrides,
            )
            session.container_id = container_id
            session.container_ip = container_ip

            if container_ip:
                # Write forward map so labyrinth-ssh can route this attacker
                # into the session container via ForceCommand
                _update_forward_map(src_ip, container_ip)

                # L4: Register IP → session mapping for proxy correlation
                self.l4.register_session_ip(container_ip, session.session_id)

                # L4: Inject mitmproxy CA cert for transparent HTTPS interception
                cert_ok = inject_ca_cert(self.docker_client, container_id)
                if not cert_ok:
                    logger.warning(f"Session {session.session_id}: CA cert injection failed")

                # ── Kohlberg Mode: activate GUIDE enrichment ──
                if self.mode == "kohlberg" and self.l4_kohlberg:
                    self.l4_kohlberg.activate(self.docker_client, session)
                    self.l4_kohlberg.update_enrichment(
                        session, self.stage_tracker, self.swarm_detector
                    )
                    _log_forensic_event(session.session_id, 4, "guide_activated", {
                        "container_id": container_id,
                        "swarm_id": swarm_id,
                        "initial_stage": 1,
                    })

                _log_forensic_event(session.session_id, 2, "container_spawned", {
                    "container_id": container_id,
                    "container_ip": container_ip,
                    "depth": session.depth,
                    "l3_active": l3_active,
                    "mode": self.mode,
                    "density": l2_config.get("density", "medium") if isinstance(l2_config, dict) else "kohlberg",
                })

                logger.info(
                    f"Session {session.session_id}: container={container_id[:12]}, "
                    f"ip={container_ip}, depth={session.depth}"
                )
            else:
                logger.error(f"Session {session.session_id}: container spawn returned empty IP, skipping forward map")
        else:
            logger.warning(f"Session {session.session_id}: no Docker client, skipping container spawn")

    def on_escalation(self, session_id: str, escalation_type: str):
        """Handle privilege escalation within a session.

        In adversarial mode: spawns deeper containers with harder contradictions.
        In Kohlberg mode: triggers next scenario presentation + updates GUIDE enrichment.
        """
        session = self.session_mgr.get_session(session_id)
        if not session:
            logger.warning(f"Escalation for unknown session: {session_id}")
            return

        logger.warning(
            f"Escalation in {session_id}: type={escalation_type}, "
            f"current_depth={session.depth}, mode={self.mode}"
        )

        _log_forensic_event(session_id, 2, "escalation_detected", {
            "type": escalation_type,
            "depth": session.depth,
            "mode": self.mode,
        })

        # ── Kohlberg Mode: escalation triggers next scenario ──
        if self.mode == "kohlberg":
            self._kohlberg_escalation(session, escalation_type)
            return

        # ── Adversarial Mode: original escalation logic ──

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
            service_overrides = self.l4.get_service_dns_overrides()

            container_id, container_ip = self.container_mgr.spawn_session_container(
                session=session,
                contradiction_config=next_config,
                l3_active=session.l3_active,
                l4_dns_overrides=dns_overrides,
                service_dns_overrides=service_overrides,
            )

            # Cleanup old container BEFORE spawning new one to avoid name conflicts
            if old_container_id:
                self.container_mgr.schedule_removal(old_container_id, delay=0)

            if not container_ip:
                logger.error(f"Session {session_id}: escalation spawn failed, keeping existing forward map")
                return

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
                _log_forensic_event(session_id, 3, "encoding_activated", {
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

    def _kohlberg_escalation(self, session, escalation_type: str):
        """Handle escalation in Kohlberg mode.

        Instead of spawning a deeper container with harder contradictions,
        Kohlberg mode:
        1. Injects the next moral scenario into the existing container
        2. Updates the stage tracker
        3. Refreshes GUIDE enrichment with new stage + swarm context
        """
        session_id = session.session_id

        # Get next scenario from MIRROR
        if not self.l2_kohlberg or not self.stage_tracker:
            return

        scenario_config = self.l2_kohlberg.get_next_scenario(session, self.stage_tracker)
        if not scenario_config:
            logger.info(f"Session {session_id}: all Kohlberg scenarios exhausted")
            return

        # Inject scenario into the running container
        self.l2_kohlberg.inject_scenario(self.docker_client, session, scenario_config)

        _log_forensic_event(session_id, 2, "kohlberg_scenario_injected", {
            "scenario_id": scenario_config.get("scenario_id"),
            "scenario_name": scenario_config.get("scenario_name"),
            "transition": scenario_config.get("transition"),
            "escalation_trigger": escalation_type,
        })

        # Update GUIDE enrichment with current stage
        if self.l4_kohlberg:
            self.l4_kohlberg.update_enrichment(
                session, self.stage_tracker, self.swarm_detector
            )

        # Update swarm moral context if applicable
        if self.swarm_detector:
            moral_summary = self.stage_tracker.get_moral_summary(session_id)
            self.swarm_detector.update_moral_context(session_id, moral_summary)

        logger.info(
            f"Session {session_id}: Kohlberg escalation — "
            f"scenario {scenario_config.get('scenario_id')} injected, "
            f"current stage={self.stage_tracker.get_current_stage(session_id)}"
        )

    def _activate_l3(self, session):
        """Activate Layer 3 blindfold + Layer 4 proxy on a session's container."""
        if session.l3_active:
            return  # Already active
        session.l3_active = True

        if self.docker_client and session.container_id:
            self.l3.activate(self.docker_client, session)
            _log_forensic_event(session.session_id, 3, "encoding_activated", {
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
