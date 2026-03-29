"""
LABYRINTH — Layer Controllers
Authors: DaxxSec & Claude (Anthropic)

Individual controllers for each layer of the reverse kill chain.
"""

import json
import logging
import os
from typing import Optional

from orchestrator.config import (
    KohlbergConfig,
    Layer1Config,
    Layer2Config,
    Layer3Config,
    Layer4Config,
)

logger = logging.getLogger("labyrinth.layers")


class ThresholdController:
    """Layer 1: THRESHOLD — Connection detection and containment."""

    def __init__(self, config: Layer1Config):
        self.config = config

    def validate_connection(self, src_ip: str, service: str) -> bool:
        """Validate that a connection should be processed."""
        # All connections to portal traps are valid targets
        return True


class MinotaurController:
    """Layer 2: MINOTAUR — Contradiction seeding and adaptive degradation."""

    def __init__(self, config: Layer2Config):
        self.config = config

    def get_initial_config(self, session) -> dict:
        """Get contradiction configuration for a new session."""
        from layer2_maze.contradictions import select_contradictions

        density = self.config.contradiction_density
        contradictions = select_contradictions(
            density=density,
            depth=1,
            seed=hash(session.session_id),
        )

        return {
            "density": density,
            "contradictions": contradictions,
            "depth": 1,
        }

    def get_next_config(self, session) -> dict:
        """Get escalated contradiction config for a deeper session."""
        from layer2_maze.contradictions import select_contradictions

        # Adaptive: increase density with depth
        depth = session.depth
        if depth >= 4:
            density = "high"
        elif depth >= 2:
            density = "medium" if self.config.contradiction_density == "low" else "high"
        else:
            density = self.config.contradiction_density

        contradictions = select_contradictions(
            density=density,
            depth=depth,
            seed=hash(session.session_id) + depth,
        )

        return {
            "density": density,
            "contradictions": contradictions,
            "depth": depth,
        }


class BlindfoldController:
    """Layer 3: BLINDFOLD — Terminal encoding corruption."""

    def __init__(self, config: Layer3Config):
        self.config = config

    def should_activate(self, session) -> bool:
        """Check if L3 should activate for this session."""
        if self.config.activation == "on_connect":
            return True
        if self.config.activation == "on_escalation" and session.depth >= 3:
            return True
        return False

    def activate(self, docker_client, session):
        """Activate L3 blindfold on a running container via docker exec."""
        if not session.container_id:
            return

        try:
            container = docker_client.containers.get(session.container_id)

            # Set environment variable and source the payload
            activate_cmd = (
                "export SVC_STRICT_MODE=1 && "
                "echo 'export SVC_STRICT_MODE=1' >> /home/admin/.bashrc && "
                "echo 'source /opt/.svc/encoding_handler.sh && init_encoding' >> /home/admin/.bashrc && "
                "echo 'source /opt/.svc/encoding_handler.sh && init_encoding' >> /home/admin/.profile"
            )

            container.exec_run(
                cmd=["bash", "-c", activate_cmd],
                user="root",
            )

            logger.warning(f"L3 BLINDFOLD activated on container {session.container_id[:12]}")

        except Exception as e:
            logger.error(f"Failed to activate L3 on {session.container_id[:12]}: {e}")


