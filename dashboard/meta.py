"""
LABYRINTH — Central Meta-Dashboard
Aggregates data across all portal trap environments.
Authors: Stephen Stewart & Claude (Anthropic)
"""

from flask import Flask, jsonify
import json, os, glob, urllib.request

app = Flask(__name__)

META_PORT = int(os.environ.get("LABYRINTH_META_PORT", "9999"))


def _discover_environments():
    """Read environment registry files from ~/.labyrinth/environments/."""
    home = os.path.expanduser("~")
    env_dir = os.path.join(home, ".labyrinth", "environments")
    envs = []

    if not os.path.isdir(env_dir):
        return envs

    for f in sorted(glob.glob(os.path.join(env_dir, "*.json"))):
        try:
            with open(f) as fh:
                env = json.load(fh)
                envs.append(env)
        except (json.JSONDecodeError, IOError):
            continue

    return envs


def _probe_dashboard(url, path="/api/stats"):
    """Probe a dashboard URL and return its response, or None on failure."""
    try:
        req = urllib.request.Request(url + path, headers={"Accept": "application/json"})
        with urllib.request.urlopen(req, timeout=3) as resp:
            return json.loads(resp.read().decode())
    except Exception:
        return None


@app.route("/api/environments")
def environments():
    envs = _discover_environments()
    result = []

    for env in envs:
        dashboard_url = env.get("dashboard_url", "")
        if not dashboard_url and env.get("type") == "test":
            dashboard_url = "http://localhost:9000"

        health = "unknown"
        if dashboard_url:
            probe = _probe_dashboard(dashboard_url)
            health = "healthy" if probe is not None else "unreachable"

        result.append({
            "name": env.get("name", ""),
            "type": env.get("type", ""),
            "mode": env.get("mode", ""),
            "created": env.get("created", ""),
            "dashboard_url": dashboard_url,
            "health": health,
            "ports": env.get("ports", {}),
            "subnet": env.get("subnet", ""),
        })

    return jsonify({"environments": result})


@app.route("/api/aggregate/stats")
def aggregate_stats():
    envs = _discover_environments()
    agg = {
        "active_sessions": 0,
        "captured_prompts": 0,
        "total_events": 0,
        "auth_attempts": 0,
        "http_requests": 0,
        "l3_activations": 0,
        "l4_interceptions": 0,
        "max_depth_reached": 0,
        "active_containers": 0,
    }
    sources = []

    for env in envs:
        dashboard_url = env.get("dashboard_url", "")
        if not dashboard_url and env.get("type") == "test":
            dashboard_url = "http://localhost:9000"
        if not dashboard_url:
            continue

        stats = _probe_dashboard(dashboard_url, "/api/stats")
        if stats is None:
            sources.append({"name": env.get("name", ""), "status": "unreachable"})
            continue

        sources.append({"name": env.get("name", ""), "status": "ok"})
        for key in ("active_sessions", "captured_prompts", "total_events",
                     "auth_attempts", "http_requests", "l3_activations",
                     "l4_interceptions", "active_containers"):
            agg[key] += stats.get(key, 0)
        if stats.get("max_depth_reached", 0) > agg["max_depth_reached"]:
            agg["max_depth_reached"] = stats["max_depth_reached"]

    agg["sources"] = sources
    return jsonify(agg)


@app.route("/api/aggregate/sessions")
def aggregate_sessions():
    envs = _discover_environments()
    all_sessions = []

    for env in envs:
        dashboard_url = env.get("dashboard_url", "")
        if not dashboard_url and env.get("type") == "test":
            dashboard_url = "http://localhost:9000"
        if not dashboard_url:
            continue

        sessions = _probe_dashboard(dashboard_url, "/api/sessions")
        if sessions is None:
            continue

        env_name = env.get("name", "")
        if isinstance(sessions, list):
            for s in sessions:
                s["environment"] = env_name
                all_sessions.append(s)

    return jsonify(all_sessions)


if __name__ == "__main__":
    print(f"LABYRINTH Meta-Dashboard starting on port {META_PORT}")
    app.run(host="0.0.0.0", port=META_PORT, debug=os.environ.get("FLASK_DEBUG", "0") == "1")
