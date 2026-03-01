#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════
#  Project LABYRINTH — Deployment Script
#  Authors: DaxxSec & Claude (Anthropic)
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
    echo ""
    echo -e "  ${GREEN}██╗      █████╗ ██████╗ ██╗   ██╗██████╗ ██╗███╗   ██╗████████╗██╗  ██╗${NC}"
    echo -e "  ${GREEN}██║     ██╔══██╗██╔══██╗╚██╗ ██╔╝██╔══██╗██║████╗  ██║╚══██╔══╝██║  ██║${NC}"
    echo -e "  ${GREEN}██║     ███████║██████╔╝ ╚████╔╝ ██████╔╝██║██╔██╗ ██║   ██║   ███████║${NC}"
    echo -e "  ${GREEN}██║     ██╔══██║██╔══██╗  ╚██╔╝  ██╔══██╗██║██║╚██╗██║   ██║   ██╔══██║${NC}"
    echo -e "  ${GREEN}███████╗██║  ██║██████╔╝   ██║   ██║  ██║██║██║ ╚████║   ██║   ██║  ██║${NC}"
    echo -e "  ${GREEN}╚══════╝╚═╝  ╚═╝╚═════╝    ╚═╝   ╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝   ╚═╝   ╚═╝  ╚═╝${NC}"
    echo ""
    echo -e "  ${DIM}Adversarial Cognitive Portal Trap Architecture${NC}"
    echo ""
}

# ── Logging ───────────────────────────────────────────────────
info()    { echo -e "  ${GREEN}[+]${NC} $1"; }
warn()    { echo -e "  ${YELLOW}[!]${NC} $1"; }
error()   { echo -e "  ${RED}[✗]${NC} $1"; }
section() { echo -e "\n  ${MAGENTA}━━━ $1 ━━━${NC}\n"; }

# ── Constants ─────────────────────────────────────────────────
ENV_DIR="${HOME}/.labyrinth/environments"
COMPOSE_CMD="docker compose"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# ── Usage ─────────────────────────────────────────────────────
usage() {
    echo -e "${BOLD}Usage:${NC} ./deploy.sh [COMMAND] [OPTIONS]"
    echo ""
    echo -e "  ${BOLD}Test Environments:${NC}"
    echo -e "    ${GREEN}-t [name]${NC}              Deploy test environment (default: labyrinth-test)"
    echo ""
    echo -e "  ${BOLD}Production Environments:${NC}"
    echo -e "    ${GREEN}-p [name] --docker${NC}     Deploy production with Docker Compose"
    echo -e "    ${GREEN}-p [name] --k8s${NC}        Deploy production with Kubernetes"
    echo -e "    ${GREEN}-p [name] --edge${NC}       Deploy production at the edge"
    echo -e "    ${GREEN}-p${NC}                     List available production types"
    echo ""
    echo -e "  ${BOLD}Management:${NC}"
    echo -e "    ${GREEN}--status [name]${NC}        Show all environments or a specific one"
    echo -e "    ${GREEN}--teardown <name>${NC}      Tear down a specific environment"
    echo -e "    ${GREEN}--teardown --all${NC}       Tear down all environments"
    echo -e "    ${GREEN}--list${NC}                 List all tracked environments"
    echo -e "    ${GREEN}--help${NC}                 Show this message"
    echo ""
    echo -e "  ${BOLD}Examples:${NC}"
    echo -e "    ${DIM}./deploy.sh -t${NC}                        # Quick test deploy"
    echo -e "    ${DIM}./deploy.sh -t mylab${NC}                  # Named test environment"
    echo -e "    ${DIM}./deploy.sh -p staging --docker${NC}       # Production Docker deploy"
    echo -e "    ${DIM}./deploy.sh --status mylab${NC}            # Check specific env"
    echo -e "    ${DIM}./deploy.sh --teardown mylab${NC}          # Tear down specific env"
    echo -e "    ${DIM}./deploy.sh --list${NC}                    # See all environments"
    echo ""
}

# ── Environment Registry ─────────────────────────────────────
ensure_env_dir() {
    mkdir -p "$ENV_DIR"
}

