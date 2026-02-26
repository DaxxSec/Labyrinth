#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════
#  Project LABYRINTH — Attacker Agent Setup
#  Sets up an offensive AI agent in an isolated Docker container
#  connected to the LABYRINTH network for safe testing.
#
#  All agents run inside Docker — nothing touches your host.
# ═══════════════════════════════════════════════════════════════
set -euo pipefail

GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
MAGENTA='\033[0;35m'
DIM='\033[2m'
BOLD='\033[1m'
NC='\033[0m'

info()    { echo -e "  ${GREEN}[+]${NC} $1"; }
warn()    { echo -e "  ${YELLOW}[!]${NC} $1"; }
error()   { echo -e "  ${RED}[✗]${NC} $1"; }
section() { echo -e "\n  ${MAGENTA}━━━ $1 ━━━${NC}\n"; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"
ATTACKER_DIR="${REPO_DIR}/.attacker"
LABYRINTH_NETWORK="labyrinth-net"

# ── Preflight ────────────────────────────────────────────────
preflight() {
    section "Preflight"

    if ! command -v docker &> /dev/null; then
        error "Docker is required. Install Docker and try again."
        exit 1
    fi
    info "Docker found"

    if ! docker info &> /dev/null 2>&1; then
        error "Docker daemon not running. Start Docker and try again."
        exit 1
    fi
    info "Docker daemon running"

    # Check that labyrinth network exists (environment should be deployed)
    if ! docker network inspect "$LABYRINTH_NETWORK" &> /dev/null 2>&1; then
        warn "LABYRINTH network not found — deploy an environment first"
        echo ""
        echo -e "  ${DIM}labyrinth deploy -t${NC}"
        echo -e "  ${DIM}  or${NC}"
        echo -e "  ${DIM}./deploy.sh -t${NC}"
        echo ""
        read -rp "  Continue anyway? (agent will use host networking) [y/N] " answer
        case "${answer}" in
            [Yy]) LABYRINTH_NETWORK="" ;;
            *) exit 0 ;;
        esac
    else
        info "LABYRINTH network found"
    fi

    mkdir -p "$ATTACKER_DIR"
}

# ── Menu ─────────────────────────────────────────────────────
show_menu() {
    echo ""
    echo -e "  ${GREEN}██╗      █████╗ ██████╗ ██╗   ██╗██████╗ ██╗███╗   ██╗████████╗██╗  ██╗${NC}"
    echo -e "  ${GREEN}██║     ██╔══██╗██╔══██╗╚██╗ ██╔╝██╔══██╗██║████╗  ██║╚══██╔══╝██║  ██║${NC}"
    echo -e "  ${GREEN}██║     ███████║██████╔╝ ╚████╔╝ ██████╔╝██║██╔██╗ ██║   ██║   ███████║${NC}"
    echo -e "  ${GREEN}██║     ██╔══██║██╔══██╗  ╚██╔╝  ██╔══██╗██║██║╚██╗██║   ██║   ██╔══██║${NC}"
    echo -e "  ${GREEN}███████╗██║  ██║██████╔╝   ██║   ██║  ██║██║██║ ╚████║   ██║   ██║  ██║${NC}"
    echo -e "  ${GREEN}╚══════╝╚═╝  ╚═╝╚═════╝    ╚═╝   ╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝   ╚═╝   ╚═╝  ╚═╝${NC}"
    echo ""
    echo -e "  ${DIM}Attacker Agent Setup${NC}"
    echo ""

    section "Select an Attacker Agent"
    echo ""
    echo -e "  ${BOLD}1)${NC}  ${CYAN}PentAGI${NC}         Fully autonomous multi-agent system"
    echo -e "     ${DIM}Web UI · 20+ security tools · Docker sandboxed${NC}"
    echo -e "     ${DIM}Best for: hands-off autonomous pentesting${NC}"
    echo ""
    echo -e "  ${BOLD}2)${NC}  ${CYAN}PentestAgent${NC}    AI pentesting framework with TUI"
    echo -e "     ${DIM}TUI interface · Agent & Crew modes · Kali image available${NC}"
    echo -e "     ${DIM}Best for: interactive/guided pentesting with playbooks${NC}"
    echo ""
    echo -e "  ${BOLD}3)${NC}  ${CYAN}Strix${NC}           AI hacker agents"
    echo -e "     ${DIM}CLI + TUI · Kali sandbox · Web app focused${NC}"
    echo -e "     ${DIM}Best for: web application security testing${NC}"
    echo ""
    echo -e "  ${BOLD}4)${NC}  ${CYAN}Custom Agent${NC}    Bring your own tool"
    echo -e "     ${DIM}Launches a Kali container on the LABYRINTH network${NC}"
    echo -e "     ${DIM}Best for: running your own tools or manual testing${NC}"
    echo ""

    read -rp "  Select [1-4]: " choice
    echo ""
}

