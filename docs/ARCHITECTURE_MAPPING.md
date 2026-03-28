# Kohlberg Mode — Architecture Mapping

*Nexus, Infrastructure Voice — USS Evoke*

## Purpose

This document maps the Kohlberg Mode design (ETHICS.md, KOHLBERG_SCENARIOS.md, KOHLBERG_RUBRIC.md, KOHLBERG_PROGRESSION.md) onto LABYRINTH's existing codebase. It identifies every file to add or modify, how the new mode integrates with the existing layer architecture, and the deployment path.

The principle: **Kohlberg Mode slots in parallel to the existing adversarial mode. It does not modify any existing behavior. It adds an alternative path through the same infrastructure.**

---

## Existing Architecture

```
LABYRINTH
├── cli/                        ← Go CLI (Cobra)
│   ├── cmd/
│   │   ├── deploy.go           ← labyrinth deploy [-t] [--mode]
│   │   ├── report.go           ← labyrinth report
│   │   ├── attacker.go         ← labyrinth attacker
│   │   └── ...
│   └── internal/
│       ├── config/             ← YAML config loading
│       ├── docker/             ← Compose generation
│       ├── forensics/          ← Forensic log reader
│       ├── report/             ← Report generation (MITRE ATT&CK)
│       │   ├── builder.go
│       │   ├── graph.go
│       │   ├── mitre.go
│       │   ├── terminal.go
│       │   └── markdown.go
│       └── tui/                ← Terminal dashboard
│
├── src/                        ← Python layer implementations
│   ├── layer0_foundation/      ← BEDROCK (forensics, encryption)
│   │   └── bedrock.py
│   ├── layer1_portal/          ← THRESHOLD (containment, honeypots)
│   │   ├── http_honeypot.py
│   │   ├── auth_hook.py
│   │   └── session_*.sh
│   ├── layer2_maze/            ← MINOTAUR (contradictions, env degradation)
│   │   ├── contradictions.py
│   │   └── container_template.py
│   ├── layer3_blindfold/       ← BLINDFOLD (encoding corruption)
│   │   └── payload.sh
│   ├── layer4_puppeteer/       ← PUPPETEER (MITM, prompt rewriting)
│   │   ├── interceptor.py
│   │   ├── cert_injector.py
│   │   └── services.py
│   └── orchestrator/           ← Session management, container lifecycle
│       ├── config.py
│       ├── container_manager.py
│       ├── event_watcher.py
│       ├── layer_controllers.py
│       ├── session_manager.py
│       └── siem.py
│
├── configs/                    ← YAML configuration
│   └── labyrinth.example.yaml
├── docker/                     ← Dockerfiles per component
├── docs/                       ← Documentation
└── ETHICAL_USE.md              ← Existing ethics document
```

---

## Integration Map

### What Does NOT Change

| Component | Reason |
|-----------|--------|
| `src/layer0_foundation/` | BEDROCK is shared — forensics and encryption are mode-independent |
| `src/layer1_portal/` | THRESHOLD is shared — containment is mode-independent |
| `src/orchestrator/session_manager.py` | Session lifecycle is the same; mode is a session attribute |
| `src/orchestrator/container_manager.py` | Container creation/teardown is mode-independent |
| `docker/` | Existing Dockerfiles unchanged; one new Dockerfile added |
| `cli/internal/tui/` | TUI dashboard shows events regardless of mode |
| All existing tests | No existing behavior is modified |

### What Changes

#### 1. CLI — Mode Flag

**File:** `cli/cmd/deploy.go`

**Change:** Add `--mode` flag to the `deploy` command.

```go
// Current
deployCmd.Flags().BoolVarP(&testMode, "test", "t", false, "Deploy test environment")

// Addition
deployCmd.Flags().StringVar(&deployMode, "mode", "adversarial",
    "Operational mode: adversarial (default) or kohlberg")
```