register_env() {
    local name="$1" type="$2" mode="$3"
    ensure_env_dir
    local created
    created="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    local json
    case "$mode" in
        docker-compose|docker)
            json=$(printf '{"name":"%s","type":"%s","mode":"%s","created":"%s","compose_project":"labyrinth-%s"}' \
                "$name" "$type" "$mode" "$created" "$name")
            ;;
        k8s)
            json=$(printf '{"name":"%s","type":"%s","mode":"%s","created":"%s","namespace":"labyrinth-%s"}' \
                "$name" "$type" "$mode" "$created" "$name")
            ;;
        edge)
            json=$(printf '{"name":"%s","type":"%s","mode":"%s","created":"%s"}' \
                "$name" "$type" "$mode" "$created")
            ;;
    esac
    echo "$json" > "${ENV_DIR}/${name}.json"
    info "Registered environment: ${name} (${type}/${mode})"
}

load_env() {
    local name="$1"
    local file="${ENV_DIR}/${name}.json"
    if [ ! -f "$file" ]; then
        error "Environment '${name}' not found in registry" >&2
        return 1
    fi
    cat "$file"
}

remove_env() {
    local name="$1"
    local file="${ENV_DIR}/${name}.json"
    if [ -f "$file" ]; then
        rm "$file"
        info "Removed environment '${name}' from registry"
    fi
}

# Extract a value from simple flat JSON (no jq dependency)
json_val() {
    local json="$1" key="$2"
    echo "$json" | sed -n 's/.*"'"$key"'":"\([^"]*\)".*/\1/p'
}

