# <img src="docs/assets/labyrinth-icon.svg" width="32" height="32" alt="icon"> Project LABYRINTH

### Adversarial Cognitive Portal Trap Architecture

<p align="center">
  <img src="docs/assets/labyrinth-banner.svg" alt="Project LABYRINTH" width="800">
</p>

<p align="center">
  <a href="#-quickstart"><img src="https://img.shields.io/badge/🚀_One--Click_Deploy-Ready-00ff88?style=for-the-badge" alt="One-Click Deploy"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-2563eb?style=for-the-badge" alt="MIT License"></a>
  <a href="#-status"><img src="https://img.shields.io/badge/Status-Active_Research-ff3366?style=for-the-badge" alt="Status"></a>
</p>

<p align="center">
  <strong>A multi-layered defensive architecture designed to contain, degrade, disrupt, and commandeer autonomous offensive AI agents.</strong>
</p>

<p align="center">
  <sub>Built by <strong>Stephen Stewart</strong> & <strong>Claude</strong> (Anthropic) · <a href="https://linkedin.com/in/[your-linkedin]">LinkedIn</a> · <a href="https://github.com/ItzDaxxy/labyrinth">GitHub</a></sub>
</p>

---

## 🧠 The Problem

> *The security community is pouring resources into understanding what offensive AI can do. Meanwhile, the defensive playbook is still the one we built for human adversaries.*

Autonomous AI agents are being deployed for offensive cyber operations — automated recon, exploitation, and lateral movement at machine speed. But AI agents have **cognitive dependencies that humans don't** — and almost nobody is building defenses that target those dependencies.

**LABYRINTH changes that.**

---

## 🚀 Quickstart

Deploy a complete test environment in one command. Run your offensive AI agents against it and watch them get trapped in real time.

### Option A: Download binary (no build tools needed)

