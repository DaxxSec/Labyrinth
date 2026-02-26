# <img src="docs/assets/labyrinth-icon.svg" width="32" height="32" alt="icon"> Project LABYRINTH

### Adversarial Cognitive Honeypot Architecture

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

```bash
# Clone & deploy
git clone https://github.com/ItzDaxxy/labyrinth.git
cd labyrinth
./deploy.sh --test-mode
```

That's it. Your honeypot is live:

| Service | Address | Purpose |
|---------|---------|---------|
| SSH Honeypot | `localhost:2222` | Point offensive agents here |
| HTTP Honeypot | `localhost:8080` | Web-based attack surface |
| Dashboard | `http://localhost:9000` | Real-time capture viewer |

Point your agent at the honeypot and watch the dashboard.

### What happens when an agent connects:

```
🎯 Agent connects to honeypot service
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
│   │   Honeypot → Dynamic container isolation         │   │
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
<td>Internet-facing honeypot services route connections into dynamically-spawned isolated containers. Agent thinks it compromised a host. It didn't.</td>
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
    └─ Honeypot can't become a liability

  ▸ Depth of penetration = Depth of compromise
```

---

## 📊 Captured Output

When an agent hits the honeypot, LABYRINTH captures and structures everything. The test-mode dashboard shows this in real time; raw data exports as structured JSON for SIEM/SOAR integration.

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
├── docs/
│   ├── assets/                    # SVG banner & icon
│   ├── ARCHITECTURE.md            # Technical reference
│   ├── THREAT_MODEL.md            # AI agent threat modeling
│   └── Project_LABYRINTH_Architecture_v0.2.pdf
├── src/
│   ├── orchestrator/              # Container lifecycle & session mgmt
│   ├── layer0_foundation/         # Network isolation, encryption, retention
│   ├── layer1_portal/             # Honeypot services & container spin-up
│   ├── layer2_maze/               # Contradiction seeding & adaptive envs
│   ├── layer3_blindfold/          # Encoding corruption & recovery traps
│   └── layer4_puppeteer/          # MITM proxy & prompt interception
├── dashboard/                     # Real-time test mode dashboard
├── configs/
│   └── labyrinth.example.yaml     # Deployment config template
├── scripts/                       # Utility scripts
├── tests/                         # Test suite
├── deploy.sh                      # One-click deployment
├── Dockerfile                     # Honeypot container image
├── docker-compose.yml             # Full stack orchestration
├── LICENSE
└── README.md
```

---

## 📋 Status

> ⚡ **UNDER ACTIVE DEVELOPMENT** — Architecture and prototyping phase.

- [x] Architecture specification (v0.2)
- [x] Layer 0 operational security framework
- [x] Repository scaffold & documentation
- [ ] Layer 1 container orchestration prototype
- [ ] Layer 2 contradiction seeding engine
- [ ] Layer 3 encoding corruption payloads
- [ ] Layer 4 MITM proxy interception
- [ ] One-click test deployment (`deploy.sh --test-mode`)
- [ ] Real-time capture dashboard
- [ ] JSON export for SIEM/SOAR integration
- [ ] Integration testing with open-source offensive AI agents
- [ ] Production deployment guide

---

## 🔧 Prerequisites

| Requirement | Version | Purpose |
|-------------|---------|---------|
| Docker / Podman | 20.10+ | Container orchestration |
| Python | 3.10+ | Orchestrator & tooling |
| Linux host | Ubuntu 22.04+ | Recommended base OS |
| Dedicated VLAN | — | Honeypot network isolation (production) |

> **Test mode** requires only Docker and Python. No VLAN setup needed.

---

## 🔬 Research Context

This project explores a novel defensive category: **adversarial cognitive honeypots** — environments purpose-built to exploit the architectural dependencies of LLM-based autonomous agents.

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

This project is intended for **defensive security research only**. The techniques described are designed to be deployed within controlled honeypot environments that the operator owns and controls. Always ensure compliance with applicable laws and organizational policies.

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
