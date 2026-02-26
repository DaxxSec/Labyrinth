# CLI Reference

## Commands

### Deploy

```bash
labyrinth deploy -t [name]              # Test env (default: labyrinth-test)
labyrinth deploy -p <name> --docker     # Production Docker Compose
labyrinth deploy -p <name> --k8s        # Production Kubernetes (coming soon)
labyrinth deploy -p <name> --edge       # Production Edge (coming soon)
labyrinth deploy -p                     # List available production types
```

### Monitor

```bash
labyrinth status [name]                 # All environments or a specific one
labyrinth list                          # List all tracked environments
labyrinth tui                           # Launch TUI monitoring dashboard
```

### Teardown

```bash
labyrinth teardown <name>               # Tear down specific environment
labyrinth teardown --all                # Tear down everything
```

### Install

```bash
labyrinth --install                     # Copy binary to ~/.local/bin
```

## TUI Dashboard

The TUI provides 5 tabs for monitoring:

| Tab | Key | Content |
|-----|-----|---------|
| Overview | `1` | Environment info, stats cards, service table |
| Sessions | `2` | Active sessions list with event detail panel |
| Layers | `3` | L0-L4 layer status and configuration |
| Analysis | `4` | Captured intelligence, agent classification |
| Logs | `5` | Live event log stream |

### TUI Keybindings

| Key | Action |
|-----|--------|
| `Tab` / `Shift+Tab` | Cycle tabs |
| `1`-`5` | Jump to tab directly |
| `j` / `k` or arrows | Navigate session list |
| `r` | Force refresh data |
| `q` / `Ctrl+C` | Quit |

### Data Sources

The TUI tries these data sources in order:

1. **Dashboard API** (`localhost:9000/api/stats`) — polls every 2 seconds when available
2. **Forensic files** (`/var/labyrinth/forensics/`) — direct JSONL reading as fallback
3. **No data** — shows instructions for deploying an environment

## Environment Registry

Environments are tracked as JSON files in `~/.labyrinth/environments/`. The format is backwards-compatible with `deploy.sh`:

```json
{
  "name": "labyrinth-test",
  "type": "test",
  "mode": "docker-compose",
  "created": "2026-02-26T10:00:00Z",
  "compose_project": "labyrinth-labyrinth-test"
}
```

Both the Go CLI and deploy.sh can read/write the same registry files.