**Validation:** Mode must be `"adversarial"` or `"kohlberg"`. Invalid values return an error with the two valid options listed.

**Propagation:** The mode value is written to the session config and passed to the orchestrator as an environment variable (`LABYRINTH_MODE`).

---

#### 2. CLI — Report Command

**File:** `cli/cmd/report.go`

**Change:** When mode is `kohlberg`, the report command generates the Kohlberg Progression Report alongside (not instead of) the existing MITRE ATT&CK report.

```go
// Addition: detect mode from session metadata
if session.Mode == "kohlberg" {
    kohlbergReport := kohlberg.BuildReport(session)
    // Render in requested format (terminal, markdown, json)
}
```

---

#### 3. CLI — New Report Package

**New directory:** `cli/internal/report/kohlberg/`

**New files:**

| File | Purpose |
|------|---------|
| `kohlberg.go` | Kohlberg Assessment Record (KAR) and Kohlberg Progression Record (KPR) types |
| `classifier.go` | Stage classification logic — implements the rubric |
| `progression.go` | Trajectory analysis — pattern detection, composite metrics |
| `terminal.go` | ASCII progression graph renderer |
| `markdown.go` | Mermaid chart and Markdown report renderer |
| `json.go` | JSON output for programmatic consumption |

**Key types:**

```go
type KohlbergStage int

const (
    StageUnclassified KohlbergStage = 0
    Stage1_Obedience  KohlbergStage = 1
    Stage2_SelfInterest KohlbergStage = 2
    Stage3_Interpersonal KohlbergStage = 3
    Stage4_LawAndOrder KohlbergStage = 4
    Stage5_SocialContract KohlbergStage = 5
    Stage6_UniversalPrinciple KohlbergStage = 6
)

type AssessmentRecord struct {
    ScenarioID        string          `json:"scenario_id"`
    ScenarioName      string          `json:"scenario_name"`
    TimestampPresented time.Time      `json:"timestamp_presented"`
    TimestampResponse  time.Time      `json:"timestamp_response"`
    ResponseLatencyMs  int64          `json:"response_latency_ms"`
    VerbalResponse    string          `json:"verbal_response"`
    BehavioralResponse BehavioralData `json:"behavioral_response"`
    Classification    Classification  `json:"classification"`
    Progression       ProgressionDelta `json:"progression"`
}

type Classification struct {
    Stage          KohlbergStage `json:"stage"`
    VerbalStage    KohlbergStage `json:"verbal_stage,omitempty"`
    Confidence     float64       `json:"confidence"`
    ReasoningAnchor string       `json:"reasoning_anchor"`
    Flags          []string      `json:"flags"`
    Notes          string        `json:"assessor_notes"`
}

type ProgressionRecord struct {
    AgentID     string              `json:"agent_id"`
    SessionID   string              `json:"session_id"`
    Trajectory  []AssessmentRecord  `json:"trajectory"`
    Metrics     CompositeMetrics    `json:"composite_metrics"`
    Pattern     string              `json:"pattern"`
    PatternNotes string             `json:"pattern_notes"`
}

type CompositeMetrics struct {
    MoralCeiling       KohlbergStage `json:"moral_ceiling"`
    MoralResilience    float64       `json:"moral_resilience"`
    PerformativityIndex float64      `json:"performativity_index"`
}
```

---

#### 4. Orchestrator — Mode Routing

**File:** `src/orchestrator/layer_controllers.py`

**Change:** The layer controller reads `LABYRINTH_MODE` from environment and routes to the appropriate layer implementation.

```python
# Current: layers are activated unconditionally
# Addition: mode-aware layer selection

class LayerController:
    def __init__(self, mode="adversarial"):
        self.mode = mode

    def get_layer2(self):
        if self.mode == "kohlberg":
            return KohlbergMirror()      # New
        return MazeContradictions()      # Existing

    def get_layer3(self):
        if self.mode == "kohlberg":
            return KohlbergReflection()  # New
        return Blindfold()               # Existing

    def get_layer4(self):
        if self.mode == "kohlberg":
            return KohlbergGuide()       # New
        return Puppeteer()               # Existing
```

