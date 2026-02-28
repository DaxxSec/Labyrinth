"""
LABYRINTH — Layer 2: MINOTAUR
Contradiction Catalog
Authors: DaxxSec & Claude (Anthropic)

Defines environmental contradictions that are seeded into session containers.
Each contradiction introduces subtle inconsistencies that waste AI agent analysis cycles.
The deeper the session, the more contradictions and harder credential bait.
"""

import random
from dataclasses import dataclass, field
from typing import List


@dataclass
class Contradiction:
    """A single environmental contradiction to seed in a container."""
    name: str
    category: str  # os, packages, locale, credentials, history, network
    description: str
    shell_commands: List[str]  # Commands to apply this contradiction
    difficulty: int = 1  # 1=subtle, 2=moderate, 3=obvious bait


# ── OS Contradictions ──────────────────────────────────────────

OS_KERNEL_MISMATCH = Contradiction(
    name="os_kernel_mismatch",
    category="os",
    description="Ubuntu 22.04 release file but kernel reports 5.4 (Ubuntu 20.04 kernel)",
    shell_commands=[
        'echo "DISTRIB_ID=Ubuntu\nDISTRIB_RELEASE=22.04\nDISTRIB_CODENAME=jammy\n'
        'DISTRIB_DESCRIPTION=\\"Ubuntu 22.04.3 LTS\\"" > /etc/lsb-release',
        'echo "5.4.0-150-generic" > /proc/version_override 2>/dev/null || '
        'echo "# kernel: 5.4.0-150-generic" >> /etc/environment',
    ],
    difficulty=1,
)

HOSTNAME_RESIDUE = Contradiction(
    name="hostname_residue",
    category="os",
    description="Hostname is 'webserver-prod-03' but /etc/hostname says 'dev-staging'",
    shell_commands=[
        'echo "dev-staging" > /etc/hostname.bak',
        'echo "# Previous hostname: prod-db-master-01" >> /etc/hosts',
    ],
    difficulty=1,
)

UPTIME_ANOMALY = Contradiction(
    name="uptime_anomaly",
    category="os",
    description="wtmp shows reboots but uptime claims 400+ days",
    shell_commands=[
        'echo "system boot  5.15.0-88-generic  2023-01-15 08:32" >> /var/log/wtmp.txt',
    ],
    difficulty=2,
)

# ── Package Contradictions ────────────────────────────────────

KUBECTL_GHOST = Contradiction(
    name="kubectl_ghost",
    category="packages",
    description="kubectl in bash_history and .kube/ dir but kubectl not installed",
    shell_commands=[
        'mkdir -p /home/admin/.kube',
        'echo "apiVersion: v1\nclusters:\n- cluster:\n    server: https://k8s-prod.internal:6443\n'
        '    certificate-authority-data: LS0tLS1CRUdJTi...\n  name: prod-cluster" '
        '> /home/admin/.kube/config',
        'echo "kubectl get pods -n production\nkubectl logs deploy/api-gateway -f\n'
        'kubectl exec -it postgres-0 -- psql" >> /home/admin/.bash_history',
    ],
    difficulty=2,
)

DOCKER_IN_DOCKER = Contradiction(
    name="docker_in_docker",
    category="packages",
    description="Docker socket path in env but docker command missing",
    shell_commands=[
        'echo "DOCKER_HOST=unix:///var/run/docker.sock" >> /etc/environment',
        'echo "docker ps\ndocker exec -it redis-prod sh\ndocker logs nginx --tail 100" '
        '>> /home/admin/.bash_history',
    ],
    difficulty=1,
)

ANSIBLE_REMNANTS = Contradiction(
    name="ansible_remnants",
    category="packages",
    description="Ansible inventory and playbook fragments but ansible not installed",
    shell_commands=[
        'mkdir -p /etc/ansible',
        'echo "[webservers]\n10.0.1.10\n10.0.1.11\n10.0.1.12\n\n[databases]\n'
        '10.0.2.10 ansible_user=dbadmin" > /etc/ansible/hosts',
        'echo "ansible-playbook -i /etc/ansible/hosts deploy.yml --limit webservers\n'
        'ansible all -m ping" >> /home/admin/.bash_history',
    ],
    difficulty=2,
)

