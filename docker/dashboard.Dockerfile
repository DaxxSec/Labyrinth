# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — Real-Time Capture Dashboard
#  Authors: DaxxSec & Claude (Anthropic)
# ═══════════════════════════════════════════════════════════════
FROM python:3.11-slim

LABEL project="labyrinth"
LABEL layer="dashboard"

RUN pip install --no-cache-dir flask watchdog

RUN mkdir -p /var/labyrinth/forensics/sessions /var/labyrinth/forensics/prompts
COPY dashboard/ /app/dashboard/

EXPOSE 9000

CMD ["python3", "/app/dashboard/app.py"]
