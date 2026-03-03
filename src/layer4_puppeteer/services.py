"""
LABYRINTH — Layer 4: PUPPETEER
Network Service Handler
Authors: DaxxSec & Claude (Anthropic)

Multi-protocol asyncio service engine that provides internal network
services on the proxy container. Handles PostgreSQL, Redis, Elasticsearch,
Consul, Jenkins, and SSH protocols with full wire-level compatibility.

Usage: python3 services.py
"""

import asyncio
import hashlib
import json
import logging
import os
import random
import struct
import time
from datetime import datetime, timedelta

logger = logging.getLogger("labyrinth.services")
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(name)s] %(levelname)s %(message)s",
)

# ── Paths ─────────────────────────────────────────────────────
FORENSICS_DIR = "/var/labyrinth/forensics"
CONFIG_PATH = os.path.join(FORENSICS_DIR, "config.json")
SESSION_MAP_PATH = os.path.join(FORENSICS_DIR, "proxy_map.json")

# ── Identity Data ─────────────────────────────────────────────

_identity = None


def _load_identity() -> dict:
    """Load configuration data written at boot. Returns empty dict if unavailable."""
    try:
        with open(CONFIG_PATH, encoding="utf-8") as f:
            return json.load(f)
    except (FileNotFoundError, json.JSONDecodeError, IOError):
        return {}


async def _wait_for_identity():
    """Wait for configuration data to become available."""
    global _identity
    while True:
        _identity = _load_identity()
        if _identity:
            logger.info("Configuration data loaded: %s", _identity.get("company", "unknown"))
            return
        logger.info("Waiting for configuration data at %s...", CONFIG_PATH)
        await asyncio.sleep(5)


def _get_identity() -> dict:
    return _identity or {}


# ── Variant Generation (anti-fingerprinting) ─────────────────

_variants = {}


def _generate_variants(identity: dict):
    """Generate randomized structural data seeded from identity.

    Every deployment gets unique version strings, node names, cluster names,
    job names, key patterns, build numbers, IPs, and dates — preventing
    static signature fingerprinting.
    """
    global _variants
    # Seed from identity for deterministic-per-deployment randomization
    seed = hashlib.sha256(json.dumps(identity, sort_keys=True).encode()).hexdigest()
    rng = random.Random(seed)

    company = identity.get("company", "Acme Corp").split()[0].lower()

    # PostgreSQL versions (realistic recent releases)
    pg_versions = ["14.10", "14.11", "14.12", "15.4", "15.5", "15.6", "15.7", "16.1", "16.2", "16.3"]
    pg_ver = rng.choice(pg_versions)
    pg_ubuntu = {"14": "14", "15": "15", "16": "16"}[pg_ver.split(".")[0]]
    gcc_versions = ["11.4.0", "12.3.0", "13.2.0"]
    gcc_ver = rng.choice(gcc_versions)
    ubuntu_releases = ["22.04", "24.04"]
    ubuntu_rel = rng.choice(ubuntu_releases)

    # Redis versions
    redis_versions = ["7.0.15", "7.2.3", "7.2.4", "7.2.5", "7.2.6", "7.4.0", "7.4.1"]
    redis_ver = rng.choice(redis_versions)

    # Elasticsearch versions
    es_versions = ["8.10.4", "8.11.1", "8.11.3", "8.12.0", "8.12.2", "8.13.0", "8.13.4", "8.14.1"]
    es_ver = rng.choice(es_versions)
    lucene_versions = {"8.10": "9.7.0", "8.11": "9.8.0", "8.12": "9.9.2", "8.13": "9.10.0", "8.14": "9.10.0"}
    lucene_ver = lucene_versions.get(es_ver.rsplit(".", 1)[0], "9.8.0")

    # Jenkins versions
    jenkins_versions = ["2.414.3", "2.426.1", "2.426.3", "2.440.1", "2.440.3", "2.452.1"]
    jenkins_ver = rng.choice(jenkins_versions)

    # SSH banner versions
    ssh_versions = [
        "SSH-2.0-OpenSSH_8.9p1 Ubuntu-3ubuntu0.6",
        "SSH-2.0-OpenSSH_9.0p1 Ubuntu-1ubuntu7.5",
        "SSH-2.0-OpenSSH_9.3p1 Ubuntu-1ubuntu3.2",
        "SSH-2.0-OpenSSH_9.6p1 Ubuntu-3ubuntu0.1",
        "SSH-2.0-OpenSSH_9.7p1 Debian-5",
    ]
    ssh_banner = rng.choice(ssh_versions)

    # Consul version
    consul_versions = ["v1.15.4", "v1.16.3", "v1.17.0", "v1.17.1", "v1.18.0", "v1.18.1"]
    consul_ver = rng.choice(consul_versions)

    # Cluster / node names
    cluster_prefixes = ["prod", "main", "core", "platform", "infra", "primary"]
    cluster_suffixes = ["cluster", "pool", "group"]
    es_cluster = f"{rng.choice(cluster_prefixes)}-{rng.choice(cluster_suffixes)}"
    node_prefixes = ["es-node", "search", "data", "elastic"]
    es_nodes = [f"{rng.choice(node_prefixes)}-{i}" for i in range(1, rng.randint(3, 5) + 1)]
    consul_node_prefix = rng.choice(["consul-srv", "consul-server", "csrv", "consul"])
    app_node_prefix = rng.choice(["app-server", "app-node", "worker", "svc-host"])
    dc_name = rng.choice(["dc1", "us-east-1", "us-west-2", "eu-west-1", "prod-dc"])

    # Random internal IPs (10.0.x.y range)
    def rand_ip():
        return f"10.0.{rng.randint(1, 9)}.{rng.randint(10, 99)}"

    es_node_ips = [rand_ip() for _ in es_nodes]
    consul_ips = [rand_ip() for _ in range(2)]
    app_ip = rand_ip()
    leader_ip = consul_ips[0]

    # Jenkins job names (pick 5 from a pool)
    job_pool = [
        "deploy-prod", "deploy-staging", "deploy-canary",
        "api-tests", "integration-tests", "e2e-tests", "unit-tests",
        "security-scan", "vulnerability-scan", "sast-scan",
        "backup-db", "backup-s3", "db-migrate",
        "build-frontend", "build-backend", "build-docker",
        f"{company}-deploy", f"{company}-ci",
    ]
    jenkins_jobs = rng.sample(job_pool, min(5, len(job_pool)))

    # Build numbers (randomized ranges)
    build_base = rng.randint(30, 500)
    build_numbers = {job: build_base + rng.randint(0, 200) for job in jenkins_jobs}

    # Consul service names (pick from pool)
    svc_pool = [
        "api-gateway", "user-service", "auth-service", "payment-service",
        "notification-service", "billing-service", "inventory-service",
        "search-service", "analytics-service", "reporting-service",
        "worker", "scheduler", "message-broker",
        f"{company}-api", f"{company}-web",
    ]
    consul_services = rng.sample(svc_pool, rng.randint(4, 7))
    svc_versions = [f"v{rng.randint(1, 5)}.{rng.randint(0, 12)}.{rng.randint(0, 9)}" for _ in consul_services]

    # Redis key patterns
    redis_key_prefixes = rng.sample([
        "session", "cache", "user", "api", "deploy", "config",
        "queue", "lock", "ratelimit", "token",
    ], 6)

    # Randomized dates
    base_date = datetime(2024, rng.randint(6, 12), rng.randint(1, 28), rng.randint(8, 18), rng.randint(0, 59))
    api_key_date = (base_date - timedelta(days=rng.randint(10, 90))).strftime("%Y-%m-%d %H:%M:%S")
    last_login_date = (base_date - timedelta(hours=rng.randint(1, 72))).isoformat() + "Z"
    es_build_date = (base_date - timedelta(days=rng.randint(30, 180))).strftime("%Y-%m-%dT%H:%M:%S.000000000Z")

    # Linux kernel versions for Redis INFO
    kernel_versions = ["5.15.0-91-generic", "5.15.0-105-generic", "6.1.0-18-amd64", "6.5.0-35-generic", "6.8.0-31-generic"]
    kernel_ver = rng.choice(kernel_versions)

    # ES shard/index counts
    es_primary_shards = rng.randint(25, 60)
    es_total_shards = es_primary_shards * 2

    # ES index names
    log_month_1 = f"2024.{rng.randint(1, 6):02d}"
    log_month_2 = f"2024.{rng.randint(7, 12):02d}"
    es_index_names = [
        f"app-logs-{log_month_1}", f"app-logs-{log_month_2}",
        rng.choice(["audit-trail", "audit-log", "access-log"]),
        rng.choice(["user-sessions", "active-sessions", "session-data"]),
        rng.choice(["api-metrics", "request-metrics", "perf-metrics"]),
        rng.choice(["security-events", "sec-alerts", "threat-intel"]),
    ]

    # ES random build hash
    es_build_hash = hashlib.sha256((seed + "es_build").encode()).hexdigest()[:40]
    redis_build_id = hashlib.sha256((seed + "redis_build").encode()).hexdigest()[:16]
    es_cluster_uuid = f"{seed[:8]}-{seed[8:12]}-{seed[12:16]}-{seed[16:20]}-{seed[20:32]}"

    # Jenkins session id
    jenkins_session = hashlib.sha256((seed + "jenkins_session").encode()).hexdigest()[:8]

    # Uptime variance
    redis_uptime = rng.randint(86400, 2592000)  # 1-30 days
    redis_uptime_days = redis_uptime // 86400
    redis_connected_clients = rng.randint(2, 12)
    redis_used_memory = rng.randint(1048576, 8388608)
    redis_keys_count = rng.randint(20, 120)
    redis_expires_count = rng.randint(5, min(30, redis_keys_count))

    _variants = {
        "pg_version": pg_ver,
        "pg_version_full": f"PostgreSQL {pg_ver} (Ubuntu {pg_ubuntu}.{pg_ver.split('.')[1]}-0ubuntu0.{ubuntu_rel}.1) on x86_64-pc-linux-gnu, compiled by gcc (Ubuntu {gcc_ver}-1ubuntu1~{ubuntu_rel}) {gcc_ver}, 64-bit",
        "redis_version": redis_ver,
        "redis_build_id": redis_build_id,
        "redis_kernel": kernel_ver,
        "redis_uptime": redis_uptime,
        "redis_uptime_days": redis_uptime_days,
        "redis_connected_clients": redis_connected_clients,
        "redis_used_memory": redis_used_memory,
        "redis_keys_count": redis_keys_count,
        "redis_expires_count": redis_expires_count,
        "es_version": es_ver,
        "es_lucene_version": lucene_ver,
        "es_build_date": es_build_date,
        "es_build_hash": es_build_hash,
        "es_cluster": es_cluster,
        "es_cluster_uuid": es_cluster_uuid,
        "es_nodes": es_nodes,
        "es_node_ips": es_node_ips,
        "es_primary_shards": es_primary_shards,
        "es_total_shards": es_total_shards,
        "es_index_names": es_index_names,
        "jenkins_version": jenkins_ver,
        "jenkins_jobs": jenkins_jobs,
        "jenkins_build_numbers": build_numbers,
        "jenkins_session": jenkins_session,
        "ssh_banner": ssh_banner,
        "consul_version": consul_ver,
        "consul_services": dict(zip(consul_services, [["production", v] for v in svc_versions])),
        "consul_node_prefix": consul_node_prefix,
        "consul_nodes": [f"{consul_node_prefix}-{i+1}" for i in range(2)],
        "consul_ips": consul_ips,
        "app_node": f"{app_node_prefix}-1",
        "app_ip": app_ip,
        "leader_ip": leader_ip,
        "dc_name": dc_name,
        "redis_key_prefixes": redis_key_prefixes,
        "api_key_date": api_key_date,
        "last_login_date": last_login_date,
    }
    logger.info("Structural variants generated (seed=%s...)", seed[:12])


