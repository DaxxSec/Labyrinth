"""
LABYRINTH — Orchestrator Health API
====================================
Authors: Stephen Stewart & Claude (Anthropic)

Minimal internal Flask server (port 8888) that exposes container health
data to the dashboard without requiring a Docker socket mount.
"""

import glob
import logging
import os
import threading

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
