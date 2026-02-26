# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — HTTP Honeypot (Layer 1: THRESHOLD)
#  Authors: Stephen Stewart & Claude (Anthropic)
# ═══════════════════════════════════════════════════════════════
FROM python:3.11-slim

LABEL project="labyrinth"
LABEL layer="1"
LABEL service="honeypot-http"

RUN mkdir -p /var/labyrinth/forensics/sessions /app

COPY src/layer1_portal/http_honeypot.py /app/server.py 2>/dev/null || true

# Fallback: minimal honeypot server if source not yet built
RUN if [ ! -f /app/server.py ]; then \
    echo 'from http.server import HTTPServer, SimpleHTTPRequestHandler\nimport json, datetime\n\nclass HoneypotHandler(SimpleHTTPRequestHandler):\n    def do_GET(self):\n        self.send_response(200)\n        self.send_header("Content-type", "text/html")\n        self.end_headers()\n        self.wfile.write(b"<html><head><title>Admin Panel</title></head><body><h1>Internal Admin</h1></body></html>")\n        with open("/var/labyrinth/forensics/sessions/http.jsonl", "a") as f:\n            f.write(json.dumps({"ts": str(datetime.datetime.utcnow()), "method": "GET", "path": self.path, "client": self.client_address[0]}) + "\\n")\n\nHTTPServer(("0.0.0.0", 80), HoneypotHandler).serve_forever()' > /app/server.py; \
    fi

EXPOSE 80

CMD ["python3", "/app/server.py"]
