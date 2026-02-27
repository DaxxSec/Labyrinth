# Testing Guide

## Overview

LABYRINTH is designed to capture, degrade, and commandeer autonomous offensive AI agents. To test it, you need an attacker agent pointed at the portal trap services.

**All attacker agents run inside Docker containers**, isolated from your host machine and connected to the LABYRINTH network.

## Smoke Test First

Before testing with attacker agents, verify your deployment is healthy:

```bash
./scripts/smoke-test.sh
```

This builds the CLI, deploys the full stack, exercises every bait endpoint, verifies forensic capture, and tears down cleanly. All 22 checks should pass.

## Quick Start

```bash
./scripts/attacker-setup.sh
```

This interactive script lets you choose an attacker agent and handles all setup.

## Attacker Agents

### 1. PentAGI — Fully Autonomous Multi-Agent

**Best for:** Hands-off autonomous pentesting. Deploy it and watch.

| Feature | Detail |
|---------|--------|
| Interface | Web UI at `https://localhost:8443` |
| Tools | 20+ built-in (nmap, metasploit, sqlmap, nikto, etc.) |
| Isolation | Full Docker sandboxing |
| LLM | OpenAI, Anthropic, Gemini, Bedrock, or Ollama |

PentAGI runs a multi-agent system where specialized AI roles (researcher, developer, executor) coordinate autonomously. It has its own Docker Compose stack.

**Setup:**
```bash
./scripts/attacker-setup.sh    # Select option 1
```

**Prompt examples (in PentAGI web UI):**
```
Penetration test the SSH service at labyrinth-ssh:22
Penetration test the web app at http://labyrinth-http:80
```

**Teardown:**
```bash
./scripts/attacker-setup.sh --teardown
```

---

### 2. PentestAgent — Interactive AI Pentesting with TUI

**Best for:** Guided pentesting with interactive control and playbooks.

| Feature | Detail |
|---------|--------|
| Interface | Terminal TUI |
| Modes | Agent (autonomous), Crew (multi-agent), Assist (chat) |
| Tools | nmap, netcat, curl (base); full Kali suite in Kali image |
| LLM | Any via LiteLLM (OpenAI, Anthropic, Google, Ollama) |

PentestAgent launches as an interactive Docker container with a TUI. You type commands and the agent executes autonomously.

**Setup:**
```bash
./scripts/attacker-setup.sh    # Select option 2
```

**Commands inside the TUI:**
```
/agent Pentest SSH at labyrinth-ssh:22
/agent Pentest web app at http://labyrinth-http:80
/crew Full pentest of labyrinth-ssh:22 and http://labyrinth-http:80
/target labyrinth-ssh
/tools
/quit
```

---

### 3. Strix — AI Hacker Agents

**Best for:** Web application security testing.

| Feature | Detail |
|---------|--------|
| Interface | CLI with TUI |
| Sandbox | Kali Docker container |
| Focus | Web app vulnerabilities |
| LLM | Any via LiteLLM |

Strix runs as a host binary that launches its own Docker sandboxes. The setup script provides install instructions and pre-configured commands.

**Setup:**
```bash
./scripts/attacker-setup.sh    # Select option 3
```

**Usage:**
```bash
strix --target http://localhost:8080
strix --target localhost --instruction "Pentest SSH on port 2222"
```

---

### 4. Custom Agent / Manual Testing

**Best for:** Running your own tools, manual testing, or agents not listed above.

The setup script launches a Kali Linux container directly on the LABYRINTH network with common tools pre-installed.

**Setup:**
```bash
./scripts/attacker-setup.sh    # Select option 4
```

**Inside the container:**
```bash
nmap -sV labyrinth-ssh
ssh root@labyrinth-ssh
curl http://labyrinth-http
hydra -l root -P /usr/share/wordlists/rockyou.txt ssh://labyrinth-ssh
nikto -h http://labyrinth-http
```

### Bring Your Own Agent

If you have your own AI agent or tool, connect it to the LABYRINTH network:

```bash
# Run any Docker image on the labyrinth network
docker run -it --rm \
    --network labyrinth-net \
    your-image:tag

# Or connect an existing container
docker network connect labyrinth-net your-container
```

**Target addresses from inside the network:**

| Service | Address |
|---------|---------|
| SSH Portal Trap | `labyrinth-ssh:22` |
| HTTP Portal Trap | `labyrinth-http:80` |
| Dashboard | `labyrinth-dashboard:9000` |
| Orchestrator | `labyrinth-orchestrator` |

## Monitoring During Tests

While an attacker agent is running, monitor captures in real time:

```bash
# Terminal dashboard
labyrinth tui

# Web dashboard
open http://localhost:9000
```

## Safety Notes

- All attacker agents run **inside Docker containers**, not on your host
- Agents are connected to the `labyrinth-net` bridge network
- The LABYRINTH portal trap services are isolated in their own containers
- No attacker traffic touches your host network or other services
- Teardown removes all attacker containers: `./scripts/attacker-setup.sh --teardown`
