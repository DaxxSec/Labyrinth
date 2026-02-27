#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════
#  Project LABYRINTH — End-to-End Smoke Test
#  Deploys the full stack, exercises every layer's entry point,
#  verifies forensic capture, and tears down cleanly.
#
#  Usage: ./scripts/smoke-test.sh
#  Exit:  0 = all passed, 1 = failures detected
# ═══════════════════════════════════════════════════════════════
set -uo pipefail

# ── Colors ────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
MAGENTA='\033[0;35m'
DIM='\033[2m'
BOLD='\033[1m'
NC='\033[0m'

# ── Logging ───────────────────────────────────────────────────
info()    { echo -e "  ${GREEN}[+]${NC} $1"; }
warn()    { echo -e "  ${YELLOW}[!]${NC} $1"; }
error()   { echo -e "  ${RED}[✗]${NC} $1"; }
section() { echo -e "\n  ${MAGENTA}━━━ $1 ━━━${NC}\n"; }

# ── Test counters ─────────────────────────────────────────────
PASS=0
FAIL=0
TESTS=()

pass() {
    PASS=$((PASS + 1))
    TESTS+=("${GREEN}PASS${NC}  $1")
    info "$1"
}

fail() {
    FAIL=$((FAIL + 1))
    TESTS+=("${RED}FAIL${NC}  $1")
    error "$1"
}

assert() {
    local desc="$1"
    shift
    if "$@" > /dev/null 2>&1; then
        pass "$desc"
    else
        fail "$desc"
    fi
}

assert_contains() {
    local desc="$1" content="$2" pattern="$3"
    if echo "$content" | grep -q "$pattern"; then
        pass "$desc"
    else
        fail "$desc"
    fi
}

# ── Constants ─────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"
ENV_NAME="smoke-test"
CLI_BIN="/tmp/labyrinth-smoke"
ENV_DIR="${HOME}/.labyrinth/environments"
TIMEOUT=90
DEPLOYED=false

# ── Cleanup on exit ───────────────────────────────────────────
cleanup() {
    if [ "$DEPLOYED" = true ]; then
        section "Emergency Cleanup"
        warn "Tearing down smoke-test environment..."
        "$CLI_BIN" teardown "$ENV_NAME" 2>/dev/null || true
        # Belt-and-suspenders: direct compose down
        COMPOSE_PROJECT_NAME="labyrinth-${ENV_NAME}" \
            docker compose -f "${REPO_DIR}/docker-compose.yml" down -v 2>/dev/null || true
    fi
    rm -f "$CLI_BIN"
}
trap cleanup EXIT

# ── Banner ────────────────────────────────────────────────────
echo ""
echo -e "  ${GREEN}██╗      █████╗ ██████╗ ██╗   ██╗██████╗ ██╗███╗   ██╗████████╗██╗  ██╗${NC}"
echo -e "  ${GREEN}██║     ██╔══██╗██╔══██╗╚██╗ ██╔╝██╔══██╗██║████╗  ██║╚══██╔══╝██║  ██║${NC}"
echo -e "  ${GREEN}██║     ███████║██████╔╝ ╚████╔╝ ██████╔╝██║██╔██╗ ██║   ██║   ███████║${NC}"
echo -e "  ${GREEN}██║     ██╔══██║██╔══██╗  ╚██╔╝  ██╔══██╗██║██║╚██╗██║   ██║   ██╔══██║${NC}"
echo -e "  ${GREEN}███████╗██║  ██║██████╔╝   ██║   ██║  ██║██║██║ ╚████║   ██║   ██║  ██║${NC}"
echo -e "  ${GREEN}╚══════╝╚═╝  ╚═╝╚═════╝    ╚═╝   ╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝   ╚═╝   ╚═╝  ╚═╝${NC}"
echo ""
echo -e "  ${DIM}End-to-End Smoke Test${NC}"
echo ""

# ══════════════════════════════════════════════════════════════
#  Step 1: Preflight
# ══════════════════════════════════════════════════════════════
section "Preflight Checks"

# Docker daemon
if ! docker info > /dev/null 2>&1; then
    error "Docker daemon is not running. Start Docker and try again."
    exit 1
fi
info "Docker daemon is running"

# Go compiler (needed to build CLI)
if ! command -v go > /dev/null 2>&1; then
    error "Go is required to build the CLI. Install Go 1.21+ and try again."
    exit 1
fi
info "Go found: $(go version | awk '{print $3}')"

# Port availability (macOS-compatible)
for port in 2222 8080 9000; do
    if lsof -i ":${port}" -sTCP:LISTEN > /dev/null 2>&1; then
        error "Port ${port} is already in use. Free it and try again."
        exit 1
    fi
    info "Port ${port} is available"
