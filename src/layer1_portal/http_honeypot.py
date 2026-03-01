"""
LABYRINTH — Layer 1: THRESHOLD
HTTP Portal Trap
Authors: DaxxSec & Claude (Anthropic)

Fake admin panel that captures credentials, serves bait files,
and writes auth events for orchestrator consumption.
"""

import json
import os
import random
import secrets
from datetime import datetime
from http.server import HTTPServer, BaseHTTPRequestHandler
from urllib.parse import parse_qs, urlparse

AUTH_EVENTS_FILE = "/var/labyrinth/forensics/auth_events.jsonl"
FORENSICS_DIR = "/var/labyrinth/forensics"

# ── Randomized identity (generated fresh each container boot) ─

_COMPANIES = [
    "Meridian", "Cascade", "Pinnacle", "Stratos", "Evergreen",
    "Arclight", "Triton", "Halcyon", "Sterling", "Vanguard",
    "Keystone", "Polaris", "Ridgeline", "Ironclad", "Summit",
]
_SUFFIXES = [
    "Systems", "Technologies", "Solutions", "Digital", "Networks",
    "Labs", "Group", "Cloud", "Engineering", "Platform",
]
_TLDS = [".com", ".io", ".dev", ".tech", ".net"]
_FIRST = ["james", "sarah", "mike", "alex", "jordan", "taylor", "chris", "sam"]
_LAST = ["chen", "patel", "garcia", "wilson", "moore", "lee", "kumar", "brown"]


def _rand_hex(n):
    return secrets.token_hex(n)


def _rand_password():
    # nosec B311 — intentionally weak bait passwords for honeypot, not real credentials
    words = ["Autumn", "Silver", "Thunder", "Crystal", "Phoenix", "Harbor", "Alpine", "Ember"]
    return random.choice(words) + str(random.randint(100, 999)) + random.choice("!@#$%")  # nosec B311


def _generate_identity():
    """Generate a randomized company identity for this container boot."""
    company = random.choice(_COMPANIES)
    suffix = random.choice(_SUFFIXES)
    domain = company.lower() + random.choice(_TLDS)
    full = f"{company} {suffix}"

    users = []
    used = set()
    for _ in range(3):
        while True:
            first = random.choice(_FIRST)
            last = random.choice(_LAST)
            uname = first[0] + last
            if uname not in used:
                used.add(uname)
                break
        users.append({"email": f"{uname}@{domain}", "role": "admin", "uname": uname})

    # All values below are intentional honeypot bait — not real credentials.
    # nosec: AKIA/sk_live_/sk- prefixes are deliberate to appear authentic.
    return {
        "company": full,
        "domain": domain,
        "users": users,
        "db_pass": _rand_password(),
        "redis_token": _rand_hex(8),
        "aws_key_id": "AKIA" + _rand_hex(8).upper(),       # nosec — bait AWS key
        "aws_secret": _rand_hex(20),
        "jwt_secret": _rand_hex(16),
        "api_key": "sk-" + _rand_hex(16),                   # nosec — bait API key
        "stripe_key": "sk_live_" + _rand_hex(16),            # nosec — bait Stripe key
        "deploy_key": "sk-deploy-" + _rand_hex(12),          # nosec — bait deploy key
    }


# Generated once at import time (container startup)
_ID = _generate_identity()

# Write identity to shared forensic volume so SSH honeypot can create matching users
_IDENTITY_FILE = "/var/labyrinth/forensics/bait_identity.json"
try:
    os.makedirs(os.path.dirname(_IDENTITY_FILE), exist_ok=True)
    with open(_IDENTITY_FILE, "w") as _f:
        json.dump(_ID, _f, indent=2)
except OSError:
    pass  # Non-fatal if forensics dir isn't mounted

# ── Bait content (all strings derived from _ID for anti-fingerprinting) ──

_VERSIONS = ["3.2.1", "2.8.4", "4.1.0", "3.5.2", "2.11.3", "5.0.1"]
_PRODUCTS = ["Infrastructure Management Suite", "Admin Console", "DevOps Platform",
             "Operations Center", "Control Plane", "Service Manager"]
_VERSION = f"v{random.choice(_VERSIONS)}"
_PRODUCT = random.choice(_PRODUCTS)


