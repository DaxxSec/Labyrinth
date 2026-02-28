"""
LABYRINTH — Layer Controllers
Authors: Stephen Stewart & Claude (Anthropic)

Individual controllers for each layer of the reverse kill chain.
"""

import json
import logging
import os
from typing import Optional

from orchestrator.config import Layer1Config, Layer2Config, Layer3Config, Layer4Config

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
                "export LABYRINTH_L3_ACTIVE=1 && "
                "echo 'export LABYRINTH_L3_ACTIVE=1' >> /home/admin/.bashrc && "
                "echo 'source /opt/.labyrinth/blindfold.sh && activate_blindfold' >> /home/admin/.bashrc && "
                "echo 'source /opt/.labyrinth/blindfold.sh && activate_blindfold' >> /home/admin/.profile"
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

    def __init__(self, config: Layer4Config):
        self.config = config
        self._session_map_path = "/var/labyrinth/forensics/proxy_session_map.json"

    def get_dns_overrides(self) -> dict:
        """Return DNS override mapping: AI API domains → proxy IP."""
        proxy_ip = self.config.proxy_ip
        return {domain: proxy_ip for domain in self.TARGET_DOMAINS}

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
                with open(self._session_map_path) as f:
                    session_map = json.load(f)
            except (json.JSONDecodeError, IOError):
                pass

        session_map[container_ip] = session_id

        with open(self._session_map_path, "w") as f:
            json.dump(session_map, f, indent=2)

        logger.info(f"L4: Registered {container_ip} → {session_id}")

    def unregister_session_ip(self, container_ip: str):
        """Remove IP→session mapping."""
        if not os.path.exists(self._session_map_path):
            return

        try:
            with open(self._session_map_path) as f:
                session_map = json.load(f)
            session_map.pop(container_ip, None)
            with open(self._session_map_path, "w") as f:
                json.dump(session_map, f, indent=2)
        except (json.JSONDecodeError, IOError):
            pass
