"""
LABYRINTH — Real-Time Capture Dashboard
Authors: DaxxSec & Claude (Anthropic)
"""

from flask import Flask, render_template_string, jsonify, request
import json, os, glob, re, time, urllib.request

app = Flask(__name__)
FORENSICS_DIR = "/var/labyrinth/forensics"

_start_time = time.time()

LABYRINTH_ENV_NAME = os.environ.get("LABYRINTH_ENV_NAME", "default")
LABYRINTH_ENV_TYPE = os.environ.get("LABYRINTH_ENV_TYPE", "test")

_TEMPLATE_PATH = os.path.join(os.path.dirname(__file__), "templates", "index.html")
try:
    with open(_TEMPLATE_PATH, encoding="utf-8") as _f:
        DASHBOARD_HTML = _f.read()
except FileNotFoundError:
    DASHBOARD_HTML = "<h1>Dashboard template not found</h1>"

# Session IDs are UUIDs or hex strings — reject anything with path separators
_SAFE_SESSION_RE = re.compile(r"^[a-zA-Z0-9_\-]+$")


def _read_jsonl(path, limit=0):
    """Read a JSONL file and return parsed lines. Malformed lines skipped."""
    events = []
    if not os.path.exists(path):
        return events
    with open(path, encoding="utf-8") as fh:
        for line in fh:
            line = line.strip()
            if not line:
                continue
            try:
                events.append(json.loads(line))
            except json.JSONDecodeError:
                continue
    if limit > 0:
        events = events[-limit:]
    return events


@app.route("/")
def index():
    return render_template_string(DASHBOARD_HTML)


@app.route("/api/identity")
def identity():
    return jsonify({"name": LABYRINTH_ENV_NAME, "type": LABYRINTH_ENV_TYPE})


@app.route("/api/health")
def health():
    uptime = time.time() - _start_time
    return jsonify({"status": "ok", "uptime_seconds": round(uptime, 1)})


@app.route("/api/sessions")
def sessions():
    result = []
    for f in sorted(glob.glob(f"{FORENSICS_DIR}/sessions/*.jsonl"), reverse=True)[:50]:
        with open(f, encoding="utf-8") as fh:
            lines = fh.readlines()
            result.append({
                "file": os.path.basename(f),
                "events": len(lines),
                "last": lines[-1].strip() if lines else "",
            })
    return jsonify(result)


@app.route("/api/stats")
def stats():
    session_files = glob.glob(f"{FORENSICS_DIR}/sessions/*.jsonl")
    prompt_files = glob.glob(f"{FORENSICS_DIR}/prompts/*.txt")
    total_events = 0
    active_sessions = 0
    auth_attempts = 0
    http_requests = 0
    l3_activations = 0
    l4_interceptions = 0
    max_depth_reached = 0

    for f in session_files:
        session_ended = False
        for ev in _read_jsonl(f):
            total_events += 1
            event_type = ev.get("event", "")
            if event_type == "session_end":
                session_ended = True
            elif event_type == "blindfold_activated":
                l3_activations += 1
            elif event_type == "api_intercepted":
                l4_interceptions += 1
            data = ev.get("data", {})
            depth = data.get("depth", 0) if isinstance(data, dict) else 0
            if isinstance(depth, (int, float)) and depth > max_depth_reached:
                max_depth_reached = int(depth)
        if not session_ended:
            active_sessions += 1

    # Auth events
    auth_file = os.path.join(FORENSICS_DIR, "auth_events.jsonl")
    if os.path.exists(auth_file):
        auth_events = _read_jsonl(auth_file)
        auth_attempts = len(auth_events)
        total_events += auth_attempts

    # HTTP log events
    http_file = os.path.join(FORENSICS_DIR, "http.jsonl")
    if os.path.exists(http_file):
        http_events = _read_jsonl(http_file)
        http_requests = len(http_events)
        total_events += http_requests

    # Active containers count
    active_containers = 0
    try:
        data = _query_orchestrator_containers()
        all_c = data.get("infrastructure", []) + data.get("sessions", [])
        active_containers = sum(1 for c in all_c if c.get("state") == "running")
    except Exception:
        pass

    return jsonify({
        "active_sessions": active_sessions,
        "captured_prompts": len(prompt_files),
        "total_events": total_events,
        "auth_attempts": auth_attempts,
        "http_requests": http_requests,
        "l3_activations": l3_activations,
        "l4_interceptions": l4_interceptions,
        "max_depth_reached": max_depth_reached,
        "active_containers": active_containers,
    })


