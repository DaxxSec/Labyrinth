#!/usr/bin/env bash
# ═══════════════════════════════════════════════════════════════
#  Project LABYRINTH — Build from Source
#  Installs Go (if needed), builds the CLI binary from source,
#  and installs it to ~/.local/bin so `labyrinth` works from
#  anywhere. For pre-built binaries, see GitHub Releases.
# ═══════════════════════════════════════════════════════════════
set -euo pipefail

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
DIM='\033[2m'
BOLD='\033[1m'
NC='\033[0m'

info()    { echo -e "  ${GREEN}[+]${NC} $1"; }
warn()    { echo -e "  ${YELLOW}[!]${NC} $1"; }
error()   { echo -e "  ${RED}[✗]${NC} $1"; }
section() { echo -e "\n  ${GREEN}━━━ $1 ━━━${NC}\n"; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${HOME}/.local/bin"
MIN_GO_VERSION="1.22"

# ── Check if a command exists ────────────────────────────────
has() { command -v "$1" &> /dev/null; }

# ── Compare semver (returns 0 if $1 >= $2) ───────────────────
version_gte() {
    printf '%s\n%s' "$2" "$1" | sort -V -C
}

# ── Detect OS and arch ───────────────────────────────────────
detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64)  ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *)
            error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
}

# ── Install Go ───────────────────────────────────────────────
install_go() {
    section "Installing Go"

    detect_platform

    # Get latest stable Go version
    local go_version
    go_version=$(curl -fsSL "https://go.dev/VERSION?m=text" | head -1)
    local tarball="${go_version}.${OS}-${ARCH}.tar.gz"
    local url="https://go.dev/dl/${tarball}"

    info "Downloading ${go_version} for ${OS}/${ARCH}..."

    local tmpdir
    tmpdir=$(mktemp -d)
    trap "rm -rf '$tmpdir'" EXIT

    curl -fsSL -o "${tmpdir}/${tarball}" "$url"

    # Install to ~/.local/go (no sudo needed)
    local go_install_dir="${HOME}/.local/go"
    if [ -d "$go_install_dir" ]; then
        info "Removing previous Go installation at ${go_install_dir}..."
        rm -rf "$go_install_dir"
    fi

    mkdir -p "${HOME}/.local"
    tar -C "${HOME}/.local" -xzf "${tmpdir}/${tarball}"

    export PATH="${go_install_dir}/bin:${PATH}"

    if has go; then
        local installed_version
        installed_version=$(go version | sed -n 's/.*go\([0-9][0-9.]*\).*/\1/p')
        info "Go ${installed_version} installed to ${go_install_dir}"
    else
        error "Go installation failed"
        exit 1
    fi

    # Advise on PATH if needed
    local shell_profile=""
    if [ -f "${HOME}/.zshrc" ]; then
        shell_profile="${HOME}/.zshrc"
    elif [ -f "${HOME}/.bashrc" ]; then
        shell_profile="${HOME}/.bashrc"
    elif [ -f "${HOME}/.profile" ]; then
        shell_profile="${HOME}/.profile"
    fi

    local path_line="export PATH=\"${go_install_dir}/bin:\$PATH\""
    if [ -n "$shell_profile" ]; then
        if ! grep -qF "${go_install_dir}/bin" "$shell_profile" 2>/dev/null; then
            echo "" >> "$shell_profile"
            echo "# Go (installed by LABYRINTH)" >> "$shell_profile"
            echo "$path_line" >> "$shell_profile"
            info "Added Go to PATH in ${shell_profile}"
        fi
    else
        warn "Could not find shell profile to update PATH"
        echo -e "  ${DIM}Add this to your shell profile:${NC}"
        echo -e "  ${DIM}${path_line}${NC}"
    fi
}

# ── Check Go version ─────────────────────────────────────────
check_go() {
    if ! has go; then
        return 1
    fi
    local current
    current=$(go version | sed -n 's/.*go\([0-9][0-9.]*\).*/\1/p')
    if version_gte "$current" "$MIN_GO_VERSION"; then
        return 0
    fi
    warn "Go ${current} found but ${MIN_GO_VERSION}+ required"
    return 1
}

