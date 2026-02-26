"""
LABYRINTH — Orchestrator Entry Point
Authors: Stephen Stewart & Claude (Anthropic)

Usage: python3 -m orchestrator
"""

import logging
import os
import signal
import sys

from orchestrator import LabyrinthOrchestrator
from orchestrator.config import LabyrinthConfig

LOG_LEVEL = os.environ.get("LABYRINTH_LOG_LEVEL", "INFO").upper()

logging.basicConfig(
    level=getattr(logging, LOG_LEVEL, logging.INFO),
    format="%(asctime)s [%(name)s] %(levelname)s: %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
logger = logging.getLogger("labyrinth")


def main():
    logger.info("=" * 60)
    logger.info("  LABYRINTH Orchestrator starting")
    logger.info("=" * 60)

    config = LabyrinthConfig.load()
    logger.info(f"Config loaded: L2 density={config.layer2.contradiction_density}, "
                f"L3 activation={config.layer3.activation}, "
                f"L4 mode={config.layer4.mode}")

    orchestrator = LabyrinthOrchestrator(config)

    # Graceful shutdown
    def shutdown(signum, frame):
        logger.info(f"Received signal {signum}, shutting down...")
        orchestrator.shutdown()
        sys.exit(0)

    signal.signal(signal.SIGTERM, shutdown)
    signal.signal(signal.SIGINT, shutdown)

    try:
        orchestrator.run()
    except KeyboardInterrupt:
        logger.info("Interrupted, shutting down...")
        orchestrator.shutdown()


if __name__ == "__main__":
    main()