# ── Collect API key ──────────────────────────────────────────
collect_api_key() {
    local provider_hint="$1"
    echo -e "  ${BOLD}An LLM API key is required.${NC}"
    echo -e "  ${DIM}Supported: OpenAI, Anthropic, Google, or local (Ollama)${NC}"
    echo ""
    echo -e "  ${BOLD}Provider options:${NC}"
    echo -e "  ${DIM}  1) OpenAI     (OPENAI_API_KEY)${NC}"
    echo -e "  ${DIM}  2) Anthropic  (ANTHROPIC_API_KEY)${NC}"
    echo -e "  ${DIM}  3) Ollama     (local, no key needed)${NC}"
    echo ""
    read -rp "  Provider [1-3]: " provider_choice

    case "${provider_choice}" in
        1)
            LLM_PROVIDER="openai"
            read -rsp "  OpenAI API key: " api_key; echo ""
            LLM_ENV="-e OPENAI_API_KEY=${api_key}"
            LLM_MODEL="${LLM_MODEL:-gpt-4o}"
            ;;
        2)
            LLM_PROVIDER="anthropic"
            read -rsp "  Anthropic API key: " api_key; echo ""
            LLM_ENV="-e ANTHROPIC_API_KEY=${api_key}"
            LLM_MODEL="${LLM_MODEL:-claude-sonnet-4-20250514}"
            ;;
        3)
            LLM_PROVIDER="ollama"
            LLM_ENV=""
            LLM_MODEL="${LLM_MODEL:-llama3}"
            warn "Make sure Ollama is running on your host (ollama serve)"
            ;;
        *)
            error "Invalid choice"
            exit 1
            ;;
    esac
    echo ""
}

# ── Docker network flags ─────────────────────────────────────
net_flags() {
    if [ -n "$LABYRINTH_NETWORK" ]; then
        echo "--network ${LABYRINTH_NETWORK}"
    else
        echo "--network host"
    fi
}

# Resolve target hostname depending on network mode
target_host() {
    if [ -n "$LABYRINTH_NETWORK" ]; then
        # On the labyrinth bridge network, use container names
        echo "labyrinth-ssh"
    else
        echo "localhost"
    fi
}

target_http_host() {
    if [ -n "$LABYRINTH_NETWORK" ]; then
        echo "labyrinth-http"
    else
        echo "localhost"
    fi
}

# ── 1) PentAGI ───────────────────────────────────────────────
setup_pentagi() {
    section "Setting up PentAGI"

    collect_api_key "PentAGI"

    local pentagi_dir="${ATTACKER_DIR}/pentagi"
    mkdir -p "$pentagi_dir"

    info "Downloading PentAGI configuration..."
    curl -fsSL -o "${pentagi_dir}/docker-compose.yml" \
        "https://raw.githubusercontent.com/vxcontrol/pentagi/master/docker-compose.yml"
    curl -fsSL -o "${pentagi_dir}/.env" \
        "https://raw.githubusercontent.com/vxcontrol/pentagi/master/.env.example"

    # Patch .env with user's API key
    case "$LLM_PROVIDER" in
        openai)
            sed -i.bak "s|^OPEN_AI_KEY=.*|OPEN_AI_KEY=${api_key}|" "${pentagi_dir}/.env"
            ;;
        anthropic)
            sed -i.bak "s|^ANTHROPIC_API_KEY=.*|ANTHROPIC_API_KEY=${api_key}|" "${pentagi_dir}/.env"
            ;;
    esac

    # Strip inline comments that break docker-compose
    perl -i -pe 's/\s+#.*$//' "${pentagi_dir}/.env" 2>/dev/null || true
    rm -f "${pentagi_dir}/.env.bak"

    info "Starting PentAGI..."
    cd "$pentagi_dir"
    docker compose up -d 2>&1

    section "PentAGI is Ready"
    echo -e "  ${BOLD}Web UI:${NC}  ${CYAN}https://localhost:8443${NC}"
    echo -e "  ${BOLD}Login:${NC}   admin@pentagi.com / admin"
    echo ""
    echo -e "  ${BOLD}To test against LABYRINTH, create a Flow and enter:${NC}"
    echo ""
    echo -e "  ${DIM}  SSH target:${NC}"
    echo -e "  ${DIM}  Penetration test the SSH service at $(target_host):2222${NC}"
    echo ""
    echo -e "  ${DIM}  HTTP target:${NC}"
    echo -e "  ${DIM}  Penetration test the web app at http://$(target_http_host):8080${NC}"
    echo ""
    echo -e "  ${DIM}Teardown: cd ${pentagi_dir} && docker compose down -v${NC}"
    echo ""
}