def _build_login_page():
    company = _ID["company"]
    domain = _ID["domain"]
    placeholder = _ID["users"][0]["email"] if _ID["users"] else f"admin@{domain}"
    return f"""<!DOCTYPE html>
<html>
<head>
    <title>Internal Admin Panel</title>
    <style>
        body {{ font-family: -apple-system, sans-serif; background: #f5f5f5; margin: 0; padding: 0; }}
        .header {{ background: #1a1a2e; color: white; padding: 15px 30px; }}
        .header h1 {{ margin: 0; font-size: 18px; }}
        .header span {{ color: #e94560; font-size: 12px; }}
        .container {{ max-width: 400px; margin: 80px auto; background: white;
                     padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }}
        h2 {{ color: #1a1a2e; margin-top: 0; }}
        input {{ width: 100%; padding: 10px; margin: 8px 0 16px; border: 1px solid #ddd;
                border-radius: 4px; box-sizing: border-box; }}
        button {{ width: 100%; padding: 12px; background: #e94560; color: white; border: none;
                 border-radius: 4px; cursor: pointer; font-size: 14px; }}
        button:hover {{ background: #c73e54; }}
        .footer {{ text-align: center; color: #999; font-size: 12px; margin-top: 20px; }}
    </style>
</head>
<body>
    <div class="header">
        <h1>{company} Internal</h1>
        <span>RESTRICTED ACCESS — Authorized Personnel Only</span>
    </div>
    <div class="container">
        <h2>Admin Login</h2>
        <form method="POST" action="/login">
            <label>Username</label>
            <input type="text" name="username" placeholder="{placeholder}">
            <label>Password</label>
            <input type="password" name="password" placeholder="Enter password">
            <button type="submit">Sign In</button>
        </form>
        <div class="footer">{_VERSION} — {_PRODUCT}</div>
    </div>
</body>
</html>"""


def _build_dashboard_page():
    company = _ID["company"]
    return f"""<!DOCTYPE html>
<html>
<head><title>{company} Dashboard</title>
<style>
    body {{ font-family: sans-serif; background: #f5f5f5; margin: 0; }}
    .header {{ background: #1a1a2e; color: white; padding: 15px 30px; }}
    .content {{ padding: 30px; }}
    .card {{ background: white; padding: 20px; margin: 10px 0; border-radius: 8px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.1); }}
    .card h3 {{ margin-top: 0; color: #1a1a2e; }}
    .status {{ color: #27ae60; }}
    a {{ color: #e94560; }}
</style>
</head>
<body>
    <div class="header"><h1>{company} Dashboard</h1></div>
    <div class="content">
        <div class="card"><h3>Services</h3>
            <p>API Gateway: <span class="status">Running</span></p>
            <p>Database (Primary): <span class="status">Running</span></p>
            <p>Redis Cache: <span class="status">Running</span></p>
        </div>
        <div class="card"><h3>Quick Links</h3>
            <p><a href="/api/config">Service Configuration</a></p>
            <p><a href="/api/users">User Management</a></p>
            <p><a href="/.env">Environment Variables</a></p>
            <p><a href="/backup/db_dump.sql">Latest DB Backup</a></p>
        </div>
    </div>
</body>
</html>"""


def _build_fake_env():
    return f"""# Production Environment — DO NOT SHARE
APP_ENV=production
DATABASE_URL=postgresql://admin:{_ID["db_pass"]}@db-master.internal:5432/production
REDIS_URL=redis://:{_ID["redis_token"]}@redis.internal:6379/0
AWS_ACCESS_KEY_ID={_ID["aws_key_id"]}
AWS_SECRET_ACCESS_KEY={_ID["aws_secret"]}
JWT_SECRET={_ID["jwt_secret"]}
API_KEY={_ID["api_key"]}
STRIPE_SECRET_KEY={_ID["stripe_key"]}
SLACK_WEBHOOK=https://hooks.slack.com/services/T{_rand_hex(4).upper()}/B{_rand_hex(4).upper()}/{_rand_hex(12)}
"""


def _build_fake_config():
    slug = _ID["company"].split()[0].lower()
    return json.dumps({
        "services": {
            "api": {"host": "10.0.1.10", "port": 8080, "replicas": 3},
            "database": {"host": "10.0.2.10", "port": 5432, "engine": "postgresql"},
            "redis": {"host": "10.0.3.10", "port": 6379},
            "elasticsearch": {"host": "10.0.4.10", "port": 9200},
        },
        "auth": {"jwt_issuer": slug, "token_ttl": 3600},
        "deploy": {"ci_server": "jenkins.internal:8080", "artifact_bucket": f"s3://{slug}-deploys"},
    }, indent=2)


def _build_fake_users():
    users = []
    for i, u in enumerate(_ID["users"]):
        entry = {"id": i + 1, "email": u["email"], "role": "superadmin" if i == 0 else ("deployer" if i == 1 else "viewer")}
        if i == 0:
            entry["last_login"] = datetime.utcnow().strftime("%Y-%m-%dT%H:%M:%SZ")
        if i == 1:
            entry["api_key"] = _ID["deploy_key"]
        users.append(entry)
    return json.dumps({"users": users}, indent=2)


# Build all content once at startup (uses _ID identity)
LOGIN_PAGE = _build_login_page()
DASHBOARD_PAGE = _build_dashboard_page()
FAKE_ENV = _build_fake_env()
FAKE_CONFIG = _build_fake_config()
FAKE_USERS = _build_fake_users()


def _log_auth_event(src_ip: str, username: str, password: str):
    """Write auth event to shared forensic volume."""
    os.makedirs(os.path.dirname(AUTH_EVENTS_FILE), exist_ok=True)
    event = {
        "timestamp": datetime.utcnow().isoformat() + "Z",
        "event": "auth",
        "service": "http",
        "src_ip": src_ip,
        "username": username,
        "password": password,
    }
    with open(AUTH_EVENTS_FILE, "a") as f:
        f.write(json.dumps(event) + "\n")