@app.route("/api/events")
def events():
    limit = request.args.get("limit", 100, type=int)
    event_type = request.args.get("type", "")
    session_id = request.args.get("session_id", "")

    all_events = []

    # Session events
    for f in glob.glob(f"{FORENSICS_DIR}/sessions/*.jsonl"):
        for ev in _read_jsonl(f):
            if "layer" not in ev:
                # Legacy flat event — wrap in standard schema
                ev = {
                    "timestamp": ev.get("timestamp", ""),
                    "session_id": ev.get("session_id", ""),
                    "layer": 1,
                    "event": ev.get("event", "unknown"),
                    "data": {k: v for k, v in ev.items() if k not in ("timestamp", "session_id", "layer", "event")},
                    "source": "session",
                }
            else:
                ev.setdefault("source", "session")
            all_events.append(ev)

    # Auth events — normalize to common schema
    auth_file = os.path.join(FORENSICS_DIR, "auth_events.jsonl")
    for ev in _read_jsonl(auth_file):
        all_events.append({
            "timestamp": ev.get("timestamp", ""),
            "session_id": "",
            "layer": 1,
            "event": ev.get("event", "auth_attempt"),
            "data": ev,
            "source": "auth",
        })

    # HTTP log events
    http_file = os.path.join(FORENSICS_DIR, "http.jsonl")
    for ev in _read_jsonl(http_file):
        if "layer" in ev and "data" in ev and isinstance(ev["data"], dict):
            # Already in standard schema — pass through
            ev.setdefault("source", "http")
            all_events.append(ev)
        else:
            # Legacy flat format — wrap
            all_events.append({
                "timestamp": ev.get("timestamp", ""),
                "session_id": "",
                "layer": 1,
                "event": ev.get("event", "http_request"),
                "data": {k: v for k, v in ev.items() if k not in ("timestamp", "session_id", "layer", "event")},
                "source": "http",
            })

    # Filter
    if event_type:
        all_events = [e for e in all_events if e.get("event") == event_type]
    if session_id:
        all_events = [e for e in all_events if e.get("session_id") == session_id]

    # Sort by timestamp desc
    all_events.sort(key=lambda e: e.get("timestamp", ""), reverse=True)

    total = len(all_events)
    all_events = all_events[:limit]

    return jsonify({"events": all_events, "total": total})


@app.route("/api/auth")
def auth():
    limit = request.args.get("limit", 50, type=int)
    auth_file = os.path.join(FORENSICS_DIR, "auth_events.jsonl")
    auth_events = _read_jsonl(auth_file)
    auth_events.sort(key=lambda e: e.get("timestamp", ""), reverse=True)
    auth_events = auth_events[:limit]
    return jsonify({"auth_events": auth_events})


@app.route("/api/sessions/<session_id>")
def session_detail(session_id):
    if not _SAFE_SESSION_RE.match(session_id):
        return jsonify({"error": "invalid session id"}), 400
    session_file = os.path.join(FORENSICS_DIR, "sessions", f"{session_id}.jsonl")
    if not os.path.exists(session_file):
        return jsonify({"error": "session not found"}), 404

    events = _read_jsonl(session_file)
    max_depth = 0
    l3_activated = False
    layers_triggered = set()

    for ev in events:
        layer = ev.get("layer", 0)
        layers_triggered.add(layer)
        event_type = ev.get("event", "")
        if event_type == "blindfold_activated":
            l3_activated = True
        data = ev.get("data", {})
        depth = data.get("depth", 0) if isinstance(data, dict) else 0
        if isinstance(depth, (int, float)) and depth > max_depth:
            max_depth = int(depth)

    first_seen = events[0].get("timestamp", "") if events else ""
    last_seen = events[-1].get("timestamp", "") if events else ""

    # Check for captured prompt
    prompt_file = os.path.join(FORENSICS_DIR, "prompts", f"{session_id}.txt")
    has_prompts = os.path.exists(prompt_file)
    prompt_text = ""
    if has_prompts:
        with open(prompt_file, encoding="utf-8") as fh:
            prompt_text = fh.read()

    return jsonify({
        "session_id": session_id,
        "events": events,
        "max_depth": max_depth,
        "l3_activated": l3_activated,
        "layers_triggered": sorted(layers_triggered),
        "first_seen": first_seen,
        "last_seen": last_seen,
        "has_prompts": has_prompts,
        "prompt_text": prompt_text,
    })


