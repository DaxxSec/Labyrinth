# ═══════════════════════════════════════════════════════════════
#  LABYRINTH — MITM Proxy (Layer 4: PUPPETEER)
#  Authors: DaxxSec & Claude (Anthropic)
# ═══════════════════════════════════════════════════════════════
FROM python:3.11-slim

LABEL project="labyrinth"
LABEL layer="4"
LABEL service="proxy"

RUN pip install --no-cache-dir mitmproxy cryptography

RUN mkdir -p /var/labyrinth/forensics/prompts /app/puppeteer
COPY src/layer4_puppeteer/ /app/puppeteer/
RUN chmod +x /app/puppeteer/start.sh

WORKDIR /app

# Generate mitmproxy CA on first run
RUN mitmdump --version || true

CMD ["/app/puppeteer/start.sh"]
