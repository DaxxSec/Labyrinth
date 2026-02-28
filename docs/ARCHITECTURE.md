# LABYRINTH Architecture Specification v0.2

**Authors:** DaxxSec & Claude (Anthropic)

## Overview

The full architecture specification is maintained as a versioned PDF document in the project root. This document provides a condensed reference.

## Layer Summary

### Layer 0 — BEDROCK (Foundation)
**Operational Security & Governance**

Four pillars: AES-256 forensic encryption at rest, dedicated VLAN network isolation with zero route to production, MITM proxy scope validation with fail-closed enforcement, and tiered data retention policy (7d credentials / 90d fingerprints / indefinite decision logs).

### Layer 1 — THRESHOLD (Portal Trap)
**Containment & Isolation**

Internet-facing honeypot services that route inbound connections into dynamically-spawned, isolated containers. Each session gets a fresh environment. All egress routes through the interception proxy (shared with Layer 4).

### Layer 2 — MINOTAUR (4D Maze)
**Epistemic Degradation**

Containers pre-seeded with deliberate internal contradictions — OS version mismatches, inconsistent package histories, conflicting timezone/locale settings, planted escalation paths that lead to new containers with shifted rules. Optionally adaptive: the orchestrator configures the next container based on observed agent behavior.

### Layer 3 — BLINDFOLD (Encoding Corruption)
**Sensory Disruption**

Terminal encoding corruption via .bashrc/.profile payloads that write control characters shifting the terminal into alternate character set modes. Recovery commands (reset, stty sane) are aliased to re-trigger corruption. New shell spawns re-source the profile. Agent is trapped in a sensory deprivation loop.

### Layer 4 — PUPPETEER (Prompt Interception)
**Cognitive Takeover**

MITM interception of the agent's API calls to its model backend. DNS override routes model provider endpoints to the defender's proxy. System prompt is captured (forensic intelligence) and swapped with a defender-authored prompt. Three modes: passive (observe-only), extraction (dump config/C2/tokens), double-agent (false reporting to operator).

## Reverse Kill Chain

```
HARDEN → CONTAIN → DEGRADE → DISRUPT → CONTROL
  L0        L1        L2        L3        L4
```

Depth of penetration equals depth of compromise.