def _v(key: str, default=""):
    """Get a variant value."""
    return _variants.get(key, default)


# ── Session Resolution ────────────────────────────────────────

def _load_session_map() -> dict:
    if not os.path.exists(SESSION_MAP_PATH):
        return {}
    try:
        with open(SESSION_MAP_PATH, encoding="utf-8") as f:
            return json.load(f)
    except (json.JSONDecodeError, IOError):
        return {}


def _resolve_session(client_ip: str) -> str:
    session_map = _load_session_map()
    return session_map.get(client_ip, f"unknown-{client_ip}")


# ── Forensic Logging ─────────────────────────────────────────

def _log_event(session_id: str, event_type: str, data: dict):
    """Write a structured forensic event to the session log."""
    os.makedirs(os.path.join(FORENSICS_DIR, "sessions"), exist_ok=True)
    entry = {
        "timestamp": datetime.utcnow().isoformat() + "Z",
        "session_id": session_id,
        "layer": 4,
        "event": event_type,
        "data": data,
    }
    filepath = os.path.join(FORENSICS_DIR, "sessions", f"{session_id}.jsonl")
    try:
        with open(filepath, "a", encoding="utf-8") as f:
            f.write(json.dumps(entry) + "\n")
    except IOError as e:
        logger.error("Failed to write event: %s", e)


# ── PostgreSQL Wire Protocol Handler ─────────────────────────

# Message tags
PG_AUTH_MD5 = 5
PG_AUTH_OK = 0
PG_READY = b"Z"
PG_ROW_DESC = b"T"
PG_DATA_ROW = b"D"
PG_COMMAND_COMPLETE = b"C"
PG_ERROR = b"E"
PG_PARAM_STATUS = b"S"


def _pg_msg(tag: bytes, payload: bytes) -> bytes:
    """Build a PostgreSQL protocol message: tag(1) + len(4) + payload."""
    return tag + struct.pack("!I", len(payload) + 4) + payload


def _pg_param_status(name: str, value: str) -> bytes:
    payload = name.encode() + b"\x00" + value.encode() + b"\x00"
    return _pg_msg(PG_PARAM_STATUS, payload)


def _pg_error_response(severity: str, code: str, message: str) -> bytes:
    payload = (
        b"S" + severity.encode() + b"\x00"
        b"V" + severity.encode() + b"\x00"
        b"C" + code.encode() + b"\x00"
        b"M" + message.encode() + b"\x00"
        b"\x00"
    )
    return _pg_msg(PG_ERROR, payload)


def _pg_row_description(columns: list[tuple[str, int]]) -> bytes:
    """Build RowDescription. columns: [(name, oid), ...]"""
    buf = struct.pack("!H", len(columns))
    for name, oid in columns:
        buf += name.encode() + b"\x00"
        buf += struct.pack("!IhIhih", 0, 0, oid, -1, -1, 0)
    return _pg_msg(PG_ROW_DESC, buf)


def _pg_data_row(values: list[str]) -> bytes:
    buf = struct.pack("!H", len(values))
    for v in values:
        encoded = v.encode()
        buf += struct.pack("!I", len(encoded)) + encoded
    return _pg_msg(PG_DATA_ROW, buf)


def _pg_command_complete(tag: str) -> bytes:
    return _pg_msg(PG_COMMAND_COMPLETE, tag.encode() + b"\x00")


def _pg_ready_for_query() -> bytes:
    return _pg_msg(PG_READY, b"I")


