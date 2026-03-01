# HTTP service container
FROM python:3.11-slim

LABEL managed-by="orchestrator"
LABEL layer="1"
LABEL service="http"

RUN mkdir -p /var/log/audit/sessions /app

COPY src/layer1_portal/http_honeypot.py /app/server.py

EXPOSE 80

CMD ["python3", "/app/server.py"]
