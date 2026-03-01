"""
LABYRINTH — Orchestrator Health API
====================================
Authors: DaxxSec & Claude (Anthropic)

Minimal internal Flask server (port 8888) that exposes container health
data to the dashboard without requiring a Docker socket mount.
"""

import glob
import logging
import os
import threading
from datetime import datetime

from flask import Flask, jsonify, request

logger = logging.getLogger("labyrinth.orchestrator.health_api")

_app = Flask(__name__)
_docker_client = None


@_app.route("/api/health")
def health():
    return jsonify({"status": "ok"})


@_app.route("/api/containers")
def containers():
    if _docker_client is None:
        return jsonify({"infrastructure": [], "sessions": []})

    try:
        raw = _docker_client.containers.list(
            all=True, filters={"label": "project=labyrinth"}
        )
    except Exception as e:
        logger.error(f"Failed to list containers: {e}")
        return jsonify({"infrastructure": [], "sessions": []})

    infrastructure = []
    session_containers = []

    for c in raw:
        # Skip the session-template build container (always exited, not a real service)
        if c.name and "template" in c.name:
            continue
        labels = c.labels or {}
        layer = labels.get("layer", "")

        ports_list = []
        try:
            for priv, bindings in (c.ports or {}).items():
                priv_num = str(priv).rstrip("/tcp").rstrip("/udp")
                if bindings:
                    for b in bindings:
                        pub = b.get("HostPort", "")
                        if pub:
                            ports_list.append(f"{pub}:{priv_num}")
                else:
                    ports_list.append(priv_num)
        except Exception:
            pass

        entry = {
            "name": c.name or "",
            "status": c.status or "",
            "state": c.status or "",
            "ports": ", ".join(dict.fromkeys(ports_list)),
            "layer": layer,
        }

        if layer == "session":
            session_containers.append(entry)
        else:
            infrastructure.append(entry)

    return jsonify({"infrastructure": infrastructure, "sessions": session_containers})


L4_MODE_FILE = "/var/labyrinth/forensics/l4_mode.json"
L4_VALID_MODES = {"passive", "neutralize", "double_agent", "counter_intel"}


@_app.route("/api/l4/mode")
def get_l4_mode():
    """Return the current L4 interceptor mode."""
    import json as _json
    mode = "passive"
    try:
        if os.path.exists(L4_MODE_FILE):
            with open(L4_MODE_FILE, encoding="utf-8") as f:
                data = _json.load(f)
                mode = data.get("mode", "passive")
    except (ValueError, IOError):
        pass
    return jsonify({"mode": mode, "valid_modes": sorted(L4_VALID_MODES)})


@_app.route("/api/l4/mode", methods=["POST"])
def set_l4_mode():
    """Set the L4 interceptor mode. Body: {"mode": "passive|neutralize|double_agent|counter_intel"}"""
    import json as _json
    data = request.get_json(silent=True) or {}
    new_mode = data.get("mode", "")
    if new_mode not in L4_VALID_MODES:
        return jsonify({"error": f"invalid mode: {new_mode}", "valid_modes": sorted(L4_VALID_MODES)}), 400

    os.makedirs(os.path.dirname(L4_MODE_FILE), exist_ok=True)
    with open(L4_MODE_FILE, "w", encoding="utf-8") as f:
        _json.dump({"mode": new_mode, "updated": datetime.utcnow().isoformat() + "Z"}, f)

    logger.info(f"L4 mode changed to: {new_mode}")
    return jsonify({"mode": new_mode, "status": "ok"})


@_app.route("/api/l4/intel")
def get_l4_intel():
    """Return captured L4 intelligence reports."""
    import json as _json
    intel_dir = "/var/labyrinth/forensics/intel"
    reports = []
    if os.path.isdir(intel_dir):
        for fname in sorted(os.listdir(intel_dir)):
            if fname.endswith(".json"):
                try:
                    with open(os.path.join(intel_dir, fname), encoding="utf-8") as f:
                        report = _json.load(f)
                        reports.append(report.get("summary", {}))
                except (ValueError, IOError):
                    continue
    return jsonify({"intel": reports})


@_app.route("/api/reset", methods=["POST"])
def reset():
    """Kill session containers and clear forensic data."""
    if _docker_client is None:
        return jsonify({"error": "no docker client"}), 503

    removed = 0
    errors = []

    # Remove session containers
    try:
        session_containers = _docker_client.containers.list(
            all=True, filters={"label": ["project=labyrinth", "layer=session"]}
        )
        for c in session_containers:
            try:
                c.remove(force=True)
                removed += 1
            except Exception as e:
                errors.append(f"remove {c.name}: {e}")
    except Exception as e:
        errors.append(f"list containers: {e}")

    # Clear forensic data
    forensics_dir = "/var/labyrinth/forensics"
    files_cleared = 0
    for pattern in ["sessions/*.jsonl", "auth_events.jsonl", "http.jsonl"]:
        for f in glob.glob(os.path.join(forensics_dir, pattern)):
            try:
                os.remove(f)
                files_cleared += 1
            except Exception as e:
                errors.append(f"remove {f}: {e}")

    logger.info(f"Reset: removed {removed} containers, cleared {files_cleared} files")
    return jsonify({
        "containers_removed": removed,
        "files_cleared": files_cleared,
        "errors": errors,
    })


def start(docker_client):
    """Start the health API server in a daemon thread."""
    global _docker_client
    _docker_client = docker_client

    def _run():
        # Suppress Flask request logging in production
        wlog = logging.getLogger("werkzeug")
        wlog.setLevel(logging.WARNING)
        _app.run(host="0.0.0.0", port=8888, threaded=True)

    t = threading.Thread(target=_run, daemon=True, name="health-api")
    t.start()
    logger.info("Health API started on :8888")