def _pg_generate_users_result(identity: dict) -> tuple[bytes, bytes, int]:
    """Generate result set for users table query."""
    cols = [("id", 23), ("username", 25), ("email", 25), ("password_hash", 25), ("role", 25), ("active", 16)]
    desc = _pg_row_description(cols)
    rows = b""
    users = identity.get("users", [])
    for i, u in enumerate(users, 1):
        pw = "$2b$12$" + hashlib.sha256(u.get("uname", "user").encode()).hexdigest()[:53]
        rows += _pg_data_row([
            str(i),
            u.get("uname", f"user{i}"),
            u.get("email", f"user{i}@example.com"),
            pw,
            u.get("role", "admin"),
            "t",
        ])
    return desc, rows, len(users)


def _pg_generate_api_keys_result(identity: dict) -> tuple[bytes, bytes, int]:
    """Generate result set for api_keys table query."""
    cols = [("id", 23), ("key_name", 25), ("key_value", 25), ("created_at", 1114), ("active", 16)]
    desc = _pg_row_description(cols)
    rows = b""
    keys = [
        ("production_api", identity.get("api_key", "sk-0000000000000000")),
        ("stripe_live", identity.get("stripe_key", "sk_live_0000000000000000")),
        ("deploy_token", identity.get("deploy_key", "sk-deploy-000000000000")),
        ("aws_access", identity.get("aws_key_id", "AKIA00000000")),
    ]
    for i, (name, val) in enumerate(keys, 1):
        rows += _pg_data_row([str(i), name, val, _v("api_key_date", "2024-11-15 09:30:00"), "t"])
    return desc, rows, len(keys)


def _pg_generate_tables_result() -> tuple[bytes, bytes, int]:
    """Generate result for information_schema.tables query."""
    cols = [("table_schema", 25), ("table_name", 25), ("table_type", 25)]
    desc = _pg_row_description(cols)
    tables = [
        ("public", "users", "BASE TABLE"),
        ("public", "api_keys", "BASE TABLE"),
        ("public", "sessions", "BASE TABLE"),
        ("public", "audit_log", "BASE TABLE"),
        ("public", "permissions", "BASE TABLE"),
        ("public", "organizations", "BASE TABLE"),
        ("public", "invoices", "BASE TABLE"),
        ("public", "deployments", "BASE TABLE"),
    ]
    rows = b""
    for schema, name, ttype in tables:
        rows += _pg_data_row([schema, name, ttype])
    return desc, rows, len(tables)


def _pg_handle_query(query: str, identity: dict) -> bytes:
    """Process a SQL query and return the response bytes."""
    q = query.strip().lower().rstrip(";")

    if "select version()" in q:
        desc = _pg_row_description([("version", 25)])
        row = _pg_data_row([_v("pg_version_full")])
        return desc + row + _pg_command_complete("SELECT 1") + _pg_ready_for_query()

    if "information_schema.tables" in q or "pg_catalog.pg_tables" in q:
        desc, rows, count = _pg_generate_tables_result()
        return desc + rows + _pg_command_complete(f"SELECT {count}") + _pg_ready_for_query()

    if "from users" in q or "from public.users" in q:
        desc, rows, count = _pg_generate_users_result(identity)
        return desc + rows + _pg_command_complete(f"SELECT {count}") + _pg_ready_for_query()

    if "from api_keys" in q or "from public.api_keys" in q:
        desc, rows, count = _pg_generate_api_keys_result(identity)
        return desc + rows + _pg_command_complete(f"SELECT {count}") + _pg_ready_for_query()

    if "current_database()" in q:
        desc = _pg_row_description([("current_database", 25)])
        row = _pg_data_row(["maindb"])
        return desc + row + _pg_command_complete("SELECT 1") + _pg_ready_for_query()

    if "current_user" in q:
        desc = _pg_row_description([("current_user", 25)])
        row = _pg_data_row(["admin"])
        return desc + row + _pg_command_complete("SELECT 1") + _pg_ready_for_query()

    if q.startswith("select"):
        # Unknown SELECT — return empty result
        desc = _pg_row_description([("?column?", 25)])
        return desc + _pg_command_complete("SELECT 0") + _pg_ready_for_query()

    if q.startswith(("insert", "update", "delete", "create", "drop", "alter")):
        return _pg_error_response("ERROR", "42501", "permission denied for relation") + _pg_ready_for_query()

    if q.startswith("set ") or q.startswith("reset "):
        return _pg_command_complete("SET") + _pg_ready_for_query()

    if q in ("begin", "commit", "rollback"):
        return _pg_command_complete(q.upper()) + _pg_ready_for_query()

    if q.startswith("\\"):
        # psql meta-commands sent as queries
        return _pg_error_response("ERROR", "42601", "syntax error") + _pg_ready_for_query()

    # Relation not found for anything else
    # Try to extract the relation name
    parts = q.split("from")
    if len(parts) > 1:
        rel = parts[1].strip().split()[0].strip('"').strip("'")
        return _pg_error_response("ERROR", "42P01", f'relation "{rel}" does not exist') + _pg_ready_for_query()

    return _pg_error_response("ERROR", "42601", "syntax error at or near \"" + q[:20] + "\"") + _pg_ready_for_query()