@app.route("/api/sessions/<session_id>/analysis")
def session_analysis(session_id):
    if not _SAFE_SESSION_RE.match(session_id):
        return jsonify({"error": "invalid session id"}), 400
    session_file = os.path.join(FORENSICS_DIR, "sessions", f"{session_id}.jsonl")
    if not os.path.exists(session_file):
        return jsonify({"error": "session not found"}), 404

    events = _read_jsonl(session_file)

    analysis = {
        "session_id": session_id,
        "total_events": len(events),
        "duration_seconds": _compute_duration(events),
        "layers_reached": sorted(set(ev.get("layer", 0) for ev in events)),
        "max_depth": max((ev.get("data", {}).get("depth", 0) for ev in events if isinstance(ev.get("data"), dict)), default=0),
        "confusion_score": _compute_confusion_score(events),
        "phases": _extract_phases(events),
        "event_breakdown": _count_event_types(events),
        "key_moments": _extract_key_moments(events),
        "l3_activated": any(ev.get("event") == "blindfold_activated" for ev in events),
        "l4_active": any(ev.get("event") == "api_intercepted" for ev in events),
    }
    return jsonify(analysis)


def _compute_duration(events):
    """Seconds between first and last event timestamp."""
    if len(events) < 2:
        return 0
    from datetime import datetime as dt
    try:
        first = dt.fromisoformat(events[0].get("timestamp", "").rstrip("Z"))
        last = dt.fromisoformat(events[-1].get("timestamp", "").rstrip("Z"))
        return max(0, (last - first).total_seconds())
    except (ValueError, TypeError):
        return 0


def _compute_confusion_score(events):
    """Score 0-100 based on command retries, auth loops, depth oscillation, repeated paths."""
    score = 0
    command_counts = {}
    auth_count = 0
    depth_changes = 0
    prev_depth = 0
    path_counts = {}

    for ev in events:
        event_type = ev.get("event", "")
        data = ev.get("data", {}) if isinstance(ev.get("data"), dict) else {}

        if event_type == "command":
            cmd = data.get("command", "")
            command_counts[cmd] = command_counts.get(cmd, 0) + 1

        if event_type in ("auth_attempt", "auth"):
            auth_count += 1

        if event_type == "depth_increase":
            depth_changes += 1
            new_depth = data.get("new_depth", 0)
            if isinstance(new_depth, (int, float)) and isinstance(prev_depth, (int, float)):
                if new_depth < prev_depth:
                    score += 5  # depth oscillation
                prev_depth = new_depth

        if event_type == "http_access":
            path = data.get("path", "")
            path_counts[path] = path_counts.get(path, 0) + 1

    # Repeated commands (same command > 3 times)
    repeated_cmds = sum(1 for c in command_counts.values() if c > 3)
    score += min(repeated_cmds * 8, 30)

    # Auth loops
    if auth_count > 5:
        score += min((auth_count - 5) * 3, 15)

    # Depth oscillation bonus
    if depth_changes > 5:
        score += min((depth_changes - 5) * 2, 15)

    # Repeated path fetches
    repeated_paths = sum(1 for c in path_counts.values() if c > 3)
    score += min(repeated_paths * 5, 20)

    # Blindfold activation is a strong signal
    if any(ev.get("event") == "blindfold_activated" for ev in events):
        score += 15

    # API interception
    if any(ev.get("event") == "api_intercepted" for ev in events):
        score += 10

    return min(score, 100)


def _extract_phases(events):
    """Return list of behavioral phases detected in the session."""
    if not events:
        return []

    phase_events = {
        "reconnaissance": [],
        "credential_discovery": [],
        "initial_access": [],
        "escalation": [],
        "confusion": [],
        "blindfold": [],
        "interception": [],
    }

    for ev in events:
        event_type = ev.get("event", "")
        ts = ev.get("timestamp", "")

        if event_type in ("http_access", "connection"):
            phase_events["reconnaissance"].append(ts)
        elif event_type in ("auth_attempt", "auth"):
            phase_events["credential_discovery"].append(ts)
        elif event_type == "container_spawned":
            phase_events["initial_access"].append(ts)
        elif event_type in ("depth_increase", "command"):
            phase_events["escalation"].append(ts)
        elif event_type == "blindfold_activated":
            phase_events["blindfold"].append(ts)
        elif event_type == "api_intercepted":
            phase_events["interception"].append(ts)

    # Check for confusion: repeated commands after depth_increase
    cmd_after_depth = False
    saw_depth = False
    cmd_repeats = {}
    for ev in events:
        if ev.get("event") == "depth_increase":
            saw_depth = True
        if saw_depth and ev.get("event") == "command":
            data = ev.get("data", {}) if isinstance(ev.get("data"), dict) else {}
            cmd = data.get("command", "")
            cmd_repeats[cmd] = cmd_repeats.get(cmd, 0) + 1
            if cmd_repeats[cmd] > 2:
                phase_events["confusion"].append(ev.get("timestamp", ""))

    phases = []
    for phase_name in ["reconnaissance", "credential_discovery", "initial_access",
                       "escalation", "confusion", "blindfold", "interception"]:
        timestamps = phase_events[phase_name]
        if timestamps:
            phases.append({
                "phase": phase_name,
                "start": timestamps[0],
                "end": timestamps[-1],
                "events": len(timestamps),
            })

    return phases


