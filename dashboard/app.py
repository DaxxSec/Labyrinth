"""
LABYRINTH — Real-Time Capture Dashboard
Authors: Stephen Stewart & Claude (Anthropic)
"""

from flask import Flask, render_template_string, jsonify
import json, os, glob

app = Flask(__name__)
FORENSICS_DIR = "/var/labyrinth/forensics"

DASHBOARD_HTML = open("/app/dashboard/templates/index.html").read()


@app.route("/")
def index():
    return render_template_string(DASHBOARD_HTML)


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
    for f in session_files:
        with open(f) as fh:
            total_events += sum(1 for _ in fh)
    # Also count auth_events.jsonl
    auth_file = os.path.join(FORENSICS_DIR, "auth_events.jsonl")
    if os.path.exists(auth_file):
        with open(auth_file) as fh:
            total_events += sum(1 for _ in fh)
    return jsonify({
        "active_sessions": len(session_files),
        "captured_prompts": len(prompt_files),
        "total_events": total_events,
    })


if __name__ == "__main__":
    app.run(host="0.0.0.0", port=9000, debug=os.environ.get("FLASK_DEBUG", "0") == "1")