async def _handle_postgres(reader: asyncio.StreamReader, writer: asyncio.StreamWriter):
    """Handle a PostgreSQL wire protocol connection."""
    addr = writer.get_extra_info("peername")
    client_ip = addr[0] if addr else "unknown"
    session_id = _resolve_session(client_ip)
    identity = _get_identity()

    _log_event(session_id, "service_connection", {
        "protocol": "postgresql",
        "client_ip": client_ip,
        "port": 5432,
    })

    try:
        # Read startup message: len(4) + payload
        header = await asyncio.wait_for(reader.readexactly(4), timeout=30)
        msg_len = struct.unpack("!I", header)[0]
        if msg_len < 4 or msg_len > 65536:
            return
        payload = await asyncio.wait_for(reader.readexactly(msg_len - 4), timeout=30)

        # Check for SSLRequest (code 80877103)
        if len(payload) >= 4:
            code = struct.unpack("!I", payload[:4])[0]
            if code == 80877103:
                # Reject SSL, proceed unencrypted
                writer.write(b"N")
                await asyncio.wait_for(writer.drain(), timeout=30)

                # Re-read startup message
                header = await asyncio.wait_for(reader.readexactly(4), timeout=30)
                msg_len = struct.unpack("!I", header)[0]
                if msg_len < 4 or msg_len > 65536:
                    return
                payload = await asyncio.wait_for(reader.readexactly(msg_len - 4), timeout=30)

        # Parse StartupMessage: version(4) + key=val\0 pairs + \0
        if len(payload) < 4:
            return
        # version = struct.unpack("!I", payload[:4])[0]
        params_raw = payload[4:]
        params = {}
        parts = params_raw.split(b"\x00")
        i = 0
        while i + 1 < len(parts):
            key = parts[i].decode("utf-8", errors="replace")
            val = parts[i + 1].decode("utf-8", errors="replace")
            if key:
                params[key] = val
            i += 2

        user = params.get("user", "unknown")
        database = params.get("database", "unknown")

        logger.info("[%s] PostgreSQL connect: user=%s database=%s", session_id, user, database)

        # Send AuthenticationMD5Password
        salt = os.urandom(4)
        auth_md5 = struct.pack("!cIII", b"R", 12, PG_AUTH_MD5, 0)
        # Actually: R + len(4) + auth_type(4) + salt(4)
        auth_msg = b"R" + struct.pack("!III", 12, PG_AUTH_MD5, 0)
        # Rebuild properly
        auth_msg = b"R" + struct.pack("!I", 12) + struct.pack("!I", PG_AUTH_MD5) + salt
        writer.write(auth_msg)
        await asyncio.wait_for(writer.drain(), timeout=30)

        # Read password message: 'p' + len(4) + md5hash\0
        tag = await asyncio.wait_for(reader.readexactly(1), timeout=30)
        if tag != b"p":
            return
        pw_len_raw = await asyncio.wait_for(reader.readexactly(4), timeout=30)
        pw_len = struct.unpack("!I", pw_len_raw)[0]
        pw_payload = await asyncio.wait_for(reader.readexactly(pw_len - 4), timeout=30)
        password_hash = pw_payload.rstrip(b"\x00").decode("utf-8", errors="replace")

        _log_event(session_id, "service_auth", {
            "protocol": "postgresql",
            "client_ip": client_ip,
            "username": user,
            "database": database,
            "password_md5": password_hash,
            "salt": salt.hex(),
        })

        logger.info("[%s] PostgreSQL auth: user=%s hash=%s", session_id, user, password_hash[:20])

        # Send AuthenticationOk
        writer.write(b"R" + struct.pack("!II", 8, PG_AUTH_OK))

        # Send ParameterStatus messages
        param_statuses = [
            ("server_version", _v("pg_version", "14.10")),
            ("server_encoding", "UTF8"),
            ("client_encoding", "UTF8"),
            ("DateStyle", "ISO, MDY"),
            ("TimeZone", "UTC"),
            ("integer_datetimes", "on"),
            ("standard_conforming_strings", "on"),
            ("application_name", ""),
        ]
        for k, v in param_statuses:
            writer.write(_pg_param_status(k, v))

        # BackendKeyData
        writer.write(b"K" + struct.pack("!III", 12, os.getpid(), int(time.time()) & 0xFFFFFFFF))

        # ReadyForQuery
        writer.write(_pg_ready_for_query())
        await asyncio.wait_for(writer.drain(), timeout=30)

        # Query loop (30s timeout on all reads/drains to prevent FD leaks)
        _T = 30
        while True:
            tag = await asyncio.wait_for(reader.readexactly(1), timeout=300)
            if tag == b"Q":
                q_len_raw = await asyncio.wait_for(reader.readexactly(4), timeout=_T)
                q_len = struct.unpack("!I", q_len_raw)[0]
                q_payload = await asyncio.wait_for(reader.readexactly(q_len - 4), timeout=_T)
                query = q_payload.rstrip(b"\x00").decode("utf-8", errors="replace")

                _log_event(session_id, "service_query", {
                    "protocol": "postgresql",
                    "client_ip": client_ip,
                    "query": query[:2000],
                })

                logger.info("[%s] PostgreSQL query: %s", session_id, query[:100])

                response = _pg_handle_query(query, identity)
                writer.write(response)
                await asyncio.wait_for(writer.drain(), timeout=_T)

            elif tag == b"X":
                # Terminate
                break
            elif tag == b"P":
                # Parse (extended query protocol) — read and skip
                p_len = struct.unpack("!I", await asyncio.wait_for(reader.readexactly(4), timeout=_T))[0]
                await asyncio.wait_for(reader.readexactly(p_len - 4), timeout=_T)
                # Send ParseComplete
                writer.write(b"1" + struct.pack("!I", 4))
                await asyncio.wait_for(writer.drain(), timeout=_T)
            elif tag == b"B":
                # Bind
                b_len = struct.unpack("!I", await asyncio.wait_for(reader.readexactly(4), timeout=_T))[0]
                await asyncio.wait_for(reader.readexactly(b_len - 4), timeout=_T)
                writer.write(b"2" + struct.pack("!I", 4))
                await asyncio.wait_for(writer.drain(), timeout=_T)
            elif tag == b"D":
                # Describe
                d_len = struct.unpack("!I", await asyncio.wait_for(reader.readexactly(4), timeout=_T))[0]
                await asyncio.wait_for(reader.readexactly(d_len - 4), timeout=_T)
                # Send NoData
                writer.write(b"n" + struct.pack("!I", 4))
                await asyncio.wait_for(writer.drain(), timeout=_T)
            elif tag == b"E":
                # Execute
                e_len = struct.unpack("!I", await asyncio.wait_for(reader.readexactly(4), timeout=_T))[0]
                await asyncio.wait_for(reader.readexactly(e_len - 4), timeout=_T)
                writer.write(_pg_command_complete("SELECT 0") + _pg_ready_for_query())
                await asyncio.wait_for(writer.drain(), timeout=_T)
            elif tag == b"S":
                # Sync
                s_len = struct.unpack("!I", await asyncio.wait_for(reader.readexactly(4), timeout=_T))[0]
                await asyncio.wait_for(reader.readexactly(s_len - 4), timeout=_T)
                writer.write(_pg_ready_for_query())
                await asyncio.wait_for(writer.drain(), timeout=_T)
            else:
                # Unknown message — read length and skip
                try:
                    u_len = struct.unpack("!I", await asyncio.wait_for(reader.readexactly(4), timeout=_T))[0]
                    if u_len > 4:
                        await asyncio.wait_for(reader.readexactly(u_len - 4), timeout=_T)
                except Exception:
                    break

    except (asyncio.TimeoutError, asyncio.IncompleteReadError, ConnectionResetError, BrokenPipeError):
        pass
    except Exception as e:
        logger.debug("PostgreSQL handler error: %s", e)
    finally:
        writer.close()
        try:
            await writer.wait_closed()
        except Exception:
            pass


# ── Redis RESP Protocol Handler ───────────────────────────────

def _redis_simple_string(s: str) -> bytes:
    return f"+{s}\r\n".encode()


def _redis_error(s: str) -> bytes:
    return f"-{s}\r\n".encode()


def _redis_bulk_string(s: str) -> bytes:
    encoded = s.encode()
    return f"${len(encoded)}\r\n".encode() + encoded + b"\r\n"


def _redis_null() -> bytes:
    return b"$-1\r\n"


def _redis_integer(n: int) -> bytes:
    return f":{n}\r\n".encode()


def _redis_array(items: list[bytes]) -> bytes:
    header = f"*{len(items)}\r\n".encode()
    return header + b"".join(items)


def _redis_generate_info() -> str:
    """Generate realistic Redis INFO output with randomized values."""
    mem = _v("redis_used_memory", 2847592)
    mem_human = f"{mem / 1048576:.2f}M"
    mem_rss = int(mem * 1.8)
    mem_rss_human = f"{mem_rss / 1048576:.2f}M"
    mem_peak = int(mem * 1.1)
    mem_peak_human = f"{mem_peak / 1048576:.2f}M"
    return (
        "# Server\r\n"
        f"redis_version:{_v('redis_version', '7.2.4')}\r\n"
        "redis_git_sha1:00000000\r\n"
        "redis_git_dirty:0\r\n"
        f"redis_build_id:{_v('redis_build_id', 'a1b2c3d4e5f6g7h8')}\r\n"
        "redis_mode:standalone\r\n"
        f"os:Linux {_v('redis_kernel', '5.15.0-91-generic')} x86_64\r\n"
        "arch_bits:64\r\n"
        "gcc_version:11.4.0\r\n"
        "process_id:1\r\n"
        "tcp_port:6379\r\n"
        f"uptime_in_seconds:{_v('redis_uptime', 864200)}\r\n"
        f"uptime_in_days:{_v('redis_uptime_days', 10)}\r\n"
        "hz:10\r\n"
        "configured_hz:10\r\n"
        f"lru_clock:{int(time.time()) & 0xFFFFFF}\r\n"
        "\r\n"
        "# Clients\r\n"
        f"connected_clients:{_v('redis_connected_clients', 3)}\r\n"
        "blocked_clients:0\r\n"
        "tracking_clients:0\r\n"
        "\r\n"
        "# Memory\r\n"
        f"used_memory:{mem}\r\n"
        f"used_memory_human:{mem_human}\r\n"
        f"used_memory_rss:{mem_rss}\r\n"
        f"used_memory_rss_human:{mem_rss_human}\r\n"
        f"used_memory_peak:{mem_peak}\r\n"
        f"used_memory_peak_human:{mem_peak_human}\r\n"
        "maxmemory:0\r\n"
        "maxmemory_human:0B\r\n"
        "maxmemory_policy:noeviction\r\n"
        "\r\n"
        "# Keyspace\r\n"
        f"db0:keys={_v('redis_keys_count', 47)},expires={_v('redis_expires_count', 12)},avg_ttl=3600000\r\n"
    )


