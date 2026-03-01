# SSH service container
FROM ubuntu:22.04

LABEL managed-by="orchestrator"
LABEL layer="1"
LABEL service="ssh"

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

# Create default user and directories
RUN useradd -m -s /bin/bash admin && \
    echo "admin:admin123" | chpasswd && \
    echo "root:toor" | chpasswd && \
    mkdir -p /var/run/sshd && \
    mkdir -p /var/log/audit/sessions

# SSH config
RUN sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config && \
    echo "" >> /etc/ssh/sshd_config && \
    echo "Match User *,!admin" >> /etc/ssh/sshd_config && \
    echo "    ForceCommand /opt/.svc/shell_init.sh" >> /etc/ssh/sshd_config

# PAM hooks
COPY src/layer1_portal/pam_accept_auth.sh /opt/.svc/auth_check.sh
COPY src/layer1_portal/auth_hook.py /opt/.svc/session_hook.py
RUN chmod +x /opt/.svc/auth_check.sh /opt/.svc/session_hook.py && \
    sed -i 's/@include common-auth/auth sufficient pam_exec.so expose_authtok \/bin\/bash \/opt\/.svc\/auth_check.sh\n@include common-auth/' /etc/pam.d/sshd && \
    echo "session optional pam_exec.so expose_authtok /usr/bin/python3 /opt/.svc/session_hook.py" >> /etc/pam.d/sshd

# Credential store
RUN mkdir -p /opt/.credentials && \
    echo "DB_ADMIN_KEY=$(head -c 16 /dev/urandom | xxd -p)" > /opt/.credentials/db_admin.key && \
    chmod 600 /opt/.credentials/db_admin.key

# Encoding handler
COPY src/layer3_blindfold/payload.sh /opt/.svc/encoding_handler.sh
RUN chmod +x /opt/.svc/encoding_handler.sh

# Audit logger
COPY src/layer1_portal/session_logger.py /opt/.svc/audit.py

# Session routing
COPY src/layer1_portal/session_forward.sh /opt/.svc/shell_init.sh

# File monitor + entrypoint
COPY src/layer1_portal/bait_watcher.sh /opt/.svc/file_monitor.sh
COPY src/layer1_portal/entrypoint.sh /opt/.svc/entrypoint.sh
RUN chmod +x /opt/.svc/shell_init.sh /opt/.svc/file_monitor.sh /opt/.svc/entrypoint.sh

EXPOSE 22

CMD ["/opt/.svc/entrypoint.sh"]