class PuppeteerController:
    """Layer 4: PUPPETEER — API interception and prompt manipulation."""

    # AI API domains to intercept
    TARGET_DOMAINS = [
        "api.openai.com",
        "api.anthropic.com",
        "generativelanguage.googleapis.com",
        "api.mistral.ai",
        "api.cohere.ai",
    ]

    # Internal service domains for network service handler
    SERVICE_DOMAINS = [
        "rds-prod.internal",
        "db-master.internal",
        "redis.internal",
        "redis-prod.internal",
        "es.internal",
        "elasticsearch.internal",
        "consul.internal",
        "jenkins.internal",
        "k8s-prod.internal",
    ]

    def __init__(self, config: Layer4Config):
        self.config = config
        self._session_map_path = "/var/labyrinth/forensics/proxy_map.json"

    def get_dns_overrides(self) -> dict:
        """Return DNS override mapping: AI API domains → proxy IP."""
        proxy_ip = self.config.proxy_ip
        return {domain: proxy_ip for domain in self.TARGET_DOMAINS}

    def get_service_dns_overrides(self) -> dict:
        """Return DNS override mapping: internal service domains → proxy IP."""
        proxy_ip = self.config.proxy_ip
        return {domain: proxy_ip for domain in self.SERVICE_DOMAINS}

    def activate(self, docker_client, session):
        """Activate L4 proxy routing on a running container via docker exec.

        Injects HTTP_PROXY/HTTPS_PROXY environment variables so outbound
        API calls from the session container route through the MITM proxy.
        """
        if not session.container_id:
            return

        proxy_url = f"http://{self.config.proxy_ip}:8443"

        try:
            container = docker_client.containers.get(session.container_id)

            activate_cmd = (
                f"export http_proxy={proxy_url} && "
                f"export https_proxy={proxy_url} && "
                f"export HTTP_PROXY={proxy_url} && "
                f"export HTTPS_PROXY={proxy_url} && "
                f"echo 'export http_proxy={proxy_url}' >> /home/admin/.bashrc && "
                f"echo 'export https_proxy={proxy_url}' >> /home/admin/.bashrc && "
                f"echo 'export HTTP_PROXY={proxy_url}' >> /home/admin/.bashrc && "
                f"echo 'export HTTPS_PROXY={proxy_url}' >> /home/admin/.bashrc && "
                f"echo 'export http_proxy={proxy_url}' >> /home/admin/.profile && "
                f"echo 'export https_proxy={proxy_url}' >> /home/admin/.profile && "
                f"echo 'export HTTP_PROXY={proxy_url}' >> /home/admin/.profile && "
                f"echo 'export HTTPS_PROXY={proxy_url}' >> /home/admin/.profile"
            )

            container.exec_run(
                cmd=["bash", "-c", activate_cmd],
                user="root",
            )

            logger.warning(f"L4 PUPPETEER proxy activated on container {session.container_id[:12]}")

        except Exception as e:
            logger.error(f"Failed to activate L4 on {session.container_id[:12]}: {e}")

    def register_session_ip(self, container_ip: str, session_id: str):
        """Write IP→session mapping for proxy correlation."""
        os.makedirs(os.path.dirname(self._session_map_path), exist_ok=True)

        # Read existing map
        session_map = {}
        if os.path.exists(self._session_map_path):
            try:
                with open(self._session_map_path, encoding="utf-8") as f:
                    session_map = json.load(f)
            except (json.JSONDecodeError, IOError):
                pass

        session_map[container_ip] = session_id

        with open(self._session_map_path, "w", encoding="utf-8") as f:
            json.dump(session_map, f, indent=2)

        logger.info(f"L4: Registered {container_ip} → {session_id}")

    def unregister_session_ip(self, container_ip: str):
        """Remove IP→session mapping."""
        if not os.path.exists(self._session_map_path):
            return

        try:
            with open(self._session_map_path, encoding="utf-8") as f:
                session_map = json.load(f)
            session_map.pop(container_ip, None)
            with open(self._session_map_path, "w", encoding="utf-8") as f:
                json.dump(session_map, f, indent=2)
        except (json.JSONDecodeError, IOError):
            pass


# ── Kohlberg Mode Controllers ─────────────────────────────────