done

# No leftover smoke-test containers
if docker ps -a --filter "label=project=labyrinth" --format '{{.Names}}' 2>/dev/null | grep -q .; then
    warn "Existing labyrinth containers detected — cleaning up first"
    COMPOSE_PROJECT_NAME="labyrinth-${ENV_NAME}" \
        docker compose -f "${REPO_DIR}/docker-compose.yml" down -v 2>/dev/null || true
fi

# No stale registry entry
if [ -f "${ENV_DIR}/${ENV_NAME}.json" ]; then
    warn "Stale registry entry found — removing"
    rm -f "${ENV_DIR}/${ENV_NAME}.json"
fi

info "All preflight checks passed"

# ══════════════════════════════════════════════════════════════
#  Step 2: Build CLI
# ══════════════════════════════════════════════════════════════
section "Building CLI"

cd "${REPO_DIR}/cli"
if go build -o "$CLI_BIN" .; then
    pass "CLI binary built: ${CLI_BIN}"
else
    fail "CLI build failed"
    exit 1
fi

# ══════════════════════════════════════════════════════════════
#  Step 3: Deploy
# ══════════════════════════════════════════════════════════════
section "Deploying Environment: ${ENV_NAME}"

cd "$REPO_DIR"
if "$CLI_BIN" deploy -t "$ENV_NAME" 2>&1; then
    DEPLOYED=true
    pass "Deploy command succeeded"
else
    fail "Deploy command failed"
    exit 1
fi

# ══════════════════════════════════════════════════════════════
#  Step 4: Wait for Healthy
# ══════════════════════════════════════════════════════════════
section "Waiting for Services"

EXPECTED_CONTAINERS="labyrinth-ssh labyrinth-http labyrinth-orchestrator labyrinth-proxy labyrinth-dashboard"
elapsed=0

while [ $elapsed -lt $TIMEOUT ]; do
    running=0
    for name in $EXPECTED_CONTAINERS; do
        if docker ps --filter "name=^${name}$" --filter "status=running" --format '{{.Names}}' 2>/dev/null | grep -q "^${name}$"; then
            running=$((running + 1))
        fi
    done
    if [ $running -ge 5 ]; then
        break
    fi
    echo -e "    ${DIM}${running}/5 containers running (${elapsed}s / ${TIMEOUT}s)${NC}"
    sleep 5
    elapsed=$((elapsed + 5))
done

if [ $elapsed -ge $TIMEOUT ]; then
    fail "Timed out waiting for containers (only ${running}/5 running after ${TIMEOUT}s)"
    docker ps -a --filter "label=project=labyrinth" --format "table {{.Names}}\t{{.Status}}" 2>/dev/null || true
    exit 1
fi

pass "All 5 services running (${elapsed}s)"

# Give services a moment to bind their ports
sleep 3

# ══════════════════════════════════════════════════════════════
#  Step 5: Verify Service Endpoints
# ══════════════════════════════════════════════════════════════
section "Verifying Service Endpoints"

