# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — Orchestrator
#  Authors: Stephen Stewart & Claude (Anthropic)
# ═══════════════════════════════════════════════════════════════
FROM python:3.11-slim

LABEL project="labyrinth"
LABEL layer="orchestrator"

RUN pip install --no-cache-dir pyyaml docker watchdog

RUN mkdir -p /var/labyrinth/forensics /app
COPY src/orchestrator/ /app/orchestrator/ 2>/dev/null || true
COPY configs/ /app/configs/ 2>/dev/null || true

WORKDIR /app

CMD ["python3", "-m", "orchestrator"]