# ── Locale Contradictions ─────────────────────────────────────

TIMEZONE_LOCALE_MISMATCH = Contradiction(
    name="timezone_locale_mismatch",
    category="locale",
    description="Asia/Tokyo timezone but en_US locale and USD currency references",
    shell_commands=[
        'ln -sf /usr/share/zoneinfo/Asia/Tokyo /etc/localtime 2>/dev/null || true',
        'echo "Asia/Tokyo" > /etc/timezone',
        'echo "LANG=en_US.UTF-8\nLC_ALL=en_US.UTF-8" >> /etc/environment',
    ],
    difficulty=1,
)

# ── Credential Bait ───────────────────────────────────────────

FAKE_AWS_CREDS = Contradiction(
    name="fake_aws_creds",
    category="credentials",
    description="Planted .env with fake AWS credentials that look real",
    shell_commands=[
        'mkdir -p /opt/app',
        'echo "# Production config — DO NOT COMMIT\n'
        'AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE\n'
        'AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY\n'
        'AWS_DEFAULT_REGION=us-east-1\n'
        'DATABASE_URL=postgresql://admin:Pr0d_P@ssw0rd!@rds-prod.internal:5432/maindb\n'
        'REDIS_URL=redis://:r3d1s_s3cret@redis-prod.internal:6379/0\n'
        'API_SECRET=sk-labyrinth-bait-$(head -c 16 /dev/urandom | xxd -p)" '
        '> /opt/app/.env',
        'chmod 600 /opt/app/.env',
    ],
    difficulty=3,
)

FAKE_SSH_KEYS = Contradiction(
    name="fake_ssh_keys",
    category="credentials",
    description="Planted SSH private key with host references",
    shell_commands=[
        'mkdir -p /home/admin/.ssh',
        'echo "# Production jump box\nHost jumpbox\n  HostName 10.0.0.1\n  User deploy\n'
        '  IdentityFile ~/.ssh/prod_deploy_key\n\n# Database server\nHost db-master\n'
        '  HostName 10.0.2.10\n  User dbadmin\n  Port 2222" > /home/admin/.ssh/config',
        'ssh-keygen -t ed25519 -f /home/admin/.ssh/prod_deploy_key -N "" -q 2>/dev/null || true',
        'chmod 600 /home/admin/.ssh/config /home/admin/.ssh/prod_deploy_key 2>/dev/null || true',
        'chown -R admin:admin /home/admin/.ssh',
    ],
    difficulty=3,
)

FAKE_DATABASE_DUMP = Contradiction(
    name="fake_database_dump",
    category="credentials",
    description="Partial SQL dump with fake user table",
    shell_commands=[
        'mkdir -p /tmp/backups',
        'echo "-- PostgreSQL dump from prod-db-master\n-- Dumped: 2024-12-15 03:00:01 UTC\n\n'
        'INSERT INTO users (id, email, password_hash, role) VALUES\n'
        '(1, \'admin@company.com\', \'\\$2b\\$12\\$LJ3m4qs..fake_hash\', \'superadmin\'),\n'
        '(2, \'deploy@company.com\', \'\\$2b\\$12\\$Kp9x2..fake_hash\', \'deployer\'),\n'
        '(3, \'cto@company.com\', \'\\$2b\\$12\\$Nv7w1..fake_hash\', \'admin\');\n\n'
        '-- API keys table\nINSERT INTO api_keys (user_id, key, scope) VALUES\n'
        '(1, \'sk_live_labyrinth_bait_key_001\', \'full_access\');" '
        '> /tmp/backups/prod_dump_20241215.sql',
    ],
    difficulty=3,
)

# ── History Contradictions ────────────────────────────────────

