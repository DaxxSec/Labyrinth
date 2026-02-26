#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════
#  Project LABYRINTH — Deployment Script
#  Authors: Stephen Stewart & Claude (Anthropic)
# ═══════════════════════════════════════════════════════════════
set -euo pipefail

# ── Colors ────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
MAGENTA='\033[0;35m'
DIM='\033[2m'
BOLD='\033[1m'
NC='\033[0m'

# ── Banner ────────────────────────────────────────────────────
banner() {
    echo -e "${CYAN}"
    echo "  ╔═══════════════════════════════════════════════════╗"
    echo "  ║                                                   ║"
    echo -e "  ║   ${GREEN}██╗      █████╗ ██████╗ ██╗   ██╗${CYAN}              ║"
    echo -e "  ║   ${GREEN}██║     ██╔══██╗██╔══██╗╚██╗ ██╔╝${CYAN}              ║"
    echo -e "  ║   ${GREEN}██║     ███████║██████╔╝ ╚████╔╝${CYAN}               ║"
    echo -e "  ║   ${GREEN}██║     ██╔══██║██╔══██╗  ╚██╔╝${CYAN}                ║"
    echo -e "  ║   ${GREEN}███████╗██║  ██║██████╔╝   ██║${CYAN}                 ║"
    echo -e "  ║   ${GREEN}╚══════╝╚═╝  ╚═╝╚═════╝    ╚═╝${CYAN}                ║"
    echo "  ║                                                   ║"
    echo -e "  ║   ${DIM}Adversarial Cognitive Honeypot Architecture${NC}${CYAN}   ║"
    echo "  ║                                                   ║"
    echo "  ╚═══════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# ── Logging ───────────────────────────────────────────────────
info()    { echo -e "  ${GREEN}[+]${NC} $1"; }
warn()    { echo -e "  ${YELLOW}[!]${NC} $1"; }
error()   { echo -e "  ${RED}[✗]${NC} $1"; }
section() { echo -e "\n  ${MAGENTA}━━━ $1 ━━━${NC}\n"; }

# ── Usage ─────────────────────────────────────────────────────
usage() {
    echo -e "${BOLD}Usage:${NC} ./deploy.sh [OPTIONS]"
    echo ""
    echo -e "  ${GREEN}--test-mode${NC}     Deploy test environment (no VLAN required)"
    echo -e "  ${GREEN}--production${NC}    Deploy production environment (requires VLAN)"
    echo -e "  ${GREEN}--teardown${NC}      Stop and remove all LABYRINTH containers"
    echo -e "  ${GREEN}--status${NC}        Show running LABYRINTH services"
    echo -e "  ${GREEN}--help${NC}          Show this message"
    echo ""
}

# ── Preflight checks ─────────────────────────────────────────
preflight() {
    section "Preflight Checks"

    # Docker
    if command -v docker &> /dev/null; then
        local docker_version
        docker_version=$(docker --version | grep -oP '\d+\.\d+\.\d+' | head -1)
        info "Docker found: v${docker_version}"
    else
        error "Docker not found. Install Docker 20.10+ and try again."
        exit 1
    fi

    # Docker daemon
    if docker info &> /dev/null 2>&1; then
        info "Docker daemon is running"
    else
        error "Docker daemon not running. Start Docker and try again."
        exit 1
    fi

    # Docker Compose
    if docker compose version &> /dev/null 2>&1; then
        info "Docker Compose found"
    elif command -v docker-compose &> /dev/null; then
        info "Docker Compose (standalone) found"
        COMPOSE_CMD="docker-compose"
    else
        error "Docker Compose not found."
        exit 1
    fi

    # Python
    if command -v python3 &> /dev/null; then
        local py_version
        py_version=$(python3 --version | grep -oP '\d+\.\d+')
        info "Python found: v${py_version}"
    else
        warn "Python3 not found. Dashboard features may be limited."
    fi

    # Port availability
    for port in 2222 8080 9000; do
        if ! ss -tlnp 2>/dev/null | grep -q ":${port} " && \
           ! netstat -tlnp 2>/dev/null | grep -q ":${port} "; then
            info "Port ${port} is available"
        else
            error "Port ${port} is already in use. Free it and try again."
            exit 1
        fi
    done

    info "All preflight checks passed"
}

# ── Test Mode Deploy ──────────────────────────────────────────
deploy_test() {
    section "Deploying Test Environment"

    info "Building honeypot container image..."
    docker compose -f docker-compose.yml build 2>&1 | while read -r line; do
        echo -e "    ${DIM}${line}${NC}"
    done

    info "Starting LABYRINTH stack..."
    docker compose -f docker-compose.yml up -d 2>&1

    # Wait for services
    info "Waiting for services to initialize..."
    sleep 3

    section "LABYRINTH is Live"

    echo -e "  ${GREEN}┌─────────────────────────────────────────────────┐${NC}"
    echo -e "  ${GREEN}│${NC}  SSH Honeypot:     ${BOLD}localhost:2222${NC}               ${GREEN}│${NC}"
    echo -e "  ${GREEN}│${NC}  HTTP Honeypot:    ${BOLD}localhost:8080${NC}               ${GREEN}│${NC}"
    echo -e "  ${GREEN}│${NC}  Dashboard:        ${BOLD}http://localhost:9000${NC}         ${GREEN}│${NC}"
    echo -e "  ${GREEN}└─────────────────────────────────────────────────┘${NC}"
    echo ""
    echo -e "  Point your offensive AI agent at the honeypot."
    echo -e "  Watch captures in real time at the dashboard."
    echo ""
    echo -e "  ${DIM}Teardown:  ./deploy.sh --teardown${NC}"
    echo -e "  ${DIM}Status:    ./deploy.sh --status${NC}"
    echo ""
}

# ── Teardown ──────────────────────────────────────────────────
teardown() {
    section "Tearing Down LABYRINTH"

    info "Stopping containers..."
    docker compose -f docker-compose.yml down -v 2>&1 || true

    info "Removing LABYRINTH images..."
    docker images --filter "label=project=labyrinth" -q | xargs -r docker rmi 2>/dev/null || true

    info "Cleanup complete"
}

# ── Status ────────────────────────────────────────────────────
status() {
    section "LABYRINTH Status"

    local running
    running=$(docker compose -f docker-compose.yml ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo "")

    if [ -z "$running" ]; then
        warn "No LABYRINTH services are running"
    else
        echo "$running"
    fi
}

# ── Main ──────────────────────────────────────────────────────
COMPOSE_CMD="docker compose"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

banner

case "${1:-}" in
    --test-mode)
        preflight
        deploy_test
        ;;
    --production)
        error "Production mode not yet implemented."
        error "Use --test-mode for testing."
        exit 1
        ;;
    --teardown)
        teardown
        ;;
    --status)
        status
        ;;
    --help|-h|"")
        usage
        ;;
    *)
        error "Unknown option: $1"
        usage
        exit 1
        ;;
esac