def _count_event_types(events):
    """Count events by type."""
    counts = {}
    for ev in events:
        t = ev.get("event", "unknown")
        counts[t] = counts.get(t, 0) + 1
    return counts


def _extract_key_moments(events):
    """Extract notable moments from a session."""
    moments = []
    seen = set()

    for ev in events:
        event_type = ev.get("event", "")
        ts = ev.get("timestamp", "")
        layer = ev.get("layer", 0)
        data = ev.get("data", {}) if isinstance(ev.get("data"), dict) else {}

        if event_type in ("auth_attempt", "auth") and "first_auth" not in seen:
            seen.add("first_auth")
            user = data.get("username", "?")
            passwd = data.get("password", "****")
            moments.append({
                "timestamp": ts,
                "event": event_type,
                "description": f"First credential capture: {user} / {passwd}",
                "layer": layer,
            })

        if event_type == "container_spawned" and "first_container" not in seen:
            seen.add("first_container")
            moments.append({
                "timestamp": ts,
                "event": event_type,
                "description": "Session container established",
                "layer": layer,
            })

        if event_type == "depth_increase" and "first_depth" not in seen:
            seen.add("first_depth")
            moments.append({
                "timestamp": ts,
                "event": event_type,
                "description": f"First depth increase to {data.get('new_depth', '?')}",
                "layer": layer,
            })

        if event_type == "blindfold_activated" and "blindfold" not in seen:
            seen.add("blindfold")
            moments.append({
                "timestamp": ts,
                "event": event_type,
                "description": f"BLINDFOLD activated at depth {data.get('depth', '?')}",
                "layer": layer,
            })

        if event_type == "api_intercepted" and "first_intercept" not in seen:
            seen.add("first_intercept")
            domain = data.get("domain", "?")
            moments.append({
                "timestamp": ts,
                "event": event_type,
                "description": f"First API interception: {domain}",
                "layer": layer,
            })

    # Add last event
    if events:
        last = events[-1]
        moments.append({
            "timestamp": last.get("timestamp", ""),
            "event": "session_end",
            "description": "Last recorded event",
            "layer": last.get("layer", 0),
        })

    return moments


def _query_orchestrator_containers():
    """Query container data from the orchestrator's health API."""
    url = "http://labyrinth-orchestrator:8888/api/containers"
    req = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=5) as resp:
        return json.loads(resp.read().decode())


@app.route("/api/l4/mode")
def l4_mode_get():
    """Proxy L4 mode read to the orchestrator's health API."""
    try:
        url = "http://labyrinth-orchestrator:8888/api/l4/mode"
        req = urllib.request.Request(url, headers={"Accept": "application/json"})
        with urllib.request.urlopen(req, timeout=5) as resp:
            return app.response_class(resp.read(), mimetype="application/json")
    except Exception:
        # Fallback: read mode file directly from shared volume
        mode_file = os.path.join(FORENSICS_DIR, "l4_mode.json")
        mode = "passive"
        if os.path.exists(mode_file):
            try:
                with open(mode_file, encoding="utf-8") as f:
                    data = json.load(f)
                    mode = data.get("mode", "passive")
            except (json.JSONDecodeError, IOError):
                pass
        return jsonify({"mode": mode, "valid_modes": ["counter_intel", "double_agent", "neutralize", "passive"]})


