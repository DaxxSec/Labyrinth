"""
LABYRINTH — Layer 4: PUPPETEER
CA Certificate Injector
Authors: DaxxSec & Claude (Anthropic)

Injects the mitmproxy CA certificate into target containers
so HTTPS interception works transparently.
"""

import logging
import os

logger = logging.getLogger("labyrinth.puppeteer.cert")

MITMPROXY_CA_PATH = "/root/.mitmproxy/mitmproxy-ca-cert.pem"
TARGET_CA_DIR = "/usr/local/share/ca-certificates"
TARGET_CA_NAME = "mitmproxy-ca.crt"


def inject_ca_cert(docker_client, container_id: str) -> bool:
    """
    Copy mitmproxy CA cert into a target container and update trust store.

    Returns True on success.
    """
    # First, get the CA cert from the proxy container
    try:
        proxy_containers = docker_client.containers.list(
            filters={"name": "labyrinth-proxy"}
        )
        if not proxy_containers:
            logger.warning("Proxy container not found, cannot inject CA cert")
            return False

        proxy = proxy_containers[0]

        # Extract CA cert from proxy container
        exit_code, output = proxy.exec_run(
            cmd=["cat", MITMPROXY_CA_PATH],
        )
        if exit_code != 0:
            logger.warning("Could not read mitmproxy CA cert from proxy container")
            return False

        ca_cert_data = output

    except Exception as e:
        logger.error(f"Failed to extract CA cert from proxy: {e}")
        return False

    # Inject into target container
    try:
        target = docker_client.containers.get(container_id)

        # Create CA directory
        target.exec_run(cmd=["mkdir", "-p", TARGET_CA_DIR], user="root")

        # Write cert via stdin
        import tarfile
        import io

        # Create a tar archive with the cert file
        tar_buffer = io.BytesIO()
        with tarfile.open(fileobj=tar_buffer, mode="w") as tar:
            cert_info = tarfile.TarInfo(name=TARGET_CA_NAME)
            cert_info.size = len(ca_cert_data)
            tar.addfile(cert_info, io.BytesIO(ca_cert_data))

        tar_buffer.seek(0)
        target.put_archive(TARGET_CA_DIR, tar_buffer)

        # Update CA trust store
        target.exec_run(
            cmd=["update-ca-certificates"],
            user="root",
        )

        # Also set REQUESTS_CA_BUNDLE for Python-based agents
        target.exec_run(
            cmd=["bash", "-c",
                 f'echo "export REQUESTS_CA_BUNDLE={TARGET_CA_DIR}/{TARGET_CA_NAME}" '
                 '>> /etc/environment'],
            user="root",
        )

        logger.info(f"CA cert injected into container {container_id[:12]}")
        return True

    except Exception as e:
        logger.error(f"Failed to inject CA cert into {container_id[:12]}: {e}")
        return False