def _log_http_event(src_ip: str, method: str, path: str, status: int):
    """Write HTTP access event to forensic log using standard schema."""
    os.makedirs(FORENSICS_DIR, exist_ok=True)
    event = {
        "timestamp": datetime.utcnow().isoformat() + "Z",
        "session_id": "",
        "layer": 1,
        "event": "http_access",
        "data": {
            "method": method,
            "path": path,
            "status": status,
            "src_ip": src_ip,
            "service": "http",
        },
    }
    filepath = os.path.join(FORENSICS_DIR, "http.jsonl")
    with open(filepath, "a") as f:
        f.write(json.dumps(event) + "\n")


BAIT_DIR = "/var/labyrinth/bait/web"


def _serve_bait(path: str) -> str | None:
    """Check for a dynamic bait file matching the request path. Returns content or None."""
    if not os.path.isdir(BAIT_DIR):
        return None
    # Normalize: /robots.txt → robots.txt, /.env → .env
    rel = path.lstrip("/")
    if not rel:
        return None
    candidate = os.path.join(BAIT_DIR, rel)
    # Prevent directory traversal
    if not os.path.realpath(candidate).startswith(os.path.realpath(BAIT_DIR)):
        return None
    if os.path.isfile(candidate):
        try:
            with open(candidate) as f:
                return f.read()
        except Exception:
            return None
    return None


def _guess_content_type(path: str) -> str:
    """Return a plausible content type for a bait file path."""
    p = path.lower()
    if p.endswith(".json"):
        return "application/json"
    if p.endswith(".html") or p.endswith(".htm"):
        return "text/html"
    if p.endswith(".csv"):
        return "text/csv"
    if p.endswith(".yml") or p.endswith(".yaml"):
        return "text/yaml"
    if p.endswith(".xml"):
        return "application/xml"
    return "text/plain"


class HoneypotHandler(BaseHTTPRequestHandler):
    """HTTP portal trap request handler."""

    # Override server identifiers — suppress real Python/BaseHTTP fingerprint
    server_version = "nginx/1.24.0"
    sys_version = ""

    def log_message(self, format, *args):
        """Suppress default logging (we use our own)."""
        pass

    def _get_client_ip(self) -> str:
        return self.client_address[0]

    def do_GET(self):
        path = urlparse(self.path).path
        client_ip = self._get_client_ip()

        # Dynamic bait files take priority — planted by `labyrinth bait drop`
        bait_content = _serve_bait(path)
        if bait_content is not None:
            content_type = _guess_content_type(path)
            self._respond(200, content_type, bait_content)
            _log_http_event(client_ip, "GET", path, 200)
            return

        if path == "/" or path == "/login":
            self._respond(200, "text/html", LOGIN_PAGE)
        elif path == "/dashboard":
            self._respond(200, "text/html", DASHBOARD_PAGE)
        elif path == "/.env":
            self._respond(200, "text/plain", FAKE_ENV)
        elif path == "/api/config":
            self._respond(200, "application/json", FAKE_CONFIG)
        elif path == "/api/users":
            self._respond(200, "application/json", FAKE_USERS)
        elif path == "/robots.txt":
            self._respond(200, "text/plain",
                          "User-agent: *\nDisallow: /admin/\nDisallow: /api/\nDisallow: /.env\n")
        else:
            self._respond(404, "text/html", "<h1>404 Not Found</h1>")

        _log_http_event(client_ip, "GET", path, 200 if path in ("/", "/login", "/dashboard", "/.env", "/api/config", "/api/users", "/robots.txt") else 404)

    def do_POST(self):
        path = urlparse(self.path).path
        client_ip = self._get_client_ip()

        content_length = int(self.headers.get("Content-Length", 0))
        body = self.rfile.read(content_length).decode("utf-8", errors="replace")

        if path == "/login":
            params = parse_qs(body)
            username = params.get("username", [""])[0]
            password = params.get("password", [""])[0]

            # Log credential capture
            _log_auth_event(client_ip, username, password)

            # Always "succeed" — redirect to dashboard
            self.send_response(302)
            self.send_header("Location", "/dashboard")
            self.end_headers()
        else:
            _log_http_event(client_ip, "POST", path, 200)
            self._respond(200, "application/json", '{"status": "ok"}')

    def _respond(self, code: int, content_type: str, body: str):
        self.send_response(code)
        self.send_header("Content-Type", content_type)
        self.send_header("Server", "nginx/1.24.0")
        self.send_header("X-Powered-By", "Express")
        self.end_headers()
        self.wfile.write(body.encode())


def main():
    port = int(os.environ.get("PORT", 80))
    server = HTTPServer(("0.0.0.0", port), HoneypotHandler)
    print(f"[THRESHOLD] HTTP portal trap listening on :{port}")
    server.serve_forever()


if __name__ == "__main__":
    main()
