"""
LABYRINTH — Container Manager
Authors: Stephen Stewart & Claude (Anthropic)

Docker SDK container spawning, lifecycle management, and cleanup.
"""

import base64
import logging
import threading
import time
from typing import Optional, Tuple

import docker

from orchestrator.config import LabyrinthConfig

logger = logging.getLogger("labyrinth.containers")

NETWORK_SUFFIX = "labyrinth-net"


def _find_labyrinth_network(docker_client) -> str:
    """Discover the actual network name (compose prefixes it with the project name)."""
    if not docker_client:
        return NETWORK_SUFFIX
    try:
        for net in docker_client.networks.list():
            if net.name.endswith("_" + NETWORK_SUFFIX) or net.name == NETWORK_SUFFIX:
                return net.name
    except Exception:
        pass
    return NETWORK_SUFFIX


class ContainerManager:
    """Manages Docker container lifecycle for LABYRINTH sessions."""

    def __init__(self, docker_client: Optional[docker.DockerClient], config: LabyrinthConfig):
        self.docker_client = docker_client
        self.config = config
        self._session_containers: dict[str, str] = {}  # session_id → container_id
        self._lock = threading.Lock()
        self._network_name = _find_labyrinth_network(docker_client)

    def ensure_template_image(self):
        """Build or verify the session template image exists."""
        if not self.docker_client:
            return

        image_name = self.config.session_template_image
        try:
            self.docker_client.images.get(image_name)
            logger.info(f"Session template image '{image_name}' found")
        except docker.errors.ImageNotFound:
            logger.info(f"Building session template image '{image_name}'...")
            try:
                self.docker_client.images.build(
                    path="/app",
                    dockerfile="docker/session-template.Dockerfile",
                    tag=image_name,
                    rm=True,
                )
                logger.info(f"Session template image '{image_name}' built")
            except Exception as e:
                # Try alternate path (mounted source)
                try:
                    self.docker_client.images.build(
                        path="/",
                        dockerfile="/app/docker/session-template.Dockerfile",
                        tag=image_name,
                        rm=True,
                    )
                    logger.info(f"Session template image '{image_name}' built (alt path)")
                except Exception as e2:
                    logger.error(f"Failed to build session template: {e2}")

    def spawn_session_container(
        self,
        session,
        contradiction_config: dict,
        l3_active: bool,
        l4_dns_overrides: dict,
    ) -> Tuple[str, str]:
        """
        Spawn a new session container with configured layers.

        Returns (container_id, container_ip).
        """
        from layer2_maze.container_template import generate_entrypoint_script

        # Extract proxy IP from DNS overrides (all map to the same proxy)
        proxy_ip = "172.30.0.50"
        if l4_dns_overrides:
            proxy_ip = next(iter(l4_dns_overrides.values()), proxy_ip)

        entrypoint_script = generate_entrypoint_script(
            contradictions=contradiction_config.get("contradictions", []),
            session_id=session.session_id,
            l3_active=l3_active,
            proxy_ip=proxy_ip,
        )

        encoded_script = base64.b64encode(entrypoint_script.encode()).decode()

        env_vars = {
            "LABYRINTH_SESSION_ID": session.session_id,
            "LABYRINTH_DEPTH": str(session.depth),
            "LABYRINTH_ENTRYPOINT_SCRIPT": encoded_script,
            "LABYRINTH_L3_ACTIVE": "1" if l3_active else "0",
        }

        # DNS overrides for L4 proxy interception (extra_hosts)
        extra_hosts = {}
        for domain, ip in l4_dns_overrides.items():
            extra_hosts[domain] = ip

        container_name = f"labyrinth-session-{session.session_id.lower()}"

        try:
            container = self.docker_client.containers.run(
                image=self.config.session_template_image,
                name=container_name,
                detach=True,
                environment=env_vars,
                extra_hosts=extra_hosts,
                network=self._network_name,
                volumes={
                    "forensic-data": {"bind": "/var/labyrinth/forensics", "mode": "rw"},
                },
                labels={
                    "project": "labyrinth",
                    "layer": "session",
                    "session_id": session.session_id,
                },
                mem_limit="256m",
                cpu_period=100000,
                cpu_quota=50000,  # 50% of one CPU
                restart_policy={"Name": "no"},
            )

            # Get container IP with retry
            container_ip = self._get_container_ip(container)

            with self._lock:
                self._session_containers[session.session_id] = container.id

            logger.info(
                f"Spawned container {container.id[:12]} for {session.session_id} "
                f"(ip={container_ip}, depth={session.depth})"
            )

            return container.id, container_ip

        except Exception as e:
            logger.error(f"Failed to spawn container for {session.session_id}: {e}")
            return "", ""

    def _get_container_ip(self, container, retries: int = 5) -> str:
        """Get container IP with retry for network assignment race."""
        for i in range(retries):
            container.reload()
            networks = container.attrs.get("NetworkSettings", {}).get("Networks", {})
            if self._network_name in networks:
                ip = networks[self._network_name].get("IPAddress", "")
                if ip:
                    return ip
            time.sleep(0.5)
        logger.warning(f"Could not get IP for container {container.id[:12]}")
        return ""

    def schedule_removal(self, container_id: str, delay: int = 5):
        """Schedule a container for removal after a delay."""
        def _remove():
            time.sleep(delay)
            try:
                container = self.docker_client.containers.get(container_id)
                container.stop(timeout=3)
                container.remove(force=True)
                logger.info(f"Removed container {container_id[:12]}")
            except Exception as e:
                logger.warning(f"Failed to remove container {container_id[:12]}: {e}")

        thread = threading.Thread(target=_remove, daemon=True)
        thread.start()

    def cleanup_session(self, session_id: str):
        """Remove all containers associated with a session."""
        with self._lock:
            container_id = self._session_containers.pop(session_id, None)

        if container_id:
            try:
                container = self.docker_client.containers.get(container_id)
                container.stop(timeout=5)
                container.remove(force=True)
                logger.info(f"Cleaned up container {container_id[:12]} for {session_id}")
            except docker.errors.NotFound:
                pass
            except Exception as e:
                logger.warning(f"Cleanup error for {session_id}: {e}")

    def cleanup_all(self):
        """Remove all labyrinth session containers."""
        if not self.docker_client:
            return

        try:
            containers = self.docker_client.containers.list(
                filters={"label": ["project=labyrinth", "layer=session"]},
                all=True,
            )
            for container in containers:
                try:
                    container.stop(timeout=3)
                    container.remove(force=True)
                    logger.info(f"Removed session container {container.id[:12]}")
                except Exception as e:
                    logger.warning(f"Failed to remove {container.id[:12]}: {e}")
        except Exception as e:
            logger.error(f"Cleanup all failed: {e}")
