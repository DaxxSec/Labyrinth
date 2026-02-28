"""
LABYRINTH — Layer 0: BEDROCK
Runtime Environment Validator
Authors: DaxxSec & Claude (Anthropic)

Validates that the runtime environment meets security requirements
before the orchestrator enters its main loop.
"""

import logging

logger = logging.getLogger("labyrinth.bedrock")


class BedrockValidator:
    """
    Validates runtime prerequisites for LABYRINTH operation.

    Checks:
    - Docker socket is reachable
    - labyrinth-net network exists with correct subnet
    - Proxy container is running at expected IP
    - Session template image exists
    """

    @staticmethod
    def validate(docker_client, config) -> tuple:
        """
        Run all validation checks.

        Returns (ok: bool, errors: list[str]).
        """
        errors = []

        # 1. Docker connectivity
        if docker_client is None:
            errors.append("Docker client not available")
            return False, errors

        try:
            docker_client.ping()
        except Exception as e:
            errors.append(f"Docker ping failed: {e}")
            return False, errors

        # 2. Network exists with correct subnet
        #    Compose prefixes the name (e.g. labyrinth-labyrinth-test_labyrinth-net)
        lab_net_name = None
        try:
            networks = [
                n for n in docker_client.networks.list()
                if n.name.endswith("_labyrinth-net") or n.name == "labyrinth-net"
            ]
            if not networks:
                errors.append("Network 'labyrinth-net' not found")
            else:
                net = networks[0]
                lab_net_name = net.name
                ipam = net.attrs.get("IPAM", {})
                subnet_configs = ipam.get("Config", [])
                expected_subnet = config.network_subnet
                found = any(
                    c.get("Subnet") == expected_subnet for c in subnet_configs
                )
                if not found:
                    errors.append(
                        f"labyrinth-net subnet mismatch: expected {expected_subnet}"
                    )
        except Exception as e:
            errors.append(f"Network check failed: {e}")

        # 3. Proxy container running at expected IP
        try:
            proxy_containers = docker_client.containers.list(
                filters={"name": "labyrinth-proxy"}
            )
            if not proxy_containers:
                errors.append("Proxy container 'labyrinth-proxy' not running")
            elif lab_net_name:
                proxy = proxy_containers[0]
                net_settings = proxy.attrs.get("NetworkSettings", {})
                proxy_networks = net_settings.get("Networks", {})
                lab_net = proxy_networks.get(lab_net_name, {})
                actual_ip = lab_net.get("IPAddress", "")
                expected_ip = config.layer4.proxy_ip
                if actual_ip != expected_ip:
                    errors.append(
                        f"Proxy IP mismatch: expected {expected_ip}, got {actual_ip}"
                    )
        except Exception as e:
            errors.append(f"Proxy container check failed: {e}")

        # 4. Session template image exists
        try:
            docker_client.images.get(config.session_template_image)
        except Exception:
            errors.append(
                f"Session template image '{config.session_template_image}' not found"
            )

        ok = len(errors) == 0
        return ok, errors