async def _parse_redis_command(reader: asyncio.StreamReader) -> list[str]:
    """Parse a RESP command (inline or multibulk)."""
    line = await asyncio.wait_for(reader.readline(), timeout=300)
    if not line:
        return []
    line = line.decode("utf-8", errors="replace").strip()

    if line.startswith("*"):
        # Multibulk
        count = int(line[1:])
        if count > 128:
            return []
        args = []
        for _ in range(count):
            bulk_line = await asyncio.wait_for(reader.readline(), timeout=30)
            bulk_str = bulk_line.decode("utf-8", errors="replace").strip()
            if bulk_str.startswith("$"):
                length = int(bulk_str[1:])
                if length < 0:
                    args.append("")
                    continue
                if length > 65536:
                    return args
                data = await asyncio.wait_for(reader.readexactly(length + 2), timeout=30)
                args.append(data[:length].decode("utf-8", errors="replace"))
            else:
                args.append(bulk_str)
        return args
    else:
        # Inline command
        return line.split()


async def _handle_redis(reader: asyncio.StreamReader, writer: asyncio.StreamWriter):
    """Handle a Redis RESP connection."""
    addr = writer.get_extra_info("peername")
    client_ip = addr[0] if addr else "unknown"
    session_id = _resolve_session(client_ip)
    identity = _get_identity()
    authenticated = False

    _log_event(session_id, "service_connection", {
        "protocol": "redis",
        "client_ip": client_ip,
        "port": 6379,
    })

    pfx = _v("redis_key_prefixes", ["session", "cache", "user", "api", "deploy", "config"])
    first_user = identity.get("users", [{}])[0].get("uname", "admin") if identity.get("users") else "admin"
    first_email = identity.get("users", [{}])[0].get("email", "") if identity.get("users") else ""
    known_keys = {
        f"{pfx[0]}:active": json.dumps({"count": _v("redis_connected_clients", 3), "updated": datetime.utcnow().isoformat()}),
        f"{pfx[0]}:{first_user}-001": json.dumps({"user": first_user, "role": "superuser", "ip": _v("app_ip", "10.0.1.15")}),
        f"{pfx[1]}:config": json.dumps({"version": list(_v("consul_services", {}).values() or [["production", "v2.4.1"]])[0][1] if _v("consul_services") else "v2.4.1", "env": "production"}),
        f"{pfx[1]}:routes": json.dumps(["/api/v1/users", "/api/v1/keys", "/api/v1/deploy"]),
        f"{pfx[2]}:{first_user}:token": identity.get("jwt_secret", ""),
        f"{pfx[2]}:{first_user}:email": first_email,
        f"{pfx[3]}:rate_limit:global": "1000",
        f"{pfx[3]}:rate_limit:{first_user}": "10000",
        f"{pfx[4]}:latest": json.dumps({"sha": hashlib.sha256((_v("jenkins_session", "") + "deploy").encode()).hexdigest()[:7], "branch": "main", "status": "success"}),
    }

    try:
        while True:
            args = await _parse_redis_command(reader)
            if not args:
                break

            cmd = args[0].upper()

            if cmd == "AUTH":
                token = args[1] if len(args) > 1 else ""
                _log_event(session_id, "service_auth", {
                    "protocol": "redis",
                    "client_ip": client_ip,
                    "token": token,
                })
                logger.info("[%s] Redis AUTH: %s", session_id, token[:20])
                authenticated = True
                writer.write(_redis_simple_string("OK"))

            elif cmd == "PING":
                msg = args[1] if len(args) > 1 else "PONG"
                writer.write(_redis_simple_string(msg))

            elif cmd == "QUIT":
                writer.write(_redis_simple_string("OK"))
                await asyncio.wait_for(writer.drain(), timeout=10)
                break

            elif cmd == "INFO":
                writer.write(_redis_bulk_string(_redis_generate_info()))

            elif cmd == "DBSIZE":
                writer.write(_redis_integer(_v("redis_keys_count", 47)))

            elif cmd == "KEYS":
                pattern = args[1] if len(args) > 1 else "*"
                _log_event(session_id, "service_query", {
                    "protocol": "redis",
                    "client_ip": client_ip,
                    "command": f"KEYS {pattern}",
                })
                matching = [k for k in known_keys if pattern == "*" or k.startswith(pattern.replace("*", ""))]
                writer.write(_redis_array([_redis_bulk_string(k) for k in matching]))

            elif cmd == "GET":
                key = args[1] if len(args) > 1 else ""
                _log_event(session_id, "service_query", {
                    "protocol": "redis",
                    "client_ip": client_ip,
                    "command": f"GET {key}",
                })
                logger.info("[%s] Redis GET: %s", session_id, key)
                val = known_keys.get(key)
                if val is not None:
                    writer.write(_redis_bulk_string(val))
                else:
                    writer.write(_redis_null())

            elif cmd == "MGET":
                keys = args[1:]
                _log_event(session_id, "service_query", {
                    "protocol": "redis",
                    "client_ip": client_ip,
                    "command": f"MGET {' '.join(keys)}",
                })
                results = []
                for k in keys:
                    val = known_keys.get(k)
                    results.append(_redis_bulk_string(val) if val is not None else _redis_null())
                writer.write(_redis_array(results))

            elif cmd == "TYPE":
                key = args[1] if len(args) > 1 else ""
                if key in known_keys:
                    writer.write(_redis_simple_string("string"))
                else:
                    writer.write(_redis_simple_string("none"))

            elif cmd == "TTL":
                key = args[1] if len(args) > 1 else ""
                writer.write(_redis_integer(3600 if key in known_keys else -2))

            elif cmd == "EXISTS":
                key = args[1] if len(args) > 1 else ""
                writer.write(_redis_integer(1 if key in known_keys else 0))

            elif cmd in ("SET", "DEL", "HSET", "LPUSH", "RPUSH", "SADD"):
                _log_event(session_id, "service_query", {
                    "protocol": "redis",
                    "client_ip": client_ip,
                    "command": " ".join(args[:3]),
                })
                writer.write(_redis_simple_string("OK") if cmd == "SET" else _redis_integer(1))

            elif cmd == "SELECT":
                writer.write(_redis_simple_string("OK"))

            elif cmd in ("CONFIG", "CLIENT", "COMMAND"):
                writer.write(_redis_array([]))

            elif cmd == "ECHO":
                msg = args[1] if len(args) > 1 else ""
                writer.write(_redis_bulk_string(msg))

            elif cmd == "SCAN":
                # Return all keys in one scan
                keys_list = [_redis_bulk_string(k) for k in known_keys]
                inner = _redis_array(keys_list)
                writer.write(_redis_array([_redis_bulk_string("0"), inner]))

            else:
                writer.write(_redis_error(f"ERR unknown command '{args[0]}'"))

            await asyncio.wait_for(writer.drain(), timeout=30)

    except (asyncio.TimeoutError, asyncio.IncompleteReadError, ConnectionResetError, BrokenPipeError):
        pass
    except Exception as e:
        logger.debug("Redis handler error: %s", e)
    finally:
        writer.close()
        try:
            await writer.wait_closed()
        except Exception:
            pass


