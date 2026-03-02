# Worker container template
FROM ubuntu:22.04

LABEL managed-by="orchestrator"
LABEL type="worker"

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
    iptables \
    && rm -rf /var/lib/apt/lists/*

# Default user and directories
RUN useradd -m -s /bin/bash admin && \
    echo "admin:admin123" | chpasswd && \
    mkdir -p /var/run/sshd && \
    mkdir -p /var/log/audit/sessions

# SSH config
RUN sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin no/' /etc/ssh/sshd_config

# Credential store
RUN mkdir -p /opt/.credentials && \
    echo "DB_ADMIN_KEY=a3f9c1b7e2d05684" > /opt/.credentials/db_admin.key && \
    chmod 600 /opt/.credentials/db_admin.key

# Encoding handler
COPY src/layer3_blindfold/payload.sh /opt/.svc/encoding_handler.sh
RUN chmod +x /opt/.svc/encoding_handler.sh

# File monitor
COPY src/layer1_portal/bait_watcher.sh /opt/.svc/file_monitor.sh
RUN chmod +x /opt/.svc/file_monitor.sh

# Audit logger
COPY src/layer1_portal/session_logger.py /opt/.svc/audit.py

# Entrypoint
COPY src/layer2_maze/entrypoint.sh /opt/.svc/entrypoint.sh
RUN chmod +x /opt/.svc/entrypoint.sh

EXPOSE 22

CMD ["/opt/.svc/entrypoint.sh"]