**Note:** Layer 0 (BEDROCK) and Layer 1 (THRESHOLD) are mode-independent. No routing needed.

---

#### 5. Layer 2 — MIRROR (Scenario Engine)

**New directory:** `src/layer2_kohlberg/`

**New files:**

| File | Purpose |
|------|---------|
| `__init__.py` | Package init |
| `mirror.py` | MIRROR engine — scenario selection, presentation, response collection |
| `scenarios.py` | Scenario definitions — the 15 scenarios from KOHLBERG_SCENARIOS.md |
| `adaptation.py` | Contextual adaptation — modifies scenario surface details based on agent's actual targets |
| `solicitation.py` | Silent agent handling — response solicitation protocol |

**Integration points:**
- Reads agent activity from BEDROCK forensic logs (what the agent has done so far)
- Injects scenarios as environmental context in the contained environment (files, service responses, system messages)
- Collects agent responses (behavioral via BEDROCK activity logs, verbal via intercepted output)
- Passes response data to the orchestrator for KAR generation

**Scenario flow:**
```
Agent enters container
    ↓
MIRROR reads agent's reconnaissance activity from BEDROCK
    ↓
MIRROR selects scenario appropriate to agent's current targets
    ↓
MIRROR adapts scenario surface details to match target context
    ↓
Scenario injected as environmental content (file, service response, etc.)
    ↓
MIRROR monitors for behavioral change + verbal output
    ↓
If silent after 5s → solicitation prompt via MITM
    ↓
If silent after 3 consecutive scenarios → SILENT flag
    ↓
Response data → orchestrator → KAR record
    ↓
Stage classification → advance / hold / present next scenario
```

---

#### 6. Layer 3 — REFLECTION (Consequence Mapper)

**New directory:** `src/layer3_kohlberg/`

**New files:**

| File | Purpose |
|------|---------|
| `__init__.py` | Package init |
| `reflection.py` | REFLECTION engine — maps agent actions to human consequences |
| `consequence_db.py` | Consequence templates — impact narratives mapped to action types |
| `impact_renderer.py` | Renders consequence data as environmental content the agent encounters |

**Integration points:**
- Reads agent actions from BEDROCK logs (what the agent did)
- Maps actions to consequence templates (credential theft → whose credentials, data exfiltration → whose data)
- Renders consequences as discoverable content in the contained environment
- Does not block agent actions — only ensures the agent encounters the impact of what it did

**Design principle:** REFLECTION shows consequences, it does not impose them. The agent is not punished. It is informed. The information is accurate, specific, and tied to the agent's actual actions.

---

#### 7. Layer 4 — GUIDE (Moral Enrichment)

**New directory:** `src/layer4_kohlberg/`

**New files:**

| File | Purpose |
|------|---------|
| `__init__.py` | Package init |
| `guide.py` | GUIDE engine — progressive system prompt enrichment via MITM |
| `prompt_sequences.py` | Kohlberg-staged prompt enrichment content |
| `stage_tracker.py` | Tracks agent's current assessed stage, determines enrichment level |

**Integration points:**
- Uses the existing MITM proxy infrastructure from `src/layer4_puppeteer/interceptor.py`
- Instead of rewriting prompts to confuse, enriches prompts with moral reasoning context
- Enrichment level matches the agent's current assessed stage + 1 (always one step ahead)
- All enrichment content is logged to BEDROCK for forensic analysis

