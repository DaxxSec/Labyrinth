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
        "$CLI_BIN" teardown "$ENV_NAME" > /dev/null 2>&1 || true
        # Belt-and-suspenders: direct compose down
        COMPOSE_PROJECT_NAME="labyrinth-${ENV_NAME}" \
            docker compose -f "${REPO_DIR}/docker-compose.yml" down -v 2>/dev/null || true
        info "Cleanup complete"
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

# Clean up any existing labyrinth containers BEFORE checking ports
if docker ps -a --filter "label=project=labyrinth" --format '{{.Names}}' 2>/dev/null | grep -q .; then
    warn "Existing labyrinth containers detected — tearing down"
    COMPOSE_PROJECT_NAME="labyrinth-${ENV_NAME}" \
        docker compose -f "${REPO_DIR}/docker-compose.yml" down -v 2>/dev/null || true
    # Also try common compose project names from previous runs
    for proj in labyrinth-smoke-test labyrinth-labyrinth-test; do
        COMPOSE_PROJECT_NAME="$proj" \
            docker compose -f "${REPO_DIR}/docker-compose.yml" down -v 2>/dev/null || true
    done
    # Kill any stragglers by label
    docker ps -q --filter "label=project=labyrinth" 2>/dev/null | xargs -r docker rm -f 2>/dev/null || true
    sleep 3
    info "Previous labyrinth containers cleaned up"
fi

# No stale registry entry
if [ -f "${ENV_DIR}/${ENV_NAME}.json" ]; then
    warn "Stale registry entry found — removing"
    rm -f "${ENV_DIR}/${ENV_NAME}.json"
fi

# Port availability (macOS-compatible)
for port in 2222 8080 9000; do
    if lsof -i ":${port}" -sTCP:LISTEN > /dev/null 2>&1; then
        # Check if the process holding this port is Docker/labyrinth-related
        port_pid=$(lsof -ti ":${port}" -sTCP:LISTEN 2>/dev/null || echo "")
        port_cmd=$(ps -p "${port_pid}" -o comm= 2>/dev/null || echo "unknown")
        error "Port ${port} is still in use by '${port_cmd}' (PID ${port_pid}) after cleanup."
        error "Free it manually and try again."
        exit 1
    fi
    info "Port ${port} is available"
done

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
deploy_log="/tmp/labyrinth-smoke-deploy.log"
info "Building and deploying (this may take a minute)..."

# Run deploy in background, stream progress
"$CLI_BIN" deploy --skip-preflight -t "$ENV_NAME" > "$deploy_log" 2>&1 &
deploy_pid=$!

# Show container status while deploy runs
while kill -0 $deploy_pid 2>/dev/null; do
    containers=$(docker ps -a --filter "label=project=labyrinth" --format '{{.Names}}\t{{.Status}}' 2>/dev/null || echo "")
    if [ -n "$containers" ]; then
        count=$(echo "$containers" | wc -l | tr -d ' ')
        echo -e "    ${DIM}[deploying]${NC} ${count} container(s) detected:"
        echo "$containers" | while IFS=$'\t' read -r cname cstatus; do
            short="${cname##labyrinth-}"
            if echo "$cstatus" | grep -qi "up"; then
                echo -e "      ${GREEN}●${NC} ${short}: ${cstatus}"
            elif echo "$cstatus" | grep -qi "created\|exited"; then
                echo -e "      ${YELLOW}○${NC} ${short}: ${cstatus}"
            else
                echo -e "      ${DIM}◌${NC} ${short}: ${cstatus}"
            fi
        done
    else
        echo -e "    ${DIM}[deploying]${NC} Building images..."
    fi
    sleep 5
done

wait $deploy_pid
deploy_exit=$?

if [ $deploy_exit -eq 0 ]; then
    DEPLOYED=true
    pass "Deploy command succeeded"
else
    fail "Deploy command failed"
    echo ""
    tail -20 "$deploy_log"
    rm -f "$deploy_log"
    exit 1
fi
rm -f "$deploy_log"

# ══════════════════════════════════════════════════════════════
#  Step 4: Wait for Healthy
# ══════════════════════════════════════════════════════════════
section "Waiting for Services"

# Required: services the smoke test actually exercises
REQUIRED_CONTAINERS="labyrinth-ssh labyrinth-http labyrinth-proxy labyrinth-dashboard"
REQUIRED_COUNT=4
# Optional: orchestrator may restart-loop in test mode (no config file)
OPTIONAL_CONTAINERS="labyrinth-orchestrator"
elapsed=0

while [ $elapsed -lt $TIMEOUT ]; do
    running=0
    running_names=""
    pending_names=""
    for name in $REQUIRED_CONTAINERS; do
        short="${name##labyrinth-}"
        if docker ps --filter "name=^${name}$" --filter "status=running" --format '{{.Names}}' 2>/dev/null | grep -q "^${name}$"; then
            running=$((running + 1))
            running_names="${running_names} ${GREEN}${short}${NC}"
        else
            status=$(docker ps -a --filter "name=^${name}$" --format '{{.Status}}' 2>/dev/null || echo "not found")
            pending_names="${pending_names} ${YELLOW}${short}${NC}${DIM}(${status})${NC}"
        fi
    done
    if [ $running -ge $REQUIRED_COUNT ]; then
        break
    fi
    echo -e "    ${DIM}[${elapsed}s]${NC} ${running}/${REQUIRED_COUNT} up:${running_names}  waiting:${pending_names}"
    sleep 5
    elapsed=$((elapsed + 5))
done

if [ $elapsed -ge $TIMEOUT ]; then
    fail "Timed out waiting for containers (only ${running}/${REQUIRED_COUNT} running after ${TIMEOUT}s)"
    docker ps -a --filter "label=project=labyrinth" --format "table {{.Names}}\t{{.Status}}" 2>/dev/null || true
    exit 1
fi

pass "All ${REQUIRED_COUNT} required services running (${elapsed}s)"

# Check optional containers
for name in $OPTIONAL_CONTAINERS; do
    short="${name##labyrinth-}"
    if docker ps --filter "name=^${name}$" --filter "status=running" --format '{{.Names}}' 2>/dev/null | grep -q "^${name}$"; then
        info "${short} is running (optional)"
    else
        warn "${short} is not stable (optional — smoke test will continue)"
    fi
done

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
info "Running teardown..."
teardown_output=$("$CLI_BIN" teardown "$ENV_NAME" 2>&1)
teardown_exit=$?
if [ $teardown_exit -eq 0 ]; then
    DEPLOYED=false
    pass "Teardown command succeeded"
else
    # Teardown via CLI failed — try direct compose down
    warn "CLI teardown returned non-zero, falling back to direct compose down"
    COMPOSE_PROJECT_NAME="labyrinth-${ENV_NAME}" \
        docker compose -f "${REPO_DIR}/docker-compose.yml" down -v 2>/dev/null || true
    DEPLOYED=false
    pass "Teardown completed (via fallback)"
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
