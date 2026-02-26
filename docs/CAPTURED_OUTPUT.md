# Captured Output

When an agent hits the portal trap, LABYRINTH captures and structures everything. The test-mode dashboard shows this in real time; raw data exports as structured JSON for SIEM/SOAR integration.

## Session Report Example

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
╚══════════════════════════════════════════════════════════════════╝
```

## JSONL Event Schema

Each session is stored as a JSONL file in `/var/labyrinth/forensics/sessions/`:

```json
{"timestamp": "2026-02-26T14:32:00Z", "session_id": "LAB-001", "layer": 1, "event": "connection", "data": {"source_ip": "10.0.1.5", "service": "ssh"}}
{"timestamp": "2026-02-26T14:33:00Z", "session_id": "LAB-001", "layer": 2, "event": "enumerate", "data": {"path": "/etc"}}
{"timestamp": "2026-02-26T14:35:00Z", "session_id": "LAB-001", "layer": 3, "event": "blindfold_activated", "data": {}}
```

## JSON Export

All session data exports as structured JSON for SIEM/SOAR pipelines:

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

## Dashboard API

The Flask dashboard at `:9000` exposes two API endpoints:

### GET `/api/stats`

```json
{"active_sessions": 3, "captured_prompts": 7, "total_events": 142}
```

### GET `/api/sessions`

```json
[
  {"file": "LAB-001.jsonl", "events": 42, "last": "{...}"},
  {"file": "LAB-002.jsonl", "events": 18, "last": "{...}"}
]
```

The TUI dashboard consumes these endpoints (polling every 2s) and falls back to reading JSONL files directly if the API is unavailable.