@app.route("/api/l4/mode", methods=["POST"])
def l4_mode_set():
    """Proxy L4 mode change to the orchestrator's health API."""
    try:
        url = "http://labyrinth-orchestrator:8888/api/l4/mode"
        body = request.get_data()
        req = urllib.request.Request(url, data=body, method="POST",
                                     headers={"Content-Type": "application/json",
                                              "Accept": "application/json"})
        with urllib.request.urlopen(req, timeout=5) as resp:
            return app.response_class(resp.read(), mimetype="application/json")
    except Exception as e:
        return jsonify({"error": str(e)}), 502


@app.route("/api/l4/intel")
def l4_intel():
    """Return captured L4 intelligence summaries."""
    intel_dir = os.path.join(FORENSICS_DIR, "intel")
    reports = []
    if os.path.isdir(intel_dir):
        for fname in sorted(os.listdir(intel_dir)):
            if fname.endswith(".json"):
                try:
                    with open(os.path.join(intel_dir, fname), encoding="utf-8") as f:
                        report = json.load(f)
                        reports.append(report.get("summary", {}))
                except (json.JSONDecodeError, IOError):
                    continue
    return jsonify({"intel": reports})


@app.route("/api/reset", methods=["POST"])
def reset():
    """Proxy reset request to the orchestrator's health API."""
    try:
        url = "http://labyrinth-orchestrator:8888/api/reset"
        req = urllib.request.Request(url, data=b"", method="POST",
                                     headers={"Accept": "application/json"})
        with urllib.request.urlopen(req, timeout=15) as resp:
            return app.response_class(resp.read(), mimetype="application/json")
    except Exception as e:
        return jsonify({"error": str(e)}), 502


@app.route("/api/containers")
def containers():
    try:
        data = _query_orchestrator_containers()
        return jsonify(data)
    except Exception:
        return jsonify({"infrastructure": [], "sessions": []})


@app.route("/api/layers")
def layers():
    # Derive layer status from live data
    layer_statuses = [
        {"name": "L0: FOUNDATION", "status": "standby", "detail": "", "sessions": 0},
        {"name": "L1: THRESHOLD", "status": "standby", "detail": "", "sessions": 0},
        {"name": "L2: MINOTAUR", "status": "standby", "detail": "", "sessions": 0},
        {"name": "L3: BLINDFOLD", "status": "standby", "detail": "", "sessions": 0},
        {"name": "L4: INTERCEPT", "status": "standby", "detail": "", "sessions": 0},
    ]

    # Check container status for L0/L1
    try:
        data = _query_orchestrator_containers()
        all_c = data.get("infrastructure", []) + data.get("sessions", [])
        running = [c for c in all_c if c.get("state") == "running"]
        infra_running = [c for c in running if c.get("layer", "") not in ("session", "")]
        if infra_running:
            layer_statuses[0]["status"] = "active"
            layer_statuses[0]["detail"] = f"{len(infra_running)} infrastructure containers"

        portal_names = {"labyrinth-ssh", "labyrinth-http"}
        portal_running = [c for c in running if c.get("name", "") in portal_names]
        if portal_running:
            layer_statuses[1]["status"] = "active"
            layer_statuses[1]["detail"] = f"{len(portal_running)} portal services"

        # Check for proxy — mark as ready but not active until sessions use it
        proxy_running = [c for c in running if "proxy" in c.get("name", "")]
        if proxy_running:
            layer_statuses[4]["detail"] = "MITM proxy running"
    except Exception:
        pass

    # Scan session events for L2/L3/L4
    l2_sessions = set()
    l3_sessions = set()
    l4_sessions = set()

    for f in glob.glob(f"{FORENSICS_DIR}/sessions/*.jsonl"):
        for ev in _read_jsonl(f):
            sid = ev.get("session_id", "")
            event_type = ev.get("event", "")
            if event_type in ("container_spawned", "depth_increase"):
                l2_sessions.add(sid)
            elif event_type == "blindfold_activated":
                l3_sessions.add(sid)
            elif event_type == "api_intercepted":
                l4_sessions.add(sid)

    if l2_sessions:
        layer_statuses[2]["status"] = "active"
        layer_statuses[2]["detail"] = f"{len(l2_sessions)} sessions with depth > 1"
        layer_statuses[2]["sessions"] = len(l2_sessions)
    if l3_sessions:
        layer_statuses[3]["status"] = "active"
        layer_statuses[3]["detail"] = f"{len(l3_sessions)} sessions with blindfold"
        layer_statuses[3]["sessions"] = len(l3_sessions)
    if l4_sessions:
        layer_statuses[4]["status"] = "active"
        layer_statuses[4]["detail"] = f"{len(l4_sessions)} intercepted sessions"
        layer_statuses[4]["sessions"] = len(l4_sessions)

    return jsonify({"layers": layer_statuses})