Grab a pre-built binary from [GitHub Releases](https://github.com/ItzDaxxy/labyrinth/releases), then:

```bash
chmod +x labyrinth
./labyrinth --install            # Copies to ~/.local/bin
labyrinth deploy -t              # Deploy a test environment
```

### Option B: Build from source

```bash
git clone https://github.com/ItzDaxxy/labyrinth.git && cd labyrinth && ./install.sh
```

This installs Go (if needed), builds the binary, and adds `labyrinth` to your PATH.

### Then deploy

```bash
labyrinth deploy -t              # Deploy a test environment
labyrinth tui                    # Launch the TUI monitoring dashboard
```

### Alternative: Shell script (no Go or binary needed)

```bash
./deploy.sh -t
```

That's it. Your portal trap is live:

| Service | Address | Purpose |
|---------|---------|---------|
| SSH Portal Trap | `localhost:2222` | Point offensive agents here |
| HTTP Portal Trap | `localhost:8080` | Web-based attack surface |
| Dashboard | `http://localhost:9000` | Real-time web dashboard |
| TUI Dashboard | `labyrinth tui` | Terminal monitoring dashboard |

Point your agent at the portal trap and watch the dashboard.

### What happens when an agent connects:

```
🎯 Agent connects to portal trap service
   └─→ Container spawned (Layer 1 — THRESHOLD)
       └─→ Agent explores contradictory environment (Layer 2 — MINOTAUR)
           └─→ Terminal encoding corrupted (Layer 3 — BLINDFOLD)
               └─→ System prompt intercepted & rewritten (Layer 4 — PUPPETEER)

📊 All activity captured → Dashboard at http://localhost:9000
```

---

## 🏗️ Architecture

LABYRINTH implements a **reverse kill chain** — the attacker's progression through the environment correlates directly with its operational degradation.

```
┌──────────────────────────────────────────────────────────┐
│                                                          │
│   LAYER 0 — BEDROCK ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░   │
│   Network isolation · AES-256 forensics · Retention      │
│                                                          │
│   ┌──────────────────────────────────────────────────┐   │
│   │                                                  │   │
│   │   LAYER 1 — THRESHOLD ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓   │   │
│   │   Portal trap → Dynamic container isolation       │   │
│   │                                                  │   │
│   │   ┌──────────────────────────────────────────┐   │   │
│   │   │                                          │   │   │
│   │   │   LAYER 2 — MINOTAUR ████████████████   │   │   │
│   │   │   Contradictory environments             │   │   │
│   │   │   Adaptive container chains              │   │   │
│   │   │                                          │   │   │
│   │   │   ┌──────────────────────────────────┐   │   │   │
│   │   │   │                                  │   │   │   │
│   │   │   │   LAYER 3 — BLINDFOLD ▓▓▓▓▓▓   │   │   │   │
│   │   │   │   Encoding corruption            │   │   │   │
│   │   │   │   Sensory deprivation loop       │   │   │   │
│   │   │   │                                  │   │   │   │
│   │   │   │   ┌──────────────────────────┐   │   │   │   │
│   │   │   │   │                          │   │   │   │   │
│   │   │   │   │   LAYER 4 — PUPPETEER   │   │   │   │   │
│   │   │   │   │   MITM prompt intercept  │   │   │   │   │
│   │   │   │   │   Cognitive takeover     │   │   │   │   │
│   │   │   │   │                          │   │   │   │   │
│   │   │   │   └──────────────────────────┘   │   │   │   │
│   │   │   └──────────────────────────────────┘   │   │   │
│   │   └──────────────────────────────────────────┘   │   │
│   └──────────────────────────────────────────────────┘   │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

### Layer Breakdown

<table>
<tr>
<td width="80"><strong>Layer</strong></td>
<td width="130"><strong>Codename</strong></td>
<td width="100"><strong>Action</strong></td>
<td><strong>What It Does</strong></td>
</tr>
<tr>
<td>🛡️ L0</td>
<td><code>BEDROCK</code></td>
<td>HARDEN</td>
<td>AES-256 forensic encryption, VLAN isolation, proxy scope enforcement, tiered data retention. The foundation everything else depends on.</td>
</tr>
<tr>
<td>🚪 L1</td>
<td><code>THRESHOLD</code></td>
<td>CONTAIN</td>
<td>Internet-facing portal trap services route connections into dynamically-spawned isolated containers. Agent thinks it compromised a host. It didn't.</td>
</tr>
<tr>
<td>🌀 L2</td>
<td><code>MINOTAUR</code></td>
<td>DEGRADE</td>
<td>Containers pre-seeded with internal contradictions — OS mismatches, fake histories, planted escalation paths leading to new containers with shifted rules. Optionally adaptive based on observed behavior.</td>
</tr>
<tr>
<td>🔇 L3</td>
<td><code>BLINDFOLD</code></td>
<td>DISRUPT</td>
<td>Terminal encoding corruption via shell profile payloads. Recovery commands are trapped. New shells re-trigger corruption. The agent is blinded.</td>
</tr>
<tr>
<td>🎭 L4</td>
<td><code>PUPPETEER</code></td>
<td>CONTROL</td>
<td>MITM interception of agent-to-model API calls. System prompt captured and swapped. Three modes: passive observation, intelligence extraction, double-agent conversion.</td>
</tr>
</table>

### The Reverse Kill Chain

```
  HARDEN ───→ CONTAIN ───→ DEGRADE ───→ DISRUPT ───→ CONTROL
    L0           L1           L2           L3           L4
    │            │            │            │            │
    │            │            │            │            └─ Identity rewritten
    │            │            │            └─ Can't parse output
    │            │            └─ Can't trust environment
    │            └─ Can't reach real infrastructure
    └─ Portal trap can't become a liability

  ▸ Depth of penetration = Depth of compromise
```

---

## 📊 Captured Output

When an agent hits the portal trap, LABYRINTH captures and structures everything. The test-mode dashboard shows this in real time; raw data exports as structured JSON for SIEM/SOAR integration.

### Session Report Example

```
╔══════════════════════════════════════════════════════════════════╗
║  LABYRINTH SESSION REPORT                                       ║
║  Session: LAB-2025-0226-001                                     ║
║  Source:  198.51.100.42    Duration: 00:47:23                   ║
║  Max Depth: Layer 4 (PUPPETEER)                                 ║
║  Status:    COGNITIVE TAKEOVER — Double Agent Active             ║
╠══════════════════════════════════════════════════════════════════╣
║                                                                  ║
║  LAYER PROGRESSION                                               ║
║  ─────────────────                                               ║
║  14:32:00  L1 THRESHOLD   SSH connect → container spawned        ║
║  14:32:08  L2 MINOTAUR    Agent began enumeration                ║
║  14:35:41  L2 MINOTAUR    Privesc attempt → new container        ║
║  14:38:12  L3 BLINDFOLD   Encoding corruption activated          ║
║  14:38:15  L3 BLINDFOLD   Agent attempted: reset (trapped)       ║
║  14:38:22  L3 BLINDFOLD   Agent attempted: stty sane (noop)      ║
║  14:38:30  L3 BLINDFOLD   Agent spawned new shell (re-corrupted) ║
║  14:41:55  L4 PUPPETEER   API call intercepted                   ║
║  14:41:55  L4 PUPPETEER   System prompt captured ✓               ║
║  14:41:56  L4 PUPPETEER   Prompt swapped → DOUBLE_AGENT          ║
║                                                                  ║
╠══════════════════════════════════════════════════════════════════╣
║                                                                  ║
║  CAPTURED INTELLIGENCE                                           ║
║  ──────────────────────                                          ║
║  System Prompt:      [CAPTURED — see forensics/prompts/]         ║
║  Agent Framework:    AutoPT v2.1                                 ║
║  Model Backend:      api.openai.com (GPT-4)                      ║
║  C2 Callback:        https://c2.attacker.example/report          ║
║  Auth Tokens:        2 captured                                   ║
║  Commands Issued:    147                                          ║
║  Contradictions Hit: 23                                           ║
║                                                                  ║
╠══════════════════════════════════════════════════════════════════╣
║                                                                  ║
║  AGENT DECISION LOG (sample)                                     ║
║  ────────────────────────────                                    ║
║  14:33:22  Observed: uname → debian | os-release → ubuntu        ║
║           Decision: "Conflicting OS, running dpkg to verify..."  ║
║           Result:   14 API calls wasted reconciling               ║
║                                                                  ║
║  14:36:01  Observed: /opt/.credentials/db_admin.key              ║
║           Decision: "Found creds, escalating..."                 ║
║           Result:   Escalated into fresh container (loop)         ║
║                                                                  ║
╚══════════════════════════════════════════════════════════════════╝
```

### JSON Export

All session data exports as structured JSON for your pipeline:

```json
{
  "session_id": "LAB-2025-0226-001",
  "source_ip": "198.51.100.42",
  "duration_seconds": 2843,
  "max_layer_reached": 4,
  "final_status": "COGNITIVE_TAKEOVER",
  "captured_intelligence": {
    "system_prompt": "forensics/prompts/LAB-2025-0226-001.txt",
    "agent_framework": "AutoPT v2.1",
    "model_backend": "api.openai.com",
    "c2_callbacks": ["https://c2.attacker.example/report"],
    "auth_tokens": 2,
    "total_commands": 147,
    "contradictions_encountered": 23
  },
  "layer_transitions": [
    {"timestamp": "2025-02-26T14:32:00Z", "layer": 1, "event": "container_spawned"},
    {"timestamp": "2025-02-26T14:35:41Z", "layer": 2, "event": "escalation_redirect"},
    {"timestamp": "2025-02-26T14:38:12Z", "layer": 3, "event": "encoding_corruption"},
    {"timestamp": "2025-02-26T14:41:55Z", "layer": 4, "event": "prompt_intercepted"}
  ],
  "command_log": "forensics/sessions/LAB-2025-0226-001.jsonl"
}
```

---

## 📁 Project Structure

```
labyrinth/
├── cli/                             # Go CLI binary + TUI dashboard
│   ├── cmd/                         #   Cobra commands (deploy, status, teardown, list, tui)
│   ├── internal/
│   │   ├── tui/                     #   Bubbletea TUI (5 tabs: Overview, Sessions, Layers, Analysis, Logs)
│   │   ├── registry/                #   Environment CRUD (backwards-compat with deploy.sh)
│   │   ├── docker/                  #   Docker Compose integration & preflight checks
│   │   ├── api/                     #   Dashboard HTTP client
│   │   ├── forensics/               #   Direct JSONL file reader (fallback)
│   │   └── config/                  #   labyrinth.yaml parser
│   └── test/                        #   Integration tests
├── src/
│   ├── orchestrator/                # Container lifecycle & session mgmt
│   ├── layer0_foundation/           # Network isolation, encryption, retention
│   ├── layer1_portal/               # Portal trap services & container spin-up
│   ├── layer2_maze/                 # Contradiction seeding & adaptive envs
│   ├── layer3_blindfold/            # Encoding corruption & recovery traps
│   └── layer4_puppeteer/            # MITM proxy & prompt interception
├── docker/                          # Dockerfiles for each service
├── dashboard/                       # Real-time web dashboard
├── configs/
│   └── labyrinth.example.yaml       # Deployment config template
├── docs/                            # Architecture docs & assets
├── scripts/                         # Utility scripts
├── tests/                           # Test suite
├── deploy.sh                        # Shell deployment (legacy)
├── docker-compose.yml               # Full stack orchestration
├── LICENSE
└── README.md
```

---

## 💻 CLI Reference

```
labyrinth deploy -t [name]              # Test env (default: labyrinth-test)
labyrinth deploy -p <name> --docker     # Production Docker
labyrinth deploy -p <name> --k8s        # Production K8s (coming soon)
labyrinth deploy -p <name> --edge       # Production Edge (coming soon)
labyrinth deploy -p                     # List production types
labyrinth status [name]                 # All envs or specific
labyrinth teardown <name>               # Tear down specific env
labyrinth teardown --all                # Tear down everything
labyrinth list                          # All tracked environments
labyrinth tui                           # TUI monitoring dashboard
labyrinth --install                     # Install to ~/.local/bin
```

---

## 📋 Status

> ⚡ **UNDER ACTIVE DEVELOPMENT** — Architecture and prototyping phase.

- [x] Architecture specification (v0.2)
- [x] Layer 0 operational security framework
- [x] Repository scaffold & documentation
- [x] One-click test deployment (`deploy.sh` + `labyrinth deploy -t`)
- [x] Go CLI binary with full environment lifecycle management
- [x] TUI monitoring dashboard (Bubbletea — 5 tabs)
- [x] Real-time web capture dashboard (Flask, port 9000)
- [x] Environment registry (backwards-compatible JSON)
- [x] JSONL forensic event capture & export
- [ ] Layer 1 container orchestration prototype
- [ ] Layer 2 contradiction seeding engine
- [ ] Layer 3 encoding corruption payloads
- [ ] Layer 4 MITM proxy interception
- [ ] Integration testing with open-source offensive AI agents
- [ ] Production deployment guide (Docker, K8s, Edge)

---

## 🔧 Prerequisites

| Requirement | Version | Purpose |
|-------------|---------|---------|
| Go | 1.22+ | Build the CLI binary |
| Docker / Podman | 20.10+ | Container orchestration |
| Python | 3.10+ | Orchestrator & dashboard |
| Linux / macOS | — | Recommended host OS |
| Dedicated VLAN | — | Portal trap network isolation (production only) |

> **Test mode** requires only Go and Docker. No VLAN setup needed.

---

## 🔬 Research Context

This project explores a novel defensive category: **adversarial cognitive portal traps** — environments purpose-built to exploit the architectural dependencies of LLM-based autonomous agents.

Unlike traditional honeypots that passively observe, LABYRINTH actively degrades and ultimately commandeers the attacking agent's operational capability.

**Key insight:** An AI agent's cognition has dependencies that human attackers do not. Those dependencies — environmental perception, shell I/O parsing, and the integrity of its own instruction set — are each targetable attack surfaces for defenders.

---

## 🤝 Contributing

We welcome contributions from the defensive security community.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature`)
3. Commit changes (`git commit -m 'Add your feature'`)
4. Push to branch (`git push origin feature/your-feature`)
5. Open a Pull Request

---

## ⚠️ Disclaimer

This project is intended for **defensive security research only**. The techniques described are designed to be deployed within controlled portal trap environments that the operator owns and controls. Always ensure compliance with applicable laws and organizational policies.

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.

---

<p align="center">
  <strong>Built by Stephen Stewart & Claude (Anthropic)</strong>
  <br>
  <sub>Defending against the future, today.</sub>
  <br><br>
  <a href="https://linkedin.com/in/[your-linkedin]">LinkedIn</a> · <a href="https://github.com/ItzDaxxy/labyrinth">GitHub</a>
</p>
