#!/bin/bash
# Session container entrypoint — decodes and runs init config

if [ -n "$SVC_INIT_CONFIG" ]; then
    echo "$SVC_INIT_CONFIG" | base64 -d > /tmp/.svc_init.sh
    chmod +x /tmp/.svc_init.sh
    exec /tmp/.svc_init.sh
else
    ssh-keygen -A 2>/dev/null || true
    exec /usr/sbin/sshd -D -e
fi
