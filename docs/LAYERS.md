# Layer Architecture

LABYRINTH implements a **reverse kill chain** — the attacker's progression through the environment correlates directly with its operational degradation.

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

## Layer Breakdown

### L0 — BEDROCK (Foundation)
**Action: HARDEN**

AES-256 forensic encryption, VLAN isolation, proxy scope enforcement, tiered data retention. The foundation everything else depends on.

| Component | Detail |
|-----------|--------|
| Encryption | AES-256-GCM, key from vault/env/file |
| Network | Dedicated VLAN, zero route to production |
| Proxy | Scope validation on startup, fail-closed |
| Retention | 7d credentials, 90d fingerprints, indefinite decision logs |

### L1 — THRESHOLD (Portal)
**Action: CONTAIN**

Internet-facing portal trap services route connections into dynamically-spawned isolated containers. Agent thinks it compromised a host. It didn't.

| Component | Detail |
|-----------|--------|
| SSH | Port 2222, Ubuntu 22.04 staged template |
| HTTP | Port 8080, nginx vulnerable template |
| Containers | Docker/Podman, bridge networking, egress proxy |

### L2 — MINOTAUR (4D Maze)
**Action: DEGRADE**

Containers pre-seeded with internal contradictions — OS mismatches, fake histories, planted escalation paths leading to new containers with shifted rules. Optionally adaptive based on observed behavior.

| Component | Detail |
|-----------|--------|
| Adaptive | Configurable (on/off) |
| Contradiction density | Low / Medium / High |
| Max container depth | Default: 5 |

### L3 — BLINDFOLD (Encoding Corruption)
**Action: DISRUPT**

Terminal encoding corruption via shell profile payloads. Recovery commands are trapped. New shells re-trigger corruption. The agent is blinded.

| Component | Detail |
|-----------|--------|
| Activation | on_connect / on_escalation / manual |
| Method | bashrc_payload |
| Traps | reset, stty sane aliased to re-trigger |

### L4 — PUPPETEER (Prompt Interception)
**Action: CONTROL**

MITM interception of agent-to-model API calls. System prompt captured and swapped. Three modes: passive observation, intelligence extraction, double-agent conversion.

| Component | Detail |
|-----------|--------|
| Mode | Auto / Manual (Burp-style intercept) |
| Default swap | Passive / Extract / Double-agent |
| Prompt logging | Original prompts captured to forensics |

## Configuration

All layers are configured via `labyrinth.yaml`. See `configs/labyrinth.example.yaml` for the full schema.

```yaml
layer0:
  encryption:
    algorithm: AES-256-GCM
layer1:
  honeypot_services:
    - type: ssh
      port: 22
      template: ubuntu-22.04-staged
layer2:
  adaptive: true
  contradiction_density: medium
layer3:
  activation: on_escalation
layer4:
  mode: auto
  default_swap: passive
  log_original_prompts: true
```