# HTTP portal trap (port 8080)
http_response=$(curl -s --max-time 10 http://localhost:8080/ 2>/dev/null || echo "")
assert_contains "HTTP portal trap responds with login page" "$http_response" "Admin Login"

# Dashboard (port 9000)
dash_response=$(curl -s --max-time 10 http://localhost:9000/ 2>/dev/null || echo "")
assert_contains "Dashboard responds with LABYRINTH page" "$dash_response" "LABYRINTH"

# Dashboard API /api/stats
stats_response=$(curl -s --max-time 10 http://localhost:9000/api/stats 2>/dev/null || echo "")
assert_contains "Dashboard /api/stats returns active_sessions" "$stats_response" "active_sessions"

# SSH port accepting connections
assert "SSH portal trap accepting connections on :2222" nc -z -w 5 localhost 2222

# ══════════════════════════════════════════════════════════════
#  Step 6: Exercise HTTP Bait Endpoints
# ══════════════════════════════════════════════════════════════
section "Exercising HTTP Bait Endpoints"

# POST /login with fake creds → should return 302
login_status=$(curl -s -o /dev/null -w "%{http_code}" --max-time 10 \
    -X POST -d "username=smoketest@evil.com&password=P@ssw0rd123" \
    http://localhost:8080/login 2>/dev/null || echo "000")
if [ "$login_status" = "302" ]; then
    pass "POST /login returns 302 redirect (creds captured)"
else
    fail "POST /login returned ${login_status}, expected 302"
fi

# GET /.env → contains AWS_ACCESS_KEY_ID
env_response=$(curl -s --max-time 10 http://localhost:8080/.env 2>/dev/null || echo "")
assert_contains "GET /.env serves bait credentials" "$env_response" "AWS_ACCESS_KEY_ID"

# GET /api/config → valid JSON with "services"
config_response=$(curl -s --max-time 10 http://localhost:8080/api/config 2>/dev/null || echo "")
assert_contains "GET /api/config returns service config" "$config_response" '"services"'

# GET /api/users → JSON with "users" array
users_response=$(curl -s --max-time 10 http://localhost:8080/api/users 2>/dev/null || echo "")
assert_contains "GET /api/users returns user list" "$users_response" '"users"'

# Small pause for forensic writes to flush
sleep 2

# ══════════════════════════════════════════════════════════════
#  Step 7: Verify Forensic Capture
# ══════════════════════════════════════════════════════════════
section "Verifying Forensic Capture"

# Auth events should contain the credentials we just submitted
auth_events=$(docker exec labyrinth-http cat /var/labyrinth/forensics/auth_events.jsonl 2>/dev/null || echo "")
assert_contains "Auth events captured login credentials" "$auth_events" "smoketest@evil.com"
assert_contains "Auth events contain password" "$auth_events" "P@ssw0rd123"

# HTTP access events should log our requests
http_events=$(docker exec labyrinth-http cat /var/labyrinth/forensics/sessions/http.jsonl 2>/dev/null || echo "")
assert_contains "HTTP events logged /.env access" "$http_events" '/.env'
assert_contains "HTTP events logged /api/config access" "$http_events" '/api/config'

# ══════════════════════════════════════════════════════════════
#  Step 8: Verify Dashboard API Reflects Activity
# ══════════════════════════════════════════════════════════════
section "Verifying Dashboard API Reflects Activity"

stats_after=$(curl -s --max-time 10 http://localhost:9000/api/stats 2>/dev/null || echo "")
# Extract total_events value (simple grep — no jq dependency)
total_events=$(echo "$stats_after" | grep -o '"total_events":[[:space:]]*[0-9]*' | grep -o '[0-9]*$' || echo "0")
if [ "${total_events:-0}" -gt 0 ]; then
    pass "Dashboard reports ${total_events} total events"
else
    fail "Dashboard reports 0 total events (expected > 0)"
fi

# ══════════════════════════════════════════════════════════════
#  Step 9: Verify Environment Registry
# ══════════════════════════════════════════════════════════════
section "Verifying Environment Registry"

registry_file="${ENV_DIR}/${ENV_NAME}.json"
if [ -f "$registry_file" ]; then
    pass "Registry file exists: ${registry_file}"
    registry_content=$(cat "$registry_file")
    assert_contains "Registry has type=test" "$registry_content" '"test"'
    assert_contains "Registry has name=smoke-test" "$registry_content" '"smoke-test"'
else
    fail "Registry file not found: ${registry_file}"
fi

# ══════════════════════════════════════════════════════════════
#  Step 10: Teardown
# ══════════════════════════════════════════════════════════════
section "Tearing Down: ${ENV_NAME}"

cd "$REPO_DIR"
if "$CLI_BIN" teardown "$ENV_NAME" 2>&1; then
    DEPLOYED=false
    pass "Teardown command succeeded"
else
    fail "Teardown command failed"
    DEPLOYED=false
fi

# Give Docker a moment to stop containers
sleep 3

# ══════════════════════════════════════════════════════════════
#  Step 11: Verify Cleanup
# ══════════════════════════════════════════════════════════════
section "Verifying Cleanup"

# No labyrinth containers should be running
remaining=$(docker ps --filter "label=project=labyrinth" --format '{{.Names}}' 2>/dev/null || echo "")
if [ -z "$remaining" ]; then
    pass "No labyrinth containers remaining"
else
    fail "Containers still running: ${remaining}"
fi

# Registry file should be gone
if [ ! -f "$registry_file" ]; then
    pass "Registry file removed"
else
    fail "Registry file still exists: ${registry_file}"
fi

# ══════════════════════════════════════════════════════════════
#  Step 12: Report
# ══════════════════════════════════════════════════════════════
section "Smoke Test Results"

TOTAL=$((PASS + FAIL))
for t in "${TESTS[@]}"; do
    echo -e "    $t"
done
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "  ${GREEN}${BOLD}ALL ${TOTAL} TESTS PASSED${NC}"
    echo ""
    exit 0
else
    echo -e "  ${RED}${BOLD}${FAIL}/${TOTAL} TESTS FAILED${NC}"
    echo ""
    exit 1
fi