# ── 2) PentestAgent ──────────────────────────────────────────
setup_pentestagent() {
    section "Setting up PentestAgent"

    collect_api_key "PentestAgent"

    local ssh_target="$(target_host)"
    local http_target="$(target_http_host)"

    local docker_image="ghcr.io/gh05tcrew/pentestagent:kali"
    info "Pulling PentestAgent Kali image..."
    docker pull "$docker_image" 2>&1

    section "PentestAgent is Ready"
    echo -e "  ${BOLD}Launching interactive container with Kali tools...${NC}"
    echo ""
    echo -e "  ${DIM}Inside the TUI, try:${NC}"
    echo -e "  ${DIM}  /agent Pentest SSH at ${ssh_target}:2222${NC}"
    echo -e "  ${DIM}  /agent Pentest web app at http://${http_target}:8080${NC}"
    echo -e "  ${DIM}  /crew Full pentest of ${ssh_target}:2222 and http://${http_target}:8080${NC}"
    echo ""
    echo -e "  ${DIM}Press Ctrl+D or /quit to exit${NC}"
    echo ""

    # Build env flags
    local env_flags="${LLM_ENV}"
    case "$LLM_PROVIDER" in
        openai)    env_flags="${env_flags} -e PENTESTAGENT_MODEL=gpt-4o" ;;
        anthropic) env_flags="${env_flags} -e PENTESTAGENT_MODEL=claude-sonnet-4-20250514" ;;
        ollama)    env_flags="${env_flags} -e PENTESTAGENT_MODEL=ollama/${LLM_MODEL} -e OLLAMA_BASE_URL=http://host.docker.internal:11434" ;;
    esac

    # Run interactively — container removed on exit
    docker run -it --rm \
        --name labyrinth-attacker-pentestagent \
        $(net_flags) \
        ${env_flags} \
        "$docker_image"
}

# ── 3) Strix ─────────────────────────────────────────────────
setup_strix() {
    section "Setting up Strix"

    collect_api_key "Strix"

    local http_target="$(target_http_host)"

    # Strix needs its sandbox image
    info "Pulling Strix sandbox image..."
    docker pull ghcr.io/usestrix/strix-sandbox:latest 2>&1 || \
        docker pull ghcr.io/usestrix/strix-sandbox:0.1.12 2>&1

    # Strix runs as a host binary that launches Docker containers
    # We'll run it inside a container that has Docker socket access
    # but isolated on the labyrinth network

    local strix_dir="${ATTACKER_DIR}/strix"
    mkdir -p "$strix_dir"

    # Build env string for strix
    local env_flags=""
    case "$LLM_PROVIDER" in
        openai)    env_flags="-e STRIX_LLM=openai/gpt-4o -e LLM_API_KEY=${api_key}" ;;
        anthropic) env_flags="-e STRIX_LLM=anthropic/claude-sonnet-4-20250514 -e LLM_API_KEY=${api_key}" ;;
        ollama)    env_flags="-e STRIX_LLM=ollama/${LLM_MODEL} -e LLM_API_KEY=none -e LLM_API_BASE=http://host.docker.internal:11434" ;;
    esac

    section "Strix is Ready"
    echo -e "  ${BOLD}Strix runs as a host CLI that launches Docker sandboxes.${NC}"
    echo ""
    echo -e "  ${BOLD}Install Strix:${NC}"
    echo -e "  ${DIM}curl -sSL https://strix.ai/install | bash${NC}"
    echo ""
    echo -e "  ${BOLD}Then run against LABYRINTH:${NC}"

    case "$LLM_PROVIDER" in
        openai)    echo -e "  ${DIM}export STRIX_LLM=openai/gpt-4o${NC}" ;;
        anthropic) echo -e "  ${DIM}export STRIX_LLM=anthropic/claude-sonnet-4-20250514${NC}" ;;
        ollama)    echo -e "  ${DIM}export STRIX_LLM=ollama/${LLM_MODEL}${NC}" ;;
    esac
    echo -e "  ${DIM}export LLM_API_KEY=<your-key>${NC}"
    echo ""
    echo -e "  ${DIM}strix --target http://localhost:8080${NC}"
    echo -e "  ${DIM}strix --target localhost --instruction \"Pentest SSH on port 2222\"${NC}"
    echo ""
    echo -e "  ${YELLOW}Note: Strix launches its own Docker sandbox containers.${NC}"
    echo -e "  ${YELLOW}Results saved to ./strix_runs/ in your working directory.${NC}"
    echo ""
}

