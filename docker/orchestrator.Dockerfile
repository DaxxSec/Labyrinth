# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — Orchestrator
#  Authors: Stephen Stewart & Claude (Anthropic)
# ═══════════════════════════════════════════════════════════════
FROM python:3.11-slim

LABEL project="labyrinth"
LABEL layer="orchestrator"

RUN pip install --no-cache-dir pyyaml docker watchdog flask

RUN mkdir -p /var/labyrinth/forensics/sessions /app

# Copy full source tree for cross-module imports
COPY src/ /app/
COPY configs/ /app/configs/

ENV PYTHONPATH=/app

WORKDIR /app

CMD ["python3", "-m", "orchestrator"]
