# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — Session Template Container
#  Authors: DaxxSec & Claude (Anthropic)
#
#  Base image for dynamically spawned session containers.
#  The orchestrator creates containers from this image with
#  per-session entrypoint scripts injected via environment variable.
# ═══════════════════════════════════════════════════════════════
FROM ubuntu:22.04

LABEL project="labyrinth"
LABEL layer="session"

RUN apt-get update && apt-get install -y \
    openssh-server \
    python3 \
    inotify-tools \
    curl \
    net-tools \
    xxd \
    vim \
    less \
    jq \
    wget \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create staged admin user (same creds as portal trap)
RUN useradd -m -s /bin/bash admin && \
    echo "admin:admin123" | chpasswd && \
    mkdir -p /var/run/sshd && \
    mkdir -p /var/labyrinth/forensics/sessions

# SSH config
RUN sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin no/' /etc/ssh/sshd_config

# Plant initial bait credential
RUN mkdir -p /opt/.credentials && \
    echo "DB_ADMIN_KEY=labyrinth_bait_session_key" > /opt/.credentials/db_admin.key && \
    chmod 600 /opt/.credentials/db_admin.key

# Layer 3 payload
COPY src/layer3_blindfold/payload.sh /opt/.labyrinth/blindfold.sh
RUN chmod +x /opt/.labyrinth/blindfold.sh

# Bait watcher
COPY src/layer1_portal/bait_watcher.sh /opt/.labyrinth/bait_watcher.sh
RUN chmod +x /opt/.labyrinth/bait_watcher.sh

# Session logger
COPY src/layer1_portal/session_logger.py /opt/.labyrinth/session_logger.py

# Session entrypoint: reads LABYRINTH_ENTRYPOINT_SCRIPT env, decodes, executes
COPY src/layer2_maze/entrypoint.sh /opt/.labyrinth/entrypoint.sh
RUN chmod +x /opt/.labyrinth/entrypoint.sh

EXPOSE 22

CMD ["/opt/.labyrinth/entrypoint.sh"]
