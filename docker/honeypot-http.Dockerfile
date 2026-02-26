# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — HTTP Portal Trap (Layer 1: THRESHOLD)
#  Authors: Stephen Stewart & Claude (Anthropic)
# ═══════════════════════════════════════════════════════════════
FROM python:3.11-slim

LABEL project="labyrinth"
LABEL layer="1"
LABEL service="honeypot-http"

RUN mkdir -p /var/labyrinth/forensics/sessions /app

COPY src/layer1_portal/http_honeypot.py /app/server.py

EXPOSE 80

CMD ["python3", "/app/server.py"]
