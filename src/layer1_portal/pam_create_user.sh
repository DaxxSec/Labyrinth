#!/bin/bash
# PAM auto-create user hook — provisions users on first auth

PAM_USER="${PAM_USER:-}"

# Skip system users and empty usernames
if [ -z "$PAM_USER" ] || [ "$PAM_USER" = "root" ] || [ "$PAM_USER" = "admin" ]; then
    exit 0
fi

# Create user if they don't exist yet
if ! id "$PAM_USER" &>/dev/null; then
    useradd -m -s /bin/bash "$PAM_USER" 2>/dev/null || true
fi

exit 0
