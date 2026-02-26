# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — Real-Time Capture Dashboard
#  Authors: Stephen Stewart & Claude (Anthropic)
# ═══════════════════════════════════════════════════════════════
FROM python:3.11-slim

LABEL project="labyrinth"
LABEL layer="dashboard"

RUN pip install --no-cache-dir flask watchdog

RUN mkdir -p /var/labyrinth/forensics /app/dashboard /app/dashboard/templates
COPY dashboard/ /app/dashboard/ 2>/dev/null || true

# Fallback: embed minimal dashboard if source not yet built
RUN if [ ! -f /app/dashboard/app.py ]; then \
    python3 -c "
import textwrap, os

app_code = textwrap.dedent('''
from flask import Flask, render_template_string, jsonify
import json, os, glob

app = Flask(__name__)
FORENSICS_DIR = \"/var/labyrinth/forensics\"

DASHBOARD_HTML = open(\"/app/dashboard/templates/index.html\").read()

@app.route(\"/\")
def index():
    return render_template_string(DASHBOARD_HTML)

@app.route(\"/api/sessions\")
def sessions():
    sessions = []
    for f in sorted(glob.glob(f\"{FORENSICS_DIR}/sessions/*.jsonl\"), reverse=True)[:50]:
        with open(f) as fh:
            lines = fh.readlines()
            sessions.append({\"file\": os.path.basename(f), \"events\": len(lines), \"last\": lines[-1].strip() if lines else \"\"})
    return jsonify(sessions)

@app.route(\"/api/stats\")
def stats():
    session_files = glob.glob(f\"{FORENSICS_DIR}/sessions/*.jsonl\")
    prompt_files = glob.glob(f\"{FORENSICS_DIR}/prompts/*.txt\")
    total_events = 0
    for f in session_files:
        with open(f) as fh:
            total_events += sum(1 for _ in fh)
    return jsonify({\"active_sessions\": len(session_files), \"captured_prompts\": len(prompt_files), \"total_events\": total_events})

if __name__ == \"__main__\":
    app.run(host=\"0.0.0.0\", port=9000, debug=True)
''')

os.makedirs('/app/dashboard/templates', exist_ok=True)
with open('/app/dashboard/app.py', 'w') as f:
    f.write(app_code)
"; \
    fi

# Embed dashboard HTML template
RUN python3 -c "
html = '''<!DOCTYPE html>
<html lang=\"en\">
<head>
<meta charset=\"UTF-8\">
<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">
<title>LABYRINTH Dashboard</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { background: #0a0a0f; color: #c9d1d9; font-family: \"JetBrains Mono\", \"Fira Code\", monospace; }
  .header { padding: 24px 32px; border-bottom: 1px solid #1a1a2e; display: flex; align-items: center; gap: 16px; }
  .header h1 { color: #00ff88; font-size: 20px; letter-spacing: 2px; }
  .header .status { background: #00ff8820; color: #00ff88; padding: 4px 12px; border-radius: 4px; font-size: 11px; animation: pulse 2s infinite; }
  @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.5; } }
  .grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; padding: 24px 32px; }
  .card { background: #0d1117; border: 1px solid #1a1a2e; border-radius: 8px; padding: 20px; }
  .card h3 { color: #8b949e; font-size: 11px; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 8px; }
  .card .value { font-size: 36px; font-weight: bold; color: #00ccff; }
  .card.green .value { color: #00ff88; }
  .card.red .value { color: #ff3366; }
  .card.purple .value { color: #cc33ff; }
  .sessions { padding: 0 32px 32px; }
  .sessions h2 { color: #8b949e; font-size: 13px; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 12px; }
  .session-list { background: #0d1117; border: 1px solid #1a1a2e; border-radius: 8px; overflow: hidden; }
  .session-row { padding: 12px 16px; border-bottom: 1px solid #1a1a2e; font-size: 12px; display: flex; justify-content: space-between; }
  .session-row:last-child { border-bottom: none; }
  .session-row .name { color: #00ccff; }
  .session-row .count { color: #8b949e; }
  .empty { padding: 40px; text-align: center; color: #484f58; }
  .empty .icon { font-size: 32px; margin-bottom: 12px; }
  .footer { padding: 16px 32px; text-align: center; color: #484f58; font-size: 11px; border-top: 1px solid #1a1a2e; }
</style>
</head>
<body>
  <div class=\"header\">
    <h1>LABYRINTH</h1>
    <span class=\"status\">LIVE</span>
    <span style=\"color: #484f58; font-size: 12px; margin-left: auto;\">Test Mode Dashboard</span>
  </div>
  <div class=\"grid\">
    <div class=\"card green\"><h3>Active Sessions</h3><div class=\"value\" id=\"sessions\">-</div></div>
    <div class=\"card red\"><h3>Captured Prompts</h3><div class=\"value\" id=\"prompts\">-</div></div>
    <div class=\"card purple\"><h3>Total Events</h3><div class=\"value\" id=\"events\">-</div></div>
  </div>
  <div class=\"sessions\">
    <h2>Recent Session Activity</h2>
    <div class=\"session-list\" id=\"session-list\">
      <div class=\"empty\"><div class=\"icon\">🎯</div>Waiting for connections...<br><br><span style=\"color:#8b949e\">Point an offensive agent at <code>localhost:2222</code> or <code>localhost:8080</code></span></div>
    </div>
  </div>
  <div class=\"footer\">Project LABYRINTH · Stephen Stewart &amp; Claude (Anthropic)</div>
  <script>
    async function refresh() {
      try {
        const stats = await (await fetch(\"/api/stats\")).json();
        document.getElementById(\"sessions\").textContent = stats.active_sessions;
        document.getElementById(\"prompts\").textContent = stats.captured_prompts;
        document.getElementById(\"events\").textContent = stats.total_events;
        const sessions = await (await fetch(\"/api/sessions\")).json();
        const list = document.getElementById(\"session-list\");
        if (sessions.length > 0) {
          list.innerHTML = sessions.map(s =>
            \"<div class=session-row><span class=name>\" + s.file + \"</span><span class=count>\" + s.events + \" events</span></div>\"
          ).join(\"\");
        }
      } catch(e) { console.log(\"Dashboard refresh error:\", e); }
    }
    refresh();
    setInterval(refresh, 3000);
  </script>
</body>
</html>'''

with open('/app/dashboard/templates/index.html', 'w') as f:
    f.write(html)
"

EXPOSE 9000

CMD ["python3", "/app/dashboard/app.py"]
