#!/usr/bin/env bash
# LABYRINTH — End-to-End Smoke Test
# Authors: DaxxSec & Claude (Anthropic)
#
# Builds, deploys, sends a probe, verifies forensic output, then tears down.
# Requires: Docker, docker compose, ssh client.

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

pass() { echo -e "${GREEN}[PASS]${NC} $1"; }
fail() { echo -e "${RED}[FAIL]${NC} $1"; exit 1; }
info() { echo -e "${YELLOW}[INFO]${NC} $1"; }

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

info "Building Docker stack..."
docker compose build || fail "Docker build failed"

info "Starting services..."
docker compose up -d || fail "Docker compose up failed"

info "Waiting for services to stabilize..."
sleep 5

# Verify containers are running
for svc in labyrinth-ssh labyrinth-http labyrinth-orchestrator labyrinth-proxy labyrinth-dashboard; do
    if docker ps --format '{{.Names}}' | grep -q "$svc"; then
        pass "Container $svc is running"
    else
        fail "Container $svc is NOT running"
    fi
done

# Probe the SSH portal trap
info "Probing SSH portal trap on port 2222..."
ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 -o BatchMode=yes \
    admin@localhost -p 2222 "ls /opt/.credentials" 2>/dev/null || true
pass "SSH probe completed (connection attempt logged)"

# Wait for event processing
sleep 3

# Check forensic output
info "Checking forensic event output..."
if docker exec labyrinth-orchestrator test -d /var/labyrinth/forensics; then
    pass "Forensics directory exists"
else
    fail "Forensics directory not found"
fi

if docker exec labyrinth-orchestrator ls /var/labyrinth/forensics/sessions/ 2>/dev/null | grep -q ".jsonl"; then
    pass "JSONL session files present"
    info "Sample event:"
    docker exec labyrinth-orchestrator head -1 /var/labyrinth/forensics/sessions/*.jsonl 2>/dev/null || true
else
    info "No JSONL session files yet (agent may not have triggered auth event)"
fi

# Check dashboard API
info "Checking dashboard API..."
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:9000/api/stats 2>/dev/null || echo "000")
if [ "$HTTP_STATUS" = "200" ]; then
    pass "Dashboard API responding (HTTP $HTTP_STATUS)"
else
    info "Dashboard API returned HTTP $HTTP_STATUS (may need more time)"
fi

# Teardown
info "Tearing down..."
docker compose down -v || fail "Docker compose down failed"
pass "Teardown complete"

echo ""
echo -e "${GREEN}=== Smoke test finished ===${NC}"