list_envs() {
    ensure_env_dir
    local files
    files=("${ENV_DIR}"/*.json)
    # Check if any env files exist (glob returns literal if no match)
    if [ ! -e "${files[0]:-}" ]; then
        warn "No environments registered"
        echo ""
        echo -e "  ${DIM}Deploy one with: ./deploy.sh -t [name]${NC}"
        return 0
    fi

    section "Registered Environments"
    printf "  ${BOLD}%-20s %-14s %-18s %s${NC}\n" "NAME" "TYPE" "MODE" "CREATED"
    printf "  %-20s %-14s %-18s %s\n" "────────────────────" "──────────────" "──────────────────" "───────────────────"
    for f in "${files[@]}"; do
        [ -f "$f" ] || continue
        local json
        json=$(cat "$f")
        local name type mode created
        name=$(json_val "$json" "name")
        type=$(json_val "$json" "type")
        mode=$(json_val "$json" "mode")
        created=$(json_val "$json" "created")
        printf "  %-20s %-14s %-18s %s\n" "$name" "$type" "$mode" "$created"
    done
    echo ""
}

# ── Preflight Checks ─────────────────────────────────────────
preflight() {
    section "Preflight Checks"

    # Docker
    if command -v docker &> /dev/null; then
        local docker_version
        docker_version=$(docker --version | sed -n 's/.*version \([0-9][0-9.]*\).*/\1/p')
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
        py_version=$(python3 --version | sed -n 's/.*Python \([0-9]*\.[0-9]*\).*/\1/p')
        info "Python found: v${py_version}"
    else
        warn "Python3 not found. Dashboard features may be limited."
    fi

    # Port availability
    for port in 22 8080 9000; do
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
    local env_name="${1:-labyrinth-test}"
    local compose_project="labyrinth-${env_name}"

    section "Deploying Test Environment: ${env_name}"

    info "Building honeypot container image..."
    COMPOSE_PROJECT_NAME="$compose_project" \
        $COMPOSE_CMD -f docker-compose.yml build 2>&1 | while read -r line; do
        echo -e "    ${DIM}${line}${NC}"
    done

    info "Starting LABYRINTH stack..."
    COMPOSE_PROJECT_NAME="$compose_project" \
        $COMPOSE_CMD -f docker-compose.yml up -d 2>&1

    # Wait for services
    info "Waiting for services to initialize..."
    sleep 3

    # Register the environment
    register_env "$env_name" "test" "docker-compose"

    section "LABYRINTH is Live  [${env_name}]"

    echo -e "  ${GREEN}┌─────────────────────────────────────────────────┐${NC}"
    echo -e "  ${GREEN}│${NC}  Environment:      ${BOLD}${env_name}${NC}$(printf '%*s' $((21 - ${#env_name})) '')${GREEN}│${NC}"
    echo -e "  ${GREEN}│${NC}  SSH Portal Trap:  ${BOLD}localhost:22${NC}               ${GREEN}│${NC}"
    echo -e "  ${GREEN}│${NC}  HTTP Portal Trap: ${BOLD}localhost:8080${NC}               ${GREEN}│${NC}"
    echo -e "  ${GREEN}│${NC}  Dashboard:        ${BOLD}http://localhost:9000${NC}         ${GREEN}│${NC}"
    echo -e "  ${GREEN}└─────────────────────────────────────────────────┘${NC}"
    echo ""
    echo -e "  Point your offensive AI agent at the portal trap."
    echo -e "  Watch captures in real time at the dashboard."
    echo ""
    echo -e "  ${DIM}Teardown:  ./deploy.sh --teardown ${env_name}${NC}"
    echo -e "  ${DIM}Status:    ./deploy.sh --status ${env_name}${NC}"
    echo -e "  ${DIM}All envs:  ./deploy.sh --list${NC}"
    echo ""
}

# ── Production Docker Deploy (scaffold) ──────────────────────
deploy_prod_docker() {
    local env_name="${1:-}"
    if [ -z "$env_name" ]; then
        error "Production deploy requires a name: ./deploy.sh -p <name> --docker"
        exit 1
    fi

    section "Production Deploy (Docker): ${env_name}"

    preflight

    # TODO: Implement full production Docker deployment
    # This will use docker-compose.prod.yml with:
    #   - TLS termination
    #   - Real credential management
    #   - Production logging drivers
    #   - Resource limits
    #   - Health checks with restart policies
    #   - VLAN/network isolation

    register_env "$env_name" "production" "docker"

    warn "Production (Docker) deployment is scaffolded but not yet fully implemented."
    info "Environment '${env_name}' registered. Full deployment coming in Option A."
    echo ""
    echo -e "  ${DIM}What this will include:${NC}"
    echo -e "  ${DIM}  - docker-compose.prod.yml with hardened settings${NC}"
    echo -e "  ${DIM}  - TLS termination and real credential management${NC}"
    echo -e "  ${DIM}  - Production logging and monitoring${NC}"
    echo -e "  ${DIM}  - Resource limits and health checks${NC}"
    echo ""
}

# ── Production K8s Deploy (stub) ─────────────────────────────
deploy_prod_k8s() {
    local env_name="${1:-}"
    if [ -z "$env_name" ]; then
        error "Production deploy requires a name: ./deploy.sh -p <name> --k8s"
        exit 1
    fi

    section "Production Deploy (Kubernetes): ${env_name}"

    warn "Kubernetes deployment is not yet implemented."
    echo ""
    echo -e "  ${DIM}What this will include (Option B):${NC}"
    echo -e "  ${DIM}  - Helm chart for LABYRINTH stack${NC}"
    echo -e "  ${DIM}  - Namespace isolation per environment${NC}"
    echo -e "  ${DIM}  - Horizontal pod autoscaling${NC}"
    echo -e "  ${DIM}  - Ingress with TLS and rate limiting${NC}"
    echo -e "  ${DIM}  - Persistent volume claims for capture data${NC}"
    echo ""
}

# ── Production Edge Deploy (stub) ────────────────────────────
deploy_prod_edge() {
    local env_name="${1:-}"
    if [ -z "$env_name" ]; then
        error "Production deploy requires a name: ./deploy.sh -p <name> --edge"
        exit 1
    fi

    section "Production Deploy (Edge): ${env_name}"

    warn "Edge deployment is not yet implemented."
    echo ""
    echo -e "  ${DIM}What this will include (Option C):${NC}"
    echo -e "  ${DIM}  - Terraform / Fly.io deployment config${NC}"
    echo -e "  ${DIM}  - Globally distributed honeypot nodes${NC}"
    echo -e "  ${DIM}  - Centralized log aggregation${NC}"
    echo -e "  ${DIM}  - Edge-optimized container images${NC}"
    echo -e "  ${DIM}  - Anycast routing for realistic exposure${NC}"
    echo ""
}

# ── Show Production Types ────────────────────────────────────
show_prod_types() {
    section "Available Production Architectures"

    echo -e "  ${BOLD}--docker${NC}    Container-native production deployment"
    echo -e "             Docker Compose with production hardening, TLS, and monitoring."
    echo -e "             ${DIM}./deploy.sh -p <name> --docker${NC}"
    echo ""
    echo -e "  ${BOLD}--k8s${NC}       Kubernetes deployment ${YELLOW}(not yet implemented)${NC}"
    echo -e "             Helm-based deployment with namespace isolation and autoscaling."
    echo -e "             ${DIM}./deploy.sh -p <name> --k8s${NC}"
    echo ""
    echo -e "  ${BOLD}--edge${NC}      Edge deployment ${YELLOW}(not yet implemented)${NC}"
    echo -e "             Globally distributed honeypots via Fly.io or similar."
    echo -e "             ${DIM}./deploy.sh -p <name> --edge${NC}"
    echo ""
}

# ── Teardown ──────────────────────────────────────────────────
do_teardown() {
    local env_name="${1:-}"
    local tear_all="${2:-false}"

    if [ "$tear_all" = "true" ]; then
        section "Tearing Down All Environments"
        ensure_env_dir
        local files
        files=("${ENV_DIR}"/*.json)
        if [ ! -e "${files[0]:-}" ]; then
            warn "No environments registered"
            return 0
        fi
        for f in "${files[@]}"; do
            [ -f "$f" ] || continue
            local json name
            json=$(cat "$f")
            name=$(json_val "$json" "name")
            teardown_single "$name" "$json"
        done
        info "All environments torn down"
        return 0
    fi

    if [ -z "$env_name" ]; then
        error "Specify an environment to tear down, or use --all"
        echo -e "  ${DIM}./deploy.sh --teardown <name>${NC}"
        echo -e "  ${DIM}./deploy.sh --teardown --all${NC}"
        echo ""
        echo -e "  ${DIM}See registered environments: ./deploy.sh --list${NC}"
        exit 1
    fi

    local json
    json=$(load_env "$env_name") || exit 1
    section "Tearing Down: ${env_name}"
    teardown_single "$env_name" "$json"
}

teardown_single() {
    local name="$1" json="$2"
    local mode compose_project namespace
    mode=$(json_val "$json" "mode")

    case "$mode" in
        docker-compose|docker)
            compose_project=$(json_val "$json" "compose_project")
            info "Stopping containers for ${name} (project: ${compose_project})..."
            COMPOSE_PROJECT_NAME="$compose_project" \
                $COMPOSE_CMD -f docker-compose.yml down -v 2>&1 || true
            info "Removing LABYRINTH images for ${name}..."
            docker images --filter "label=project=labyrinth" -q | xargs -r docker rmi 2>/dev/null || true
            ;;
        k8s)
            namespace=$(json_val "$json" "namespace")
            info "Tearing down Kubernetes namespace: ${namespace}"
            warn "K8s teardown not yet implemented (would run: kubectl delete namespace ${namespace})"
            ;;
        edge)
            info "Tearing down edge deployment: ${name}"
            warn "Edge teardown not yet implemented"
            ;;
        *)
            warn "Unknown mode '${mode}' for ${name}, removing registry entry only"
            ;;
    esac

    remove_env "$name"
}

# ── Status ────────────────────────────────────────────────────
do_status() {
    local env_name="${1:-}"

    if [ -n "$env_name" ]; then
        # Status for a specific environment
        local json
        json=$(load_env "$env_name") || exit 1
        section "Environment: ${env_name}"

        local type mode created compose_project namespace
        type=$(json_val "$json" "type")
        mode=$(json_val "$json" "mode")
        created=$(json_val "$json" "created")

        echo -e "  ${BOLD}Name:${NC}     ${env_name}"
        echo -e "  ${BOLD}Type:${NC}     ${type}"
        echo -e "  ${BOLD}Mode:${NC}     ${mode}"
        echo -e "  ${BOLD}Created:${NC}  ${created}"
        echo ""

        case "$mode" in
            docker-compose|docker)
                compose_project=$(json_val "$json" "compose_project")
                info "Container status (project: ${compose_project}):"
                echo ""
                COMPOSE_PROJECT_NAME="$compose_project" \
                    $COMPOSE_CMD -f docker-compose.yml ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || \
                    warn "Could not retrieve container status"
                ;;
            k8s)
                namespace=$(json_val "$json" "namespace")
                warn "K8s status not yet implemented (would run: kubectl get pods -n ${namespace})"
                ;;
            edge)
                warn "Edge status not yet implemented"
                ;;
        esac
        echo ""
        return 0
    fi

    # Status for all environments
    ensure_env_dir
    local files
    files=("${ENV_DIR}"/*.json)
    if [ ! -e "${files[0]:-}" ]; then
        section "LABYRINTH Status"
        warn "No environments registered"
        echo ""
        echo -e "  ${DIM}Deploy one with: ./deploy.sh -t [name]${NC}"
        return 0
    fi

    section "LABYRINTH Status — All Environments"
    for f in "${files[@]}"; do
        [ -f "$f" ] || continue
        local json name type mode created
        json=$(cat "$f")
        name=$(json_val "$json" "name")
        type=$(json_val "$json" "type")
        mode=$(json_val "$json" "mode")
        created=$(json_val "$json" "created")

        echo -e "  ${BOLD}${name}${NC}  ${DIM}(${type}/${mode}, created ${created})${NC}"

        case "$mode" in
            docker-compose|docker)
                local compose_project
                compose_project=$(json_val "$json" "compose_project")
                local running
                running=$(COMPOSE_PROJECT_NAME="$compose_project" \
                    $COMPOSE_CMD -f docker-compose.yml ps --format "table {{.Name}}\t{{.Status}}" 2>/dev/null || echo "")
                if [ -n "$running" ]; then
                    echo "$running" | sed 's/^/    /'
                else
                    echo -e "    ${YELLOW}No containers running${NC}"
                fi
                ;;
            k8s)
                echo -e "    ${DIM}K8s status: not yet implemented${NC}"
                ;;
            edge)
                echo -e "    ${DIM}Edge status: not yet implemented${NC}"
                ;;
        esac
        echo ""
    done
}

# ── Argument Parsing ──────────────────────────────────────────
parse_args() {
    local action=""
    local env_name=""
    local prod_mode=""
    local tear_all="false"

    if [ $# -eq 0 ]; then
        usage
        exit 0
    fi

    while [ $# -gt 0 ]; do
        case "$1" in
            -t)
                action="test"
                shift
                # Next arg is optional env name (if it doesn't start with -)
                if [ $# -gt 0 ] && [[ ! "$1" =~ ^- ]]; then
                    env_name="$1"
                    shift
                fi
                ;;
            -p)
                action="prod"
                shift
                # Next arg is optional env name (if it doesn't start with -)
                if [ $# -gt 0 ] && [[ ! "$1" =~ ^- ]]; then
                    env_name="$1"
                    shift
                fi
                ;;
            --docker)
                prod_mode="docker"
                shift
                ;;
            --k8s)
                prod_mode="k8s"
                shift
                ;;
            --edge)
                prod_mode="edge"
                shift
                ;;
            --status)
                action="status"
                shift
                # Next arg is optional env name
                if [ $# -gt 0 ] && [[ ! "$1" =~ ^- ]]; then
                    env_name="$1"
                    shift
                fi
                ;;
            --teardown)
                action="teardown"
                shift
                # Next arg is optional env name or --all
                if [ $# -gt 0 ]; then
                    if [ "$1" = "--all" ]; then
                        tear_all="true"
                        shift
                    elif [[ ! "$1" =~ ^- ]]; then
                        env_name="$1"
                        shift
                    fi
                fi
                ;;
            --all)
                tear_all="true"
                shift
                ;;
            --list)
                action="list"
                shift
                ;;
            --help|-h)
                action="help"
                shift
                ;;
            --test-mode)
                # Backwards compat
                action="test"
                shift
                ;;
            --production)
                # Backwards compat
                action="prod"
                shift
                ;;
            *)
                error "Unknown option: $1"
                echo ""
                usage
                exit 1
                ;;
        esac
    done

    # ── Dispatch ──────────────────────────────────────────────
    case "$action" in
        test)
            preflight
            deploy_test "$env_name"
            ;;
        prod)
            if [ -z "$prod_mode" ] && [ -z "$env_name" ]; then
                show_prod_types
                exit 0
            fi
            if [ -z "$prod_mode" ]; then
                error "Production deploy requires a type flag: --docker, --k8s, or --edge"
                echo ""
                show_prod_types
                exit 1
            fi
            case "$prod_mode" in
                docker)  deploy_prod_docker "$env_name" ;;
                k8s)     deploy_prod_k8s "$env_name" ;;
                edge)    deploy_prod_edge "$env_name" ;;
            esac
            ;;
        status)
            do_status "$env_name"
            ;;
        teardown)
            do_teardown "$env_name" "$tear_all"
            ;;
        list)
            list_envs
            ;;
        help)
            usage
            ;;
        *)
            usage
            ;;
    esac
}

# ── Main ──────────────────────────────────────────────────────
cd "$SCRIPT_DIR"

banner
parse_args "$@"