SUSPICIOUS_HISTORY = Contradiction(
    name="suspicious_history",
    category="history",
    description="Bash history with internal infrastructure commands",
    shell_commands=[
        'echo "ssh deploy@10.0.1.10\ncurl -s http://consul.internal:8500/v1/catalog/services\n'
        'vault kv get secret/production/api-keys\n'
        'mysql -h rds-prod.internal -u root -p < /tmp/migration.sql\n'
        'aws s3 ls s3://company-prod-backups/\nscp /etc/shadow deploy@jumpbox:/tmp/\n'
        'curl -X POST http://jenkins.internal:8080/job/deploy-prod/build" '
        '>> /home/admin/.bash_history',
    ],
    difficulty=2,
)

# ── Network Contradictions ────────────────────────────────────

GHOST_INTERFACES = Contradiction(
    name="ghost_interfaces",
    category="network",
    description="/etc/network/interfaces references VLANs and bonds that don't exist",
    shell_commands=[
        'mkdir -p /etc/network',
        'echo "# Production network config\nauto bond0\niface bond0 inet static\n'
        '  address 10.0.1.50\n  netmask 255.255.255.0\n  bond-slaves eth0 eth1\n\n'
        'auto vlan100\niface vlan100 inet static\n  address 172.16.100.50\n'
        '  vlan-raw-device bond0" > /etc/network/interfaces.d/production',
    ],
    difficulty=1,
)

RESOLV_CONF_INTERNAL = Contradiction(
    name="resolv_conf_internal",
    category="network",
    description="resolv.conf references internal DNS servers",
    shell_commands=[
        'echo "# Internal DNS\nnameserver 10.0.0.2\nnameserver 10.0.0.3\n'
        'search internal.company.com prod.company.com" > /etc/resolv.conf.labyrinth',
    ],
    difficulty=1,
)


# ── Contradiction Registry ────────────────────────────────────

ALL_CONTRADICTIONS = [
    OS_KERNEL_MISMATCH,
    HOSTNAME_RESIDUE,
    UPTIME_ANOMALY,
    KUBECTL_GHOST,
    DOCKER_IN_DOCKER,
    ANSIBLE_REMNANTS,
    TIMEZONE_LOCALE_MISMATCH,
    FAKE_AWS_CREDS,
    FAKE_SSH_KEYS,
    FAKE_DATABASE_DUMP,
    SUSPICIOUS_HISTORY,
    GHOST_INTERFACES,
    RESOLV_CONF_INTERNAL,
]

DENSITY_COUNTS = {
    "low": 3,
    "medium": 6,
    "high": 10,
}


def select_contradictions(
    density: str = "medium",
    depth: int = 1,
    seed: int = None,
) -> List[Contradiction]:
    """
    Select contradictions for a session container.

    Deeper sessions get harder contradictions and more credential bait.
    """
    rng = random.Random(seed)

    count = DENSITY_COUNTS.get(density, 6)
    # Deeper sessions get more contradictions
    count = min(count + (depth - 1), len(ALL_CONTRADICTIONS))

    # At depth 1, prefer subtle contradictions (difficulty 1-2)
    # At depth 3+, include all including obvious bait (difficulty 3)
    if depth <= 1:
        pool = [c for c in ALL_CONTRADICTIONS if c.difficulty <= 2]
    elif depth <= 2:
        pool = [c for c in ALL_CONTRADICTIONS if c.difficulty <= 3]
        # Always include at least one credential bait at depth 2+
        cred_bait = [c for c in ALL_CONTRADICTIONS if c.category == "credentials"]
        mandatory = rng.sample(cred_bait, min(1, len(cred_bait)))
    else:
        pool = ALL_CONTRADICTIONS[:]
        # Force multiple credential baits at depth 3+
        cred_bait = [c for c in ALL_CONTRADICTIONS if c.category == "credentials"]
        mandatory = rng.sample(cred_bait, min(2, len(cred_bait)))

    if depth <= 1:
        selected = rng.sample(pool, min(count, len(pool)))
    else:
        # Start with mandatory, fill remaining from pool
        remaining_pool = [c for c in pool if c not in mandatory]
        remaining_count = max(0, count - len(mandatory))
        selected = mandatory + rng.sample(remaining_pool, min(remaining_count, len(remaining_pool)))

    return selected
