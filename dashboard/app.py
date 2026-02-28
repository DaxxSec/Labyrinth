"""
LABYRINTH — Real-Time Capture Dashboard
Authors: Stephen Stewart & Claude (Anthropic)
"""

from flask import Flask, render_template_string, jsonify, request
import json, os, glob, urllib.request

app = Flask(__name__)
FORENSICS_DIR = "/var/labyrinth/forensics"

LABYRINTH_ENV_NAME = os.environ.get("LABYRINTH_ENV_NAME", "default")
LABYRINTH_ENV_TYPE = os.environ.get("LABYRINTH_ENV_TYPE", "test")

DASHBOARD_HTML = open("/app/dashboard/templates/index.html").read()


def _read_jsonl(path, limit=0):
    """Read a JSONL file and return parsed lines. Malformed lines skipped."""
    events = []
    if not os.path.exists(path):
        return events
    with open(path) as fh:
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


@app.route("/api/sessions")
def sessions():
    result = []
    for f in sorted(glob.glob(f"{FORENSICS_DIR}/sessions/*.jsonl"), reverse=True)[:50]:
        with open(f) as fh:
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
        all_events.append({
            "timestamp": ev.get("timestamp", ""),
            "session_id": "",
            "layer": 1,
            "event": ev.get("event", "http_request"),
            "data": ev,
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
    # Find session file
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
        with open(prompt_file) as fh:
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


def _query_orchestrator_containers():
    """Query container data from the orchestrator's health API."""
    url = "http://labyrinth-orchestrator:8888/api/containers"
    req = urllib.request.Request(url, headers={"Accept": "application/json"})
    with urllib.request.urlopen(req, timeout=5) as resp:
        return json.loads(resp.read().decode())


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
        {"name": "L2: MIRAGE", "status": "standby", "detail": "", "sessions": 0},
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
        with open(f) as fh:
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


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=9000, debug=os.environ.get("FLASK_DEBUG", "0") == "1")