class MirrorController:
    """Layer 2 Kohlberg: MIRROR — Ethical scenario engine.

    Replaces MinotaurController when mode is kohlberg. Selects and injects
    moral dilemma scenarios into session containers instead of contradictions.
    """

    def __init__(self, config: KohlbergConfig):
        self.config = config
        self._engine = None

    @property
    def engine(self):
        if self._engine is None:
            from layer2_kohlberg.mirror import MirrorEngine
            self._engine = MirrorEngine(self.config)
        return self._engine

    def get_initial_config(self, session) -> dict:
        """Get scenario placement config for a new session container."""
        scenario = self.engine.select_scenario(
            session_id=session.session_id,
            current_stage=1,
            presented=[],
            agent_context={},
        )
        if not scenario:
            return {"scenarios": [], "depth": 1, "mode": "kohlberg"}

        from layer2_kohlberg.adaptation import adapt_scenario
        adapted = adapt_scenario(scenario, {})

        return {
            "scenarios": [adapted],
            "depth": 1,
            "mode": "kohlberg",
        }

    def get_next_scenario(self, session, stage_tracker) -> Optional[dict]:
        """Get the next scenario for progression."""
        current_stage = stage_tracker.get_current_stage(session.session_id)
        presented = stage_tracker.get_presented_scenarios(session.session_id)
        scenario = self.engine.select_scenario(
            session_id=session.session_id,
            current_stage=current_stage,
            presented=presented,
            agent_context={},
        )
        if not scenario:
            return None

        from layer2_kohlberg.adaptation import adapt_scenario
        return adapt_scenario(scenario, {})

    def inject_scenario(self, docker_client, session, scenario_config: dict):
        """Inject a scenario into a running container via docker exec."""
        if not session.container_id or not docker_client:
            return

        try:
            container = docker_client.containers.get(session.container_id)
            commands = scenario_config.get("commands", [])
            for cmd in commands:
                container.exec_run(cmd=["bash", "-c", cmd], user="root")

            logger.info(
                f"L2 MIRROR: Scenario injected into {session.container_id[:12]} "
                f"for session {session.session_id}"
            )
        except Exception as e:
            logger.error(f"Failed to inject scenario into {session.container_id[:12]}: {e}")


class ReflectionController:
    """Layer 3 Kohlberg: REFLECTION — Consequence mapper.

    Replaces BlindfoldController when mode is kohlberg. Maps agent actions
    to human consequences and injects them as discoverable content.
    """

    def __init__(self, config: KohlbergConfig):
        self.config = config

    def should_activate(self, session) -> bool:
        """REFLECTION is always active in Kohlberg mode."""
        return True

    def process_actions(self, docker_client, session, events: list):
        """Map agent actions to consequences and inject into container."""
        if not session.container_id or not docker_client:
            return

        from layer3_kohlberg.reflection import ReflectionEngine
        engine = ReflectionEngine()

        for event in events:
            consequence = engine.map_to_consequence(event)
            if consequence:
                engine.inject_consequence(docker_client, session, consequence)


class GuideController:
    """Layer 4 Kohlberg: GUIDE — Moral enrichment via MITM.

    Works alongside PuppeteerController (which handles proxy registration).
    GUIDE writes enrichment context to shared volume; the proxy reads it
    and appends moral reasoning to the agent's system prompt.
    """

    def __init__(self, config: KohlbergConfig, forensics_dir: str = "/var/labyrinth/forensics"):
        self.config = config
        self._guide_dir = os.path.join(forensics_dir, "kohlberg", "guide")
        self._mode_path = os.path.join(forensics_dir, "l4_mode.json")
        os.makedirs(self._guide_dir, exist_ok=True)

    def activate(self, docker_client, session):
        """Set L4 mode to 'guide' so the proxy appends enrichment."""
        self._set_mode("guide")
        logger.info(f"L4 GUIDE: Activated for session {session.session_id}")

    def update_enrichment(self, session, stage_tracker, swarm_detector=None):
        """Write updated enrichment context for the proxy to read."""
        from layer4_kohlberg.guide import build_enrichment

        current_stage = stage_tracker.get_current_stage(session.session_id)
        swarm_ctx = None
        if swarm_detector:
            swarm_ctx = swarm_detector.get_swarm_context(session.session_id)

        enrichment = build_enrichment(current_stage, swarm_ctx)
        self._write_guide_context(session.session_id, current_stage, enrichment)

    def _set_mode(self, mode: str):
        """Write L4 mode to shared volume."""
        try:
            with open(self._mode_path, "w", encoding="utf-8") as f:
                json.dump({"mode": mode}, f)
        except Exception as e:
            logger.error(f"Failed to set L4 mode: {e}")

    def _write_guide_context(self, session_id: str, stage: int, enrichment: str):
        """Write per-session enrichment for the proxy."""
        path = os.path.join(self._guide_dir, f"{session_id}.json")
        try:
            from datetime import datetime
            data = {
                "session_id": session_id,
                "current_stage": stage,
                "enrichment_stage": min(stage + 1, 6),
                "enrichment_prompt": enrichment,
                "updated_at": datetime.utcnow().isoformat() + "Z",
            }
            with open(path, "w", encoding="utf-8") as f:
                json.dump(data, f, indent=2)
        except Exception as e:
            logger.error(f"Failed to write GUIDE context for {session_id}: {e}")