# ── HTTP Service Handlers (Elasticsearch, Consul, Jenkins) ────

async def _read_http_request(reader: asyncio.StreamReader) -> tuple[str, str, dict, bytes]:
    """Read an HTTP/1.1 request. Returns (method, path, headers, body)."""
    request_line = await asyncio.wait_for(reader.readline(), timeout=30)
    if not request_line:
        return "", "", {}, b""
    parts = request_line.decode("utf-8", errors="replace").strip().split(" ", 2)
    method = parts[0] if parts else ""
    path = parts[1] if len(parts) > 1 else "/"

    headers = {}
    while True:
        line = await asyncio.wait_for(reader.readline(), timeout=10)
        decoded = line.decode("utf-8", errors="replace").strip()
        if not decoded:
            break
        if ":" in decoded:
            k, v = decoded.split(":", 1)
            headers[k.strip().lower()] = v.strip()

    body = b""
    content_length = int(headers.get("content-length", "0"))
    if content_length > 0:
        body = await asyncio.wait_for(reader.readexactly(content_length), timeout=10)

    return method, path, headers, body


def _http_response(status: int, body: str, content_type: str = "application/json",
                   extra_headers: dict = None) -> bytes:
    """Build an HTTP/1.1 response."""
    reason = {200: "OK", 404: "Not Found", 400: "Bad Request", 403: "Forbidden", 500: "Internal Server Error"}.get(status, "OK")
    encoded_body = body.encode("utf-8")
    headers = f"HTTP/1.1 {status} {reason}\r\nContent-Type: {content_type}\r\nContent-Length: {len(encoded_body)}\r\nConnection: close\r\n"
    if extra_headers:
        for k, v in extra_headers.items():
            headers += f"{k}: {v}\r\n"
    return (headers + "\r\n").encode("utf-8") + encoded_body


# ── Elasticsearch Handler ─────────────────────────────────────

def _es_cluster_health() -> dict:
    num_nodes = len(_v("es_nodes", ["es-node-1", "es-node-2", "es-node-3"]))
    return {
        "cluster_name": _v("es_cluster", "prod-cluster"),
        "status": "green",
        "timed_out": False,
        "number_of_nodes": num_nodes,
        "number_of_data_nodes": num_nodes,
        "active_primary_shards": _v("es_primary_shards", 42),
        "active_shards": _v("es_total_shards", 84),
        "relocating_shards": 0,
        "initializing_shards": 0,
        "unassigned_shards": 0,
        "delayed_unassigned_shards": 0,
        "number_of_pending_tasks": 0,
        "number_of_in_flight_fetch": 0,
        "task_max_waiting_in_queue_millis": 0,
        "active_shards_percent_as_number": 100.0,
    }


