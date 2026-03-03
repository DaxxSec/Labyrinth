#!/bin/bash
# LABYRINTH — Layer 4: PUPPETEER entrypoint
# Launches network service handler alongside the MITM interceptor.

set -e

echo "[PUPPETEER] Starting network service handler..."
python3 /app/puppeteer/services.py &

echo "[PUPPETEER] Starting MITM interceptor..."
exec mitmdump --mode transparent --listen-port 8443 --set ssl_insecure=true -s /app/puppeteer/interceptor.py