# ── Main ─────────────────────────────────────────────────────
echo ""
echo -e "  ${GREEN}██╗      █████╗ ██████╗ ██╗   ██╗██████╗ ██╗███╗   ██╗████████╗██╗  ██╗${NC}"
echo -e "  ${GREEN}██║     ██╔══██╗██╔══██╗╚██╗ ██╔╝██╔══██╗██║████╗  ██║╚══██╔══╝██║  ██║${NC}"
echo -e "  ${GREEN}██║     ███████║██████╔╝ ╚████╔╝ ██████╔╝██║██╔██╗ ██║   ██║   ███████║${NC}"
echo -e "  ${GREEN}██║     ██╔══██║██╔══██╗  ╚██╔╝  ██╔══██╗██║██║╚██╗██║   ██║   ██╔══██║${NC}"
echo -e "  ${GREEN}███████╗██║  ██║██████╔╝   ██║   ██║  ██║██║██║ ╚████║   ██║   ██║  ██║${NC}"
echo -e "  ${GREEN}╚══════╝╚═╝  ╚═╝╚═════╝    ╚═╝   ╚═╝  ╚═╝╚═╝╚═╝  ╚═══╝   ╚═╝   ╚═╝  ╚═╝${NC}"
echo ""
echo -e "  ${DIM}Installer${NC}"
echo ""

section "Checking Prerequisites"

# Check/install Go
if check_go; then
    local_go_version=$(go version | sed -n 's/.*go\([0-9][0-9.]*\).*/\1/p')
    info "Go ${local_go_version} found"
else
    warn "Go not found or version too old"
    echo ""
    echo -e "  ${BOLD}Go ${MIN_GO_VERSION}+ is required to build LABYRINTH.${NC}"
    echo ""
    read -rp "  Install Go automatically? [Y/n] " answer
    case "${answer:-Y}" in
        [Yy]|"")
            install_go
            ;;
        *)
            error "Go is required. Install it from https://go.dev/dl/ and re-run this script."
            exit 1
            ;;
    esac
fi

# Check Docker (warn only — not required for build)
if has docker; then
    info "Docker found"
else
    warn "Docker not found — needed for deploying environments, not for building"
fi

section "Building LABYRINTH CLI"

cd "${SCRIPT_DIR}/../cli"

info "Fetching dependencies..."
go mod download 2>&1 | while read -r line; do
    echo -e "    ${DIM}${line}${NC}"
done

info "Building binary..."
go build -ldflags="-s -w" -o labyrinth .
binary_size=$(du -h labyrinth | cut -f1 | xargs)
info "Built labyrinth binary (${binary_size})"

section "Installing"

mkdir -p "$INSTALL_DIR"
cp labyrinth "${INSTALL_DIR}/labyrinth"
chmod +x "${INSTALL_DIR}/labyrinth"
info "Installed to ${INSTALL_DIR}/labyrinth"

# Check PATH
in_path=false
for p in $(echo "$PATH" | tr ':' '\n'); do
    if [ "$p" = "$INSTALL_DIR" ]; then
        in_path=true
        break
    fi
done

if [ "$in_path" = false ]; then
    warn "${INSTALL_DIR} is not in your PATH"

    local shell_profile=""
    if [ -f "${HOME}/.zshrc" ]; then
        shell_profile="${HOME}/.zshrc"
    elif [ -f "${HOME}/.bashrc" ]; then
        shell_profile="${HOME}/.bashrc"
    elif [ -f "${HOME}/.profile" ]; then
        shell_profile="${HOME}/.profile"
    fi

    path_line="export PATH=\"${INSTALL_DIR}:\$PATH\""
    if [ -n "$shell_profile" ]; then
        if ! grep -qF "${INSTALL_DIR}" "$shell_profile" 2>/dev/null; then
            echo "" >> "$shell_profile"
            echo "# LABYRINTH CLI" >> "$shell_profile"
            echo "$path_line" >> "$shell_profile"
            info "Added ${INSTALL_DIR} to PATH in ${shell_profile}"
            warn "Restart your terminal or run: source ${shell_profile}"
        fi
    else
        echo -e "  ${DIM}Add to your shell profile:${NC}"
        echo -e "  ${DIM}${path_line}${NC}"
    fi
fi

section "Done"

info "LABYRINTH CLI installed successfully"
echo ""
echo -e "  ${BOLD}Get started:${NC}"
echo -e "    ${DIM}labyrinth --help${NC}            Show all commands"
echo -e "    ${DIM}labyrinth deploy -t${NC}         Deploy a test environment"
echo -e "    ${DIM}labyrinth tui${NC}               Launch the TUI dashboard"
echo ""