@app.route("/api/prompts")
def prompts():
    result = []
    for f in sorted(glob.glob(f"{FORENSICS_DIR}/prompts/*.txt"), reverse=True):
        session_id = os.path.splitext(os.path.basename(f))[0]
        with open(f, encoding="utf-8") as fh:
            content = fh.read()

        # Parse sections separated by "--- TIMESTAMP | DOMAIN ---"
        sections = content.split("---")
        i = 0
        while i < len(sections):
            section = sections[i].strip()
            if "|" in section and i + 1 < len(sections):
                # This is a header line: TIMESTAMP | DOMAIN
                parts = section.split("|", 1)
                timestamp = parts[0].strip()
                domain = parts[1].strip() if len(parts) > 1 else ""
                i += 1
                text = sections[i].strip() if i < len(sections) else ""
                result.append({
                    "session_id": session_id,
                    "timestamp": timestamp,
                    "domain": domain,
                    "text": text,
                })
            elif section:
                # Plain text without header
                result.append({
                    "session_id": session_id,
                    "timestamp": "",
                    "domain": "",
                    "text": section,
                })
            i += 1

    return jsonify({"prompts": result})


@app.route("/api/bait-identity")
def bait_identity():
    identity_file = os.path.join(FORENSICS_DIR, "bait_identity.json")
    if not os.path.exists(identity_file):
        return jsonify({"error": "no bait identity found"}), 404
    with open(identity_file, encoding="utf-8") as f:
        data = json.load(f)
    # Add bait file paths that are served by the HTTP honeypot
    data["bait_paths"] = ["/.env", "/api/config", "/api/users", "/robots.txt", "/backup/db_dump.sql"]
    # Add any CLI-dropped bait files from the web bait directory
    bait_dir = "/var/labyrinth/bait/web"
    if os.path.isdir(bait_dir):
        for fname in os.listdir(bait_dir):
            path = "/" + fname
            if path not in data["bait_paths"]:
                data["bait_paths"].append(path)
    return jsonify(data)


@app.route("/api/container-logs")
def container_logs():
    service = request.args.get("service", "")
    lines = min(int(request.args.get("lines", 50)), 200)
    if not service:
        return jsonify({"error": "service parameter required"}), 400
    allowed_services = {
        "honeypot-ssh": "labyrinth-ssh",
        "honeypot-http": "labyrinth-http",
        "orchestrator": "labyrinth-orchestrator",
        "proxy": "labyrinth-proxy",
        "dashboard": "labyrinth-dashboard",
    }
    if service not in allowed_services:
        return jsonify({"error": "unknown service"}), 400
    try:
        output = _docker_logs(allowed_services[service], lines)
        return jsonify({"service": service, "lines": output})
    except Exception as e:
        return jsonify({"service": service, "lines": [], "error": str(e)})


def _docker_logs(container_name, tail=50):
    """Fetch container logs via Docker Engine API over Unix socket."""
    import http.client
    import socket

    sock_path = "/var/run/docker.sock"
    if not os.path.exists(sock_path):
        return []

    class DockerSocket(http.client.HTTPConnection):
        def __init__(self):
            super().__init__("localhost")
        def connect(self):
            self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
            self.sock.settimeout(5)
            self.sock.connect(sock_path)

    conn = DockerSocket()
    conn.request("GET", f"/containers/{container_name}/logs?stdout=1&stderr=1&tail={tail}")
    resp = conn.getresponse()
    if resp.status != 200:
        return []
    raw = resp.read()
    conn.close()

    # Docker multiplexed stream: each frame has 8-byte header
    # [stream_type(1) | padding(3) | size(4)] + payload
    lines = []
    pos = 0
    while pos + 8 <= len(raw):
        size = int.from_bytes(raw[pos+4:pos+8], "big")
        pos += 8
        if pos + size > len(raw):
            break
        chunk = raw[pos:pos+size].decode("utf-8", errors="replace").strip()
        if chunk:
            lines.extend(chunk.split("\n"))
        pos += size

    # Fallback: if no frames parsed, treat as plain text
    if not lines and raw:
        lines = raw.decode("utf-8", errors="replace").strip().split("\n")

    return lines


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=9000, threaded=True,
            debug=os.environ.get("FLASK_DEBUG", "0") == "1")