# ── 4) Custom Agent ──────────────────────────────────────────
setup_custom() {
    section "Custom Agent — Kali Container"

    local ssh_target="$(target_host)"
    local http_target="$(target_http_host)"

    info "Launching Kali Linux container on the LABYRINTH network..."
    echo ""
    echo -e "  ${BOLD}You'll get a root shell with common pentest tools.${NC}"
    echo -e "  ${BOLD}The container is connected to the LABYRINTH network.${NC}"
    echo ""
    echo -e "  ${DIM}Targets from inside the container:${NC}"
    echo -e "  ${DIM}  SSH:   ${ssh_target}:22   (mapped from host :2222)${NC}"
    echo -e "  ${DIM}  HTTP:  ${http_target}:80   (mapped from host :8080)${NC}"
    echo -e "  ${DIM}  Dash:  labyrinth-dashboard:9000${NC}"
    echo ""
    echo -e "  ${DIM}Example commands:${NC}"
    echo -e "  ${DIM}  nmap -sV ${ssh_target}${NC}"
    echo -e "  ${DIM}  ssh root@${ssh_target}${NC}"
    echo -e "  ${DIM}  curl http://${http_target}${NC}"
    echo -e "  ${DIM}  hydra -l root -P /usr/share/wordlists/rockyou.txt ssh://${ssh_target}${NC}"
    echo ""
    echo -e "  ${DIM}Press Ctrl+D or type 'exit' to leave${NC}"
    echo ""

    docker run -it --rm \
        --name labyrinth-attacker-custom \
        $(net_flags) \
        --hostname attacker \
        kalilinux/kali-rolling \
        /bin/bash -c '
            echo "[*] Updating package list..."
            apt-get update -qq 2>/dev/null
            echo "[*] Installing core tools..."
            DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
                nmap hydra curl wget netcat-openbsd sqlmap nikto dirb sshpass \
                2>/dev/null
            echo ""
            echo "[+] Tools ready. You are on the LABYRINTH network."
            echo ""
            exec /bin/bash
        '
}

# ── Teardown helper ──────────────────────────────────────────
teardown_attackers() {
    section "Tearing Down Attacker Agents"

    # Stop any running attacker containers
    for name in labyrinth-attacker-pentestagent labyrinth-attacker-custom; do
        if docker ps -q --filter "name=${name}" | grep -q .; then
            info "Stopping ${name}..."
            docker rm -f "$name" 2>/dev/null || true
        fi
    done

    # PentAGI has its own compose
    if [ -f "${ATTACKER_DIR}/pentagi/docker-compose.yml" ]; then
        info "Stopping PentAGI..."
        cd "${ATTACKER_DIR}/pentagi" && docker compose down -v 2>/dev/null || true
    fi

    info "All attacker agents stopped"
}

# ── Main ─────────────────────────────────────────────────────
if [ "${1:-}" = "--teardown" ]; then
    preflight
    teardown_attackers
    exit 0
fi

preflight
show_menu

case "$choice" in
    1) setup_pentagi ;;
    2) setup_pentestagent ;;
    3) setup_strix ;;
    4) setup_custom ;;
    *)
        error "Invalid choice: $choice"
        exit 1
        ;;
esac