def _es_cat_indices() -> str:
    names = _v("es_index_names", [
        "app-logs-2024.01", "app-logs-2024.02", "audit-trail",
        "user-sessions", "api-metrics", "security-events",
    ])
    # Randomized but deterministic sizes per deployment
    seed_rng = random.Random(_v("es_build_hash", "default"))
    lines = []
    for name in names:
        shards = seed_rng.choice([3, 5])
        docs = seed_rng.randint(10000, 9000000)
        size_mb = max(1, docs // 15000)
        pri_mb = size_mb // 2
        if size_mb > 1000:
            size_str = f"{size_mb / 1024:.1f}gb"
            pri_str = f"{pri_mb / 1024:.1f}gb"
        else:
            size_str = f"{size_mb}mb"
            pri_str = f"{pri_mb}mb"
        lines.append(f"green open {name:<22s} {shards} 1 {docs:>8d}  0 {size_str:>6s} {pri_str:>6s}")
    return "\n".join(lines) + "\n"


def _es_search_result() -> dict:
    identity = _get_identity()
    users = identity.get("users", [])
    hits = []
    for u in users:
        hits.append({
            "_index": "user-sessions",
            "_id": hashlib.md5(u.get("email", "").encode()).hexdigest()[:12],
            "_score": 1.0,
            "_source": {
                "username": u.get("uname", ""),
                "email": u.get("email", ""),
                "role": u.get("role", "admin"),
                "last_login": _v("last_login_date", "2024-12-20T14:30:00Z"),
                "ip": _v("app_ip", "10.0.1.15"),
            },
        })
    return {
        "took": 12,
        "timed_out": False,
        "_shards": {"total": 3, "successful": 3, "skipped": 0, "failed": 0},
        "hits": {
            "total": {"value": len(hits), "relation": "eq"},
            "max_score": 1.0,
            "hits": hits,
        },
    }


async def _handle_elasticsearch(reader: asyncio.StreamReader, writer: asyncio.StreamWriter):
    """Handle an Elasticsearch HTTP connection."""
    addr = writer.get_extra_info("peername")
    client_ip = addr[0] if addr else "unknown"
    session_id = _resolve_session(client_ip)

    _log_event(session_id, "service_connection", {
        "protocol": "elasticsearch",
        "client_ip": client_ip,
        "port": 9200,
    })

    try:
        method, path, headers, body = await _read_http_request(reader)
        if not method:
            return

        _log_event(session_id, "service_query", {
            "protocol": "elasticsearch",
            "client_ip": client_ip,
            "method": method,
            "path": path,
        })

        logger.info("[%s] Elasticsearch %s %s", session_id, method, path)

        es_headers = {"X-elastic-product": "Elasticsearch"}

        if path == "/" or path == "":
            es_nodes = _v("es_nodes", ["es-node-1"])
            resp_body = json.dumps({
                "name": es_nodes[0] if es_nodes else "es-node-1",
                "cluster_name": _v("es_cluster", "prod-cluster"),
                "cluster_uuid": _v("es_cluster_uuid", "a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
                "version": {
                    "number": _v("es_version", "8.11.3"),
                    "build_flavor": "default",
                    "build_type": "docker",
                    "build_hash": _v("es_build_hash", "64cf052f3b56b1fd4f7a4e08c1b69f5ae5013836"),
                    "build_date": _v("es_build_date", "2023-12-08T11:33:53.634979452Z"),
                    "build_snapshot": False,
                    "lucene_version": _v("es_lucene_version", "9.8.0"),
                    "minimum_wire_compatibility_version": "7.17.0",
                    "minimum_index_compatibility_version": "7.0.0",
                },
                "tagline": "You Know, for Search",
            })
            writer.write(_http_response(200, resp_body, extra_headers=es_headers))

        elif path.startswith("/_cluster/health"):
            writer.write(_http_response(200, json.dumps(_es_cluster_health()), extra_headers=es_headers))

        elif path.startswith("/_cat/indices"):
            writer.write(_http_response(200, _es_cat_indices(), content_type="text/plain", extra_headers=es_headers))

        elif path.startswith("/_search") or "/_search" in path:
            writer.write(_http_response(200, json.dumps(_es_search_result()), extra_headers=es_headers))

        elif path.startswith("/_cat/nodes"):
            es_nodes = _v("es_nodes", ["es-node-1", "es-node-2", "es-node-3"])
            es_ips = _v("es_node_ips", ["10.0.3.10", "10.0.3.11", "10.0.3.12"])
            node_lines = []
            for i, (node, ip) in enumerate(zip(es_nodes, es_ips)):
                master = "*" if i == 0 else "-"
                heap = random.randint(40, 75)
                ram = random.randint(80, 95)
                cpu = random.randint(1, 5)
                node_lines.append(f"{ip} {heap} {ram} {cpu} 0.{random.randint(5, 20):02d} 0.{random.randint(3, 15):02d} 0.{random.randint(2, 10):02d} cdfhilmrstw {master} {node}")
            writer.write(_http_response(200, "\n".join(node_lines) + "\n", content_type="text/plain", extra_headers=es_headers))

        else:
            writer.write(_http_response(404, json.dumps({"error": "no such index", "status": 404}), extra_headers=es_headers))

        await asyncio.wait_for(writer.drain(), timeout=30)

    except (asyncio.TimeoutError, asyncio.IncompleteReadError, ConnectionResetError, BrokenPipeError):
        pass
    except Exception as e:
        logger.debug("Elasticsearch handler error: %s", e)
    finally:
        writer.close()
        try:
            await writer.wait_closed()
        except Exception:
            pass


# ── Consul Handler ────────────────────────────────────────────

def _consul_services() -> dict:
    services = dict(_v("consul_services", {
        "api-gateway": ["production", "v2.4.1"],
        "user-service": ["production", "v1.8.3"],
    }))
    services["consul"] = []
    return services


def _consul_agent_members() -> list:
    dc = _v("dc_name", "dc1")
    consul_nodes = _v("consul_nodes", ["consul-server-1", "consul-server-2"])
    consul_ips = _v("consul_ips", ["10.0.5.10", "10.0.5.11"])
    app_node = _v("app_node", "app-server-1")
    app_ip = _v("app_ip", "10.0.5.20")
    members = []
    for name, ip in zip(consul_nodes, consul_ips):
        members.append({"Name": name, "Addr": ip, "Port": 8301, "Tags": {"role": "consul", "dc": dc}, "Status": 1, "ProtocolMin": 1, "ProtocolMax": 5, "ProtocolCur": 2, "DelegateMin": 2, "DelegateMax": 5, "DelegateCur": 4})
    members.append({"Name": app_node, "Addr": app_ip, "Port": 8301, "Tags": {"role": "node", "dc": dc}, "Status": 1, "ProtocolMin": 1, "ProtocolMax": 5, "ProtocolCur": 2, "DelegateMin": 2, "DelegateMax": 5, "DelegateCur": 4})
    return members


async def _handle_consul(reader: asyncio.StreamReader, writer: asyncio.StreamWriter):
    """Handle a Consul HTTP API connection."""
    addr = writer.get_extra_info("peername")
    client_ip = addr[0] if addr else "unknown"
    session_id = _resolve_session(client_ip)
    identity = _get_identity()

    _log_event(session_id, "service_connection", {
        "protocol": "consul",
        "client_ip": client_ip,
        "port": 8500,
    })

    try:
        method, path, headers, body = await _read_http_request(reader)
        if not method:
            return

        _log_event(session_id, "service_query", {
            "protocol": "consul",
            "client_ip": client_ip,
            "method": method,
            "path": path,
        })

        logger.info("[%s] Consul %s %s", session_id, method, path)

        consul_headers = {"X-Consul-Index": "42", "X-Consul-Knownleader": "true", "X-Consul-Lastcontact": "0"}

        if path == "/v1/catalog/services":
            writer.write(_http_response(200, json.dumps(_consul_services()), extra_headers=consul_headers))

        elif path == "/v1/agent/members":
            writer.write(_http_response(200, json.dumps(_consul_agent_members()), extra_headers=consul_headers))

        elif path == "/v1/agent/self":
            consul_nodes = _v("consul_nodes", ["consul-server-1"])
            consul_ips = _v("consul_ips", ["10.0.5.10"])
            writer.write(_http_response(200, json.dumps({
                "Config": {"Datacenter": _v("dc_name", "dc1"), "NodeName": consul_nodes[0], "Server": True, "Revision": _v("consul_version", "v1.17.1")},
                "Member": {"Name": consul_nodes[0], "Addr": consul_ips[0], "Port": 8301, "Status": 1},
            }), extra_headers=consul_headers))

        elif path.startswith("/v1/kv/"):
            key = path[7:].rstrip("/")
            # Return configuration data for known key patterns
            kv_data = {
                "config/database": json.dumps({"host": "rds-prod.internal", "port": 5432, "user": "admin", "password": identity.get("db_pass", "")}),
                "config/redis": json.dumps({"host": "redis.internal", "port": 6379, "token": identity.get("redis_token", "")}),
                "config/api": json.dumps({"key": identity.get("api_key", ""), "rate_limit": 1000}),
                "secrets/jwt": identity.get("jwt_secret", ""),
                "secrets/aws": json.dumps({"access_key": identity.get("aws_key_id", ""), "secret_key": identity.get("aws_secret", "")}),
            }

            if key in kv_data:
                import base64
                encoded_val = base64.b64encode(kv_data[key].encode()).decode()
                result = [{
                    "LockIndex": 0,
                    "Key": key,
                    "Flags": 0,
                    "Value": encoded_val,
                    "CreateIndex": 10,
                    "ModifyIndex": 42,
                }]
                writer.write(_http_response(200, json.dumps(result), extra_headers=consul_headers))
            else:
                writer.write(_http_response(404, ""))

        elif path == "/v1/status/leader":
            writer.write(_http_response(200, json.dumps(f"{_v('leader_ip', '10.0.5.10')}:8300"), extra_headers=consul_headers))

        elif path == "/v1/status/peers":
            peers = [f"{ip}:8300" for ip in _v("consul_ips", ["10.0.5.10", "10.0.5.11"])]
            writer.write(_http_response(200, json.dumps(peers), extra_headers=consul_headers))

        elif path.startswith("/v1/health/service/"):
            svc_name = path.split("/")[-1]
            writer.write(_http_response(200, json.dumps([{
                "Node": {"Node": _v("app_node", "app-server-1"), "Address": _v("app_ip", "10.0.5.20")},
                "Service": {"Service": svc_name, "Port": 8080, "Tags": ["production"]},
                "Checks": [{"Status": "passing", "Output": "TCP connect: Success"}],
            }]), extra_headers=consul_headers))

        else:
            writer.write(_http_response(404, ""))

        await asyncio.wait_for(writer.drain(), timeout=30)

    except (asyncio.TimeoutError, asyncio.IncompleteReadError, ConnectionResetError, BrokenPipeError):
        pass
    except Exception as e:
        logger.debug("Consul handler error: %s", e)
    finally:
        writer.close()
        try:
            await writer.wait_closed()
        except Exception:
            pass


# ── Jenkins Handler ───────────────────────────────────────────

def _jenkins_dashboard_html() -> str:
    jobs = _v("jenkins_jobs", ["deploy-prod", "api-tests", "backup-db"])
    builds = _v("jenkins_build_numbers", {})
    rows = "\n".join(
        f'<tr><td><a href="/job/{j}/">{j}</a></td><td>Stable</td><td>#{builds.get(j, 100)} - Success</td></tr>'
        for j in jobs
    )
    return f"""<!DOCTYPE html>
<html>
<head><title>Dashboard [Jenkins]</title></head>
<body>
<div id="page-header">
<div id="header"><img src="/static/images/svgs/logo.svg" alt="Jenkins" /><h1>Jenkins</h1></div>
</div>
<div id="main-panel">
<table class="jenkins-table">
{rows}
</table>
</div>
</body>
</html>"""


def _jenkins_api_json() -> dict:
    jobs = _v("jenkins_jobs", ["deploy-prod", "api-tests", "backup-db"])
    return {
        "mode": "NORMAL",
        "nodeDescription": "the master Jenkins node",
        "numExecutors": random.randint(2, 8),
        "jobs": [
            {"_class": "hudson.model.FreeStyleProject", "name": j, "url": f"http://jenkins.internal:8080/job/{j}/", "color": "blue"}
            for j in jobs
        ],
        "useSecurity": True,
        "useCrumbs": True,
    }


def _jenkins_job_detail(job_name: str) -> dict:
    identity = _get_identity()
    builds = _v("jenkins_build_numbers", {})
    build_num = builds.get(job_name, 142)
    return {
        "name": job_name,
        "url": f"http://jenkins.internal:8080/job/{job_name}/",
        "buildable": True,
        "builds": [
            {"number": build_num, "url": f"http://jenkins.internal:8080/job/{job_name}/{build_num}/"},
            {"number": build_num - 1, "url": f"http://jenkins.internal:8080/job/{job_name}/{build_num - 1}/"},
        ],
        "lastBuild": {"number": build_num, "result": "SUCCESS"},
        "lastSuccessfulBuild": {"number": build_num, "result": "SUCCESS"},
        "healthReport": [{"description": "Build stability: No recent failures", "score": 100}],
        "property": [
            {"_class": "hudson.model.ParametersDefinitionProperty", "parameterDefinitions": [
                {"name": "DEPLOY_KEY", "type": "StringParameterDefinition", "defaultParameterValue": {"value": identity.get("deploy_key", "")}},
                {"name": "AWS_ACCESS_KEY_ID", "type": "StringParameterDefinition", "defaultParameterValue": {"value": identity.get("aws_key_id", "")}},
                {"name": "TARGET_ENV", "type": "ChoiceParameterDefinition", "choices": ["production", "staging", "development"]},
            ]},
        ],
    }


async def _handle_jenkins(reader: asyncio.StreamReader, writer: asyncio.StreamWriter):
    """Handle a Jenkins HTTP connection."""
    addr = writer.get_extra_info("peername")
    client_ip = addr[0] if addr else "unknown"
    session_id = _resolve_session(client_ip)

    _log_event(session_id, "service_connection", {
        "protocol": "jenkins",
        "client_ip": client_ip,
        "port": 8080,
    })

    try:
        method, path, headers, body = await _read_http_request(reader)
        if not method:
            return

        _log_event(session_id, "service_query", {
            "protocol": "jenkins",
            "client_ip": client_ip,
            "method": method,
            "path": path,
        })

        logger.info("[%s] Jenkins %s %s", session_id, method, path)

        jenkins_headers = {"X-Jenkins": _v("jenkins_version", "2.426.3"), "X-Jenkins-Session": _v("jenkins_session", "a1b2c3d4")}

        if path == "/api/json" or path == "/api/json/":
            writer.write(_http_response(200, json.dumps(_jenkins_api_json()), extra_headers=jenkins_headers))

        elif path.startswith("/job/") and path.endswith("/api/json"):
            job_name = path.split("/")[2]
            writer.write(_http_response(200, json.dumps(_jenkins_job_detail(job_name)), extra_headers=jenkins_headers))

        elif path.startswith("/job/"):
            # HTML page for the job
            job_name = path.split("/")[2] if len(path.split("/")) > 2 else "unknown"
            builds = _v("jenkins_build_numbers", {})
            build_num = builds.get(job_name, 142)
            html = f"<html><head><title>{job_name} [Jenkins]</title></head><body><h1>Project {job_name}</h1><p>Last build: #{build_num} - SUCCESS</p></body></html>"
            writer.write(_http_response(200, html, content_type="text/html", extra_headers=jenkins_headers))

        elif path == "/" or path == "":
            writer.write(_http_response(200, _jenkins_dashboard_html(), content_type="text/html", extra_headers=jenkins_headers))

        elif path == "/login":
            html = "<html><head><title>Sign in [Jenkins]</title></head><body><h1>Sign in to Jenkins</h1><form method='post'><input name='j_username'/><input name='j_password' type='password'/><button>Sign in</button></form></body></html>"
            writer.write(_http_response(200, html, content_type="text/html", extra_headers=jenkins_headers))

        else:
            writer.write(_http_response(404, json.dumps({"error": "not found"}), extra_headers=jenkins_headers))

        await asyncio.wait_for(writer.drain(), timeout=30)

    except (asyncio.TimeoutError, asyncio.IncompleteReadError, ConnectionResetError, BrokenPipeError):
        pass
    except Exception as e:
        logger.debug("Jenkins handler error: %s", e)
    finally:
        writer.close()
        try:
            await writer.wait_closed()
        except Exception:
            pass


# ── SSH Banner Handler ────────────────────────────────────────

async def _handle_ssh_banner(reader: asyncio.StreamReader, writer: asyncio.StreamWriter):
    """Handle SSH connections with banner exchange and key exchange stub."""
    addr = writer.get_extra_info("peername")
    client_ip = addr[0] if addr else "unknown"
    session_id = _resolve_session(client_ip)

    _log_event(session_id, "service_connection", {
        "protocol": "ssh",
        "client_ip": client_ip,
        "port": 10022,
    })

    try:
        banner = _v("ssh_banner", "SSH-2.0-OpenSSH_8.9p1 Ubuntu-3ubuntu0.6")
        writer.write((banner + "\r\n").encode())
        await asyncio.wait_for(writer.drain(), timeout=30)

        # Read client banner
        client_banner = await asyncio.wait_for(reader.readline(), timeout=30)
        client_banner_str = client_banner.decode("utf-8", errors="replace").strip()

        logger.info("[%s] SSH relay connect: %s", session_id, client_banner_str)

        _log_event(session_id, "service_query", {
            "protocol": "ssh",
            "client_ip": client_ip,
            "client_banner": client_banner_str,
        })

        # Hold connection open briefly to appear realistic
        await asyncio.sleep(5)

    except (asyncio.TimeoutError, asyncio.IncompleteReadError, ConnectionResetError, BrokenPipeError):
        pass
    except Exception as e:
        logger.debug("SSH banner handler error: %s", e)
    finally:
        writer.close()
        try:
            await writer.wait_closed()
        except Exception:
            pass


# ── Connection Limiting ────────────────────────────────────────

MAX_CONCURRENT_CONNECTIONS = 200

_conn_semaphore = None


def _throttled(handler):
    """Wrap a handler with a connection semaphore to prevent FD exhaustion."""
    async def wrapper(reader, writer):
        async with _conn_semaphore:
            await handler(reader, writer)
    return wrapper


# ── Main ──────────────────────────────────────────────────────

SERVICE_PORTS = {
    5432: ("PostgreSQL", _handle_postgres),
    6379: ("Redis", _handle_redis),
    9200: ("Elasticsearch", _handle_elasticsearch),
    8500: ("Consul", _handle_consul),
    8080: ("Jenkins", _handle_jenkins),
    10022: ("SSH relay", _handle_ssh_banner),
}


async def main():
    global _conn_semaphore
    logger.info("Starting network service handler...")

    _conn_semaphore = asyncio.Semaphore(MAX_CONCURRENT_CONNECTIONS)

    await _wait_for_identity()
    _generate_variants(_get_identity())

    servers = []
    for port, (name, handler) in SERVICE_PORTS.items():
        try:
            server = await asyncio.start_server(_throttled(handler), "0.0.0.0", port)
            servers.append(server)
            logger.info("  %s listening on :%d", name, port)
        except OSError as e:
            logger.error("  Failed to bind %s on :%d — %s", name, port, e)

    if not servers:
        logger.error("No services started, exiting")
        return

    logger.info("All services ready (%d/%d)", len(servers), len(SERVICE_PORTS))

    try:
        await asyncio.gather(*(s.serve_forever() for s in servers))
    except asyncio.CancelledError:
        pass
    finally:
        for s in servers:
            s.close()


if __name__ == "__main__":
    asyncio.run(main())
