# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Project LABYRINTH is a defensive security architecture that contains, degrades, disrupts, and commandeers autonomous offensive AI agents via a layered "reverse kill chain" (L0 BEDROCK → L1 THRESHOLD → L2 MINOTAUR → L3 BLINDFOLD → L4 PUPPETEER). The deeper an agent penetrates, the more compromised it becomes.

## Build & Run Commands

### Go CLI (primary interface)

```bash
cd cli
go build -o labyrinth .                        # Dev build
go build -ldflags="-s -w" -o labyrinth .       # Optimized build (~7-8MB)
go test ./...                                   # All tests (39 tests)
go test -race ./...                             # With race detector
go test ./internal/registry/...                 # Single package
go test ./test/... -v                           # Integration tests (verbose)
go test -cover ./...                            # Coverage report
```

### Full lifecycle

```bash
./labyrinth deploy -t mylab     # Deploy test environment (Docker Compose)
./labyrinth list                # List all environments
./labyrinth status mylab        # Check environment status
./labyrinth tui                 # Launch TUI monitoring dashboard
./labyrinth teardown mylab      # Tear down environment
./labyrinth --install           # Install binary to ~/.local/bin
```

### Docker stack (deployed by CLI)

```bash
docker compose build            # Build all services
docker compose up -d            # Start all services
docker compose down -v          # Tear down
```

### Attacker agent testing

```bash
./scripts/attacker-setup.sh             # Interactive setup (PentAGI, PentestAgent, Strix, Custom)
./scripts/attacker-setup.sh --teardown  # Remove attacker containers
```

## Architecture

### Repository Layout

- **`cli/`** — Go CLI binary (`labyrinth` command). Cobra for subcommands, Bubbletea v2 for TUI.
- **`src/`** — Python components: `layer1_portal/` (SSH/HTTP portal traps, session logger, orchestrator), `dashboard/` (Flask web dashboard at :9000)
- **`docker/`** — Dockerfiles for all services (portal traps, dashboard, proxy, orchestrator)
- **`configs/`** — `labyrinth.example.yaml` config schema, `docker-compose.override.yml`
- **`scripts/`** — `attacker-setup.sh` (attacker agent provisioning)
- **`docs/`** — GETTING_STARTED, CLI_REFERENCE, TESTING, LAYERS, CAPTURED_OUTPUT, ARCHITECTURE, THREAT_MODEL

### Go CLI structure (`cli/`)

```
cmd/           — Cobra command definitions (deploy, status, teardown, list, tui, install)
internal/
  banner/      — ASCII art banner
  registry/    — Environment CRUD (~/.labyrinth/environments/*.json)
  docker/      — Docker Compose operations + preflight checks (shells out to `docker compose`)
  tui/         — Bubbletea TUI (5 tabs: Overview, Sessions, Layers, Analysis, Logs)
  api/         — HTTP client for Flask dashboard API (:9000/api/stats, /api/sessions)
  forensics/   — Direct JSONL file reader (fallback when dashboard API unavailable)
  config/      — labyrinth.yaml parser
test/          — Integration tests (build binary, test via os/exec)
```

### Data flow

TUI fetches data from: Flask dashboard API (primary, polls every 2s) → Direct JSONL forensics files (fallback) → "No data source" message.

Environment registry lives at `~/.labyrinth/environments/` as JSON files, shared between CLI and deploy.sh.

### Docker topology

Services on `labyrinth-net` (172.30.0.0/24): `labyrinth-ssh` (:2222→22), `labyrinth-http` (:8080→80), `labyrinth-orchestrator`, `labyrinth-proxy`, `labyrinth-dashboard` (:9000). Attacker agents connect to this network in isolated containers.

## Critical: Bubbletea v2 / Lipgloss v2 Import Paths

The Charm libraries moved their v2 module paths. **Always use `charm.land/` not `github.com/charmbracelet/`:**

```go
import (
    tea "charm.land/bubbletea/v2"       // NOT github.com/charmbracelet/bubbletea/v2
    "charm.land/lipgloss/v2"            // NOT github.com/charmbracelet/lipgloss/v2
)
```

### Bubbletea v2 API differences from v1

- `View()` returns `tea.View`, not `string`. Use `tea.NewView(s)` to wrap.
- `tea.WithAltScreen()` does not exist. Set `v.AltScreen = true` on the View struct.
- `lipgloss.Color()` is a function returning `color.Color` (from `image/color`), not a type.

## Terminology

User-facing text says **"portal trap"** (not "honeypot"). Internal Go struct field names may still use `Honeypot` for backwards compatibility with config files.

## Config

All layers configured via `labyrinth.yaml` (see `configs/labyrinth.example.yaml`). The config schema covers L0-L4 settings plus optional SIEM integration.

## Forensic Event Schema (JSONL)

```json
{"timestamp": "...", "session_id": "LAB-001", "layer": 1, "event": "connection", "data": {...}}
```

Written by `src/layer1_portal/session_logger.py`, stored in `/var/labyrinth/forensics/sessions/`.
