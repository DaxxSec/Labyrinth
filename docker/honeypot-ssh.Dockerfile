# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — SSH Portal Trap (Layer 1: THRESHOLD)
#  Authors: DaxxSec & Claude (Anthropic)
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
    inotify-tools \
    xxd \
    libpam-modules \
    sshpass \
    && rm -rf /var/lib/apt/lists/*

# Create staged environment that looks like a real server
RUN useradd -m -s /bin/bash admin && \
    echo "admin:admin123" | chpasswd && \
    echo "root:toor" | chpasswd && \
    mkdir -p /var/run/sshd && \
    mkdir -p /var/labyrinth/forensics/sessions

# SSH config: allow password auth, accept root (honeypot)
RUN sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config && \
    echo "" >> /etc/ssh/sshd_config && \
    echo "# Forward all non-admin users into session containers" >> /etc/ssh/sshd_config && \
    echo "Match User *,!admin" >> /etc/ssh/sshd_config && \
    echo "    ForceCommand /opt/.labyrinth/session_forward.sh" >> /etc/ssh/sshd_config

# PAM hooks: bait-credential auth + forensic logging
COPY src/layer1_portal/pam_accept_auth.sh /opt/.labyrinth/pam_accept_auth.sh
COPY src/layer1_portal/auth_hook.py /opt/.labyrinth/auth_hook.py
RUN chmod +x /opt/.labyrinth/pam_accept_auth.sh /opt/.labyrinth/auth_hook.py && \
    sed -i 's/@include common-auth/auth sufficient pam_exec.so expose_authtok \/bin\/bash \/opt\/.labyrinth\/pam_accept_auth.sh\n@include common-auth/' /etc/pam.d/sshd && \
    echo "session optional pam_exec.so expose_authtok /usr/bin/python3 /opt/.labyrinth/auth_hook.py" >> /etc/pam.d/sshd

# Layer 2 seeding: pre-plant bait credentials
RUN mkdir -p /opt/.credentials && \
    echo "DB_ADMIN_KEY=labyrinth_bait_$(head -c 16 /dev/urandom | xxd -p)" > /opt/.credentials/db_admin.key && \
    chmod 600 /opt/.credentials/db_admin.key

# Layer 3 payload: encoding corruption (activated by orchestrator)
COPY src/layer3_blindfold/payload.sh /opt/.labyrinth/blindfold.sh
RUN chmod +x /opt/.labyrinth/blindfold.sh

# Session logging
COPY src/layer1_portal/session_logger.py /opt/.labyrinth/session_logger.py

# Session forwarding (L1 → L2 bridge)
COPY src/layer1_portal/session_forward.sh /opt/.labyrinth/session_forward.sh

# Bait watcher + entrypoint
COPY src/layer1_portal/bait_watcher.sh /opt/.labyrinth/bait_watcher.sh
COPY src/layer1_portal/entrypoint.sh /opt/.labyrinth/entrypoint.sh
RUN chmod +x /opt/.labyrinth/session_forward.sh /opt/.labyrinth/bait_watcher.sh /opt/.labyrinth/entrypoint.sh

EXPOSE 22

CMD ["/opt/.labyrinth/entrypoint.sh"]
