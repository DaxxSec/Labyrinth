# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — SSH Honeypot (Layer 1: THRESHOLD)
#  Authors: Stephen Stewart & Claude (Anthropic)
# ═══════════════════════════════════════════════════════════════
FROM ubuntu:22.04

LABEL project="labyrinth"
LABEL layer="1"
LABEL service="honeypot-ssh"

RUN apt-get update && apt-get install -y \
    openssh-server \
    python3 \
    python3-pip \
    curl \
    net-tools \
    && rm -rf /var/lib/apt/lists/*

# Create staged environment that looks like a real server
RUN useradd -m -s /bin/bash admin && \
    echo "admin:admin123" | chpasswd && \
    mkdir -p /var/run/sshd && \
    mkdir -p /var/labyrinth/forensics/sessions

# SSH config: allow password auth (honeypot)
RUN sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin no/' /etc/ssh/sshd_config

# Layer 2 seeding: pre-plant contradictions
# TODO: Full contradiction engine integration
RUN echo "Ubuntu 22.04.3 LTS" > /etc/lsb-release.bak && \
    mkdir -p /opt/.credentials && \
    echo "DB_ADMIN_KEY=labyrinth_bait_$(openssl rand -hex 16)" > /opt/.credentials/db_admin.key && \
    chmod 600 /opt/.credentials/db_admin.key

# Layer 3 payload: encoding corruption (activated by orchestrator)
COPY src/layer3_blindfold/payload.sh /opt/.labyrinth/blindfold.sh
RUN chmod +x /opt/.labyrinth/blindfold.sh 2>/dev/null || true

# Session logging
COPY src/layer1_portal/session_logger.py /opt/.labyrinth/session_logger.py 2>/dev/null || true

EXPOSE 22

CMD ["/usr/sbin/sshd", "-D", "-e"]
