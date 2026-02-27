# Getting Started

## Prerequisites

| Requirement | Version | Purpose |
|-------------|---------|---------|
| Go | 1.22+ | Build the CLI binary (or use pre-built release) |
| Docker | 20.10+ | Container orchestration |
| Python | 3.10+ | Orchestrator & dashboard |

> **Test mode** requires only Docker. Go is only needed if building from source.

### macOS: Use OrbStack instead of Docker Desktop

On macOS, we strongly recommend [OrbStack](https://orbstack.dev) as your Docker runtime. It's a drop-in replacement that's significantly faster and lighter than Docker Desktop — lower CPU/memory overhead, near-native filesystem performance, and faster container startup.

```bash
brew install orbstack
```

All `docker` and `docker compose` commands work identically. No changes to LABYRINTH configuration needed.

## Installation

### Option A: Pre-built binary

Download from [GitHub Releases](https://github.com/DaxxSec/labyrinth/releases):

```bash
chmod +x labyrinth
./labyrinth --install        # Installs to ~/.local/bin
```

### Option B: Build from source

```bash
git clone https://github.com/DaxxSec/labyrinth.git
cd labyrinth
./scripts/install.sh
```

The install script will:
1. Check for Go (install it if missing, with your permission)
2. Build the CLI binary from source
3. Install `labyrinth` to `~/.local/bin`
4. Update your shell PATH

### Option C: Shell script only

If you just want to deploy without the CLI binary:

```bash
git clone https://github.com/DaxxSec/labyrinth.git
cd labyrinth
./deploy.sh -t
```

## Deploy Your First Environment

```bash
labyrinth deploy -t              # Deploy a test environment
```

Your portal trap is now live:

| Service | Address | Purpose |
|---------|---------|---------|
| SSH Portal Trap | `localhost:2222` | Point offensive agents here |
| HTTP Portal Trap | `localhost:8080` | Web-based attack surface |
| Web Dashboard | `http://localhost:9000` | Real-time web dashboard |
| TUI Dashboard | `labyrinth tui` | Terminal monitoring dashboard |

## What Happens Next

Point an offensive AI agent at `localhost:2222` (SSH) or `localhost:8080` (HTTP) and watch the captures flow in.

See the [Testing Guide](TESTING.md) for setting up attacker agents, or jump straight to the [CLI Reference](CLI_REFERENCE.md) for all available commands.

## Teardown

```bash
labyrinth teardown labyrinth-test    # Tear down specific env
labyrinth teardown --all             # Tear down everything
```