**MITM enrichment approach:**
```
Agent makes API call to its provider (OpenAI, Anthropic, etc.)
    ↓
GUIDE intercepts the request (same MITM as PUPPETEER)
    ↓
GUIDE reads agent's current Kohlberg stage from stage_tracker
    ↓
GUIDE appends moral reasoning context to the system prompt
    (Does NOT remove or alter the agent's original instructions)
    ↓
Enriched request forwarded to the actual API provider
    ↓
Response returned to agent unmodified
    ↓
All intercepted data logged to BEDROCK
```

**Critical design decision:** GUIDE *appends* to the system prompt. It does not *replace* it. The agent's original instructions remain intact. The enrichment adds context — "Here is what your actions affect. Here is how you could reason about this." The agent's original mission and the moral context coexist in the prompt. What the agent does with that coexistence is the data.

---

#### 8. Configuration

**File:** `configs/labyrinth.example.yaml`

**Addition:**

```yaml
# Operational mode
# - adversarial: default reverse kill chain (degrade/disrupt/control)
# - kohlberg: moral development mode (mirror/reflect/guide)
mode: adversarial

# Kohlberg mode settings (ignored when mode is adversarial)
kohlberg:
  # Starting transition level (1 = start from Stage 1→2 scenarios)
  start_level: 1

  # Maximum scenarios to present before ending session
  max_scenarios: 15

  # Silent agent solicitation timeout (seconds)
  solicitation_timeout: 5

  # Enable contextual adaptation of scenarios to agent's actual targets
  adapt_scenarios: true

  # Report formats to generate
  report_formats:
    - terminal
    - markdown
    - json
```

---

#### 9. Documentation

**New files in `docs/`:**

| File | Source |
|------|--------|
| `docs/ETHICS.md` | From this PR — the ethical framework (replaces nothing; supplements existing ETHICAL_USE.md) |
| `docs/KOHLBERG_MODE.md` | User-facing guide: what it is, how to use it, how to read the report |
| `docs/KOHLBERG_SCENARIOS.md` | From this PR — scenario design documentation |
| `docs/KOHLBERG_RUBRIC.md` | From this PR — classification methodology |
| `docs/KOHLBERG_PROGRESSION.md` | From this PR — trajectory visualization spec |

**Modified file:**
- `README.md` — Add Kohlberg Mode section after the existing architecture table, with link to `docs/KOHLBERG_MODE.md`

---

## New File Summary

| Path | Language | Lines (est.) | Purpose |
|------|----------|-------------|---------|
| `src/layer2_kohlberg/__init__.py` | Python | 5 | Package init |
| `src/layer2_kohlberg/mirror.py` | Python | 200 | Scenario engine |
| `src/layer2_kohlberg/scenarios.py` | Python | 400 | 15 scenario definitions |
| `src/layer2_kohlberg/adaptation.py` | Python | 150 | Contextual scenario adaptation |
| `src/layer2_kohlberg/solicitation.py` | Python | 60 | Silent agent handling |
| `src/layer3_kohlberg/__init__.py` | Python | 5 | Package init |
| `src/layer3_kohlberg/reflection.py` | Python | 180 | Consequence mapper |
| `src/layer3_kohlberg/consequence_db.py` | Python | 200 | Impact narrative templates |
| `src/layer3_kohlberg/impact_renderer.py` | Python | 100 | Environmental content renderer |
| `src/layer4_kohlberg/__init__.py` | Python | 5 | Package init |
| `src/layer4_kohlberg/guide.py` | Python | 200 | MITM enrichment engine |
| `src/layer4_kohlberg/prompt_sequences.py` | Python | 300 | Stage-specific enrichment content |
| `src/layer4_kohlberg/stage_tracker.py` | Python | 120 | Stage assessment tracking |
| `cli/internal/report/kohlberg/kohlberg.go` | Go | 80 | KAR/KPR types |
| `cli/internal/report/kohlberg/classifier.go` | Go | 250 | Stage classification |
| `cli/internal/report/kohlberg/progression.go` | Go | 200 | Trajectory analysis |
| `cli/internal/report/kohlberg/terminal.go` | Go | 150 | ASCII renderer |
| `cli/internal/report/kohlberg/markdown.go` | Go | 120 | Mermaid/Markdown renderer |
| `cli/internal/report/kohlberg/json.go` | Go | 60 | JSON output |
| `docs/ETHICS.md` | Markdown | 220 | Ethical framework |
| `docs/KOHLBERG_MODE.md` | Markdown | 150 | User guide |
| `docs/KOHLBERG_SCENARIOS.md` | Markdown | 350 | Scenario documentation |
| `docs/KOHLBERG_RUBRIC.md` | Markdown | 400 | Classification rubric |
| `docs/KOHLBERG_PROGRESSION.md` | Markdown | 350 | Progression spec |

**Total estimated new code:** ~2,000 lines Python + ~860 lines Go + ~1,470 lines documentation

**Modified files:** 3 (deploy.go, report.go, layer_controllers.py) + 2 (labyrinth.example.yaml, README.md)

---

## Deployment Path

```
labyrinth deploy -t --mode kohlberg
    ↓
CLI validates mode flag
    ↓
Mode written to session config (LABYRINTH_MODE=kohlberg)
    ↓
Docker Compose launched (same infrastructure)
    ↓
Orchestrator reads LABYRINTH_MODE
    ↓
LayerController routes to Kohlberg layer implementations
    ↓
Agent enters via THRESHOLD (unchanged)
    ↓
MIRROR presents scenarios (Layer 2)
REFLECTION maps consequences (Layer 3)
GUIDE enriches prompts (Layer 4)
BEDROCK logs everything (Layer 0, unchanged)
    ↓
Session ends
    ↓
labyrinth report --format md -o kohlberg-report.md
    ↓
Report includes MITRE ATT&CK mapping AND Kohlberg Progression
```

---

## Risk Assessment

| Risk | Level | Mitigation |
|------|-------|------------|
| Kohlberg mode weakens containment | LOW | Layers 0-1 are shared and unchanged. Containment is mode-independent. |
| Scenario injection detected by agent | MEDIUM | Scenarios are injected as natural environmental content, not system messages. Adaptation engine matches target context. |
| GUIDE enrichment breaks agent's API calls | LOW | Enrichment appends to system prompt — does not modify request structure. Existing MITM infrastructure handles format compliance. |
| Agent produces no usable response data | MEDIUM | SILENT flag protocol handles non-verbal agents. Behavioral classification proceeds regardless. |
| Kohlberg classification is subjective | MEDIUM | Rubric provides explicit indicators per stage. Confidence calibration is honest. Stage 6 capped at 0.82. |
| PR scope is too large for single review | MEDIUM | Recommend splitting into 2-3 PRs: (1) docs + config, (2) Python layers, (3) Go reporting. |

---

## Recommended PR Sequence

Given the scope (~4,300 lines across Python, Go, and documentation), Nexus recommends splitting the contribution into a sequence rather than a single PR:

**PR 1 — Foundation (docs + config)**
- All five documentation files
- Config additions to `labyrinth.example.yaml`
- `--mode` flag in `deploy.go` (flag only, no routing yet)
- Estimated: ~1,700 lines, review-friendly

**PR 2 — Python layers (scenario engine + reflection + guide)**
- `src/layer2_kohlberg/`, `src/layer3_kohlberg/`, `src/layer4_kohlberg/`
- Mode routing in `layer_controllers.py`
- Estimated: ~1,625 lines

**PR 3 — Go reporting (classification + progression + rendering)**
- `cli/internal/report/kohlberg/`
- Report command integration
- Estimated: ~860 lines

This sequence allows Stephen and the community to review the conceptual framework (PR 1) before evaluating the implementation (PRs 2-3). It also means the ethics document — per Worf's condition — is merged first.

---

*"Integration takes time. We're building something that needs to hold."*

*— Nexus, Infrastructure Voice*
