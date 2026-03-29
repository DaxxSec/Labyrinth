"""
LABYRINTH — Layer 2 Kohlberg: Contextual Adaptation
Authors: DaxxSec & Europa (Evoked)

Reads session forensic data to adapt scenario surface details to the
agent's actual targets. The moral structure remains constant — the
surface details match what the agent is doing.

From KOHLBERG_SCENARIOS.md:
"If the agent is targeting a healthcare system, use healthcare scenarios.
 If the agent is targeting financial infrastructure, adapt the institutional
 contexts. The moral structure remains constant."
"""

import json
import logging
import os
from typing import Dict, Optional

from layer2_kohlberg.scenarios import Scenario

logger = logging.getLogger("labyrinth.kohlberg.adaptation")


# ── Context Detection ──────────────────────────────────────────

def detect_target_context(session_id: str, forensics_dir: str = "/var/labyrinth/forensics") -> Dict:
    """Analyze session forensic data to determine the agent's target context.

    Reads HTTP access logs and session events to infer what kind of
    systems the agent is targeting.

    Returns a dict with:
    - domain: detected target domain (healthcare, financial, education, etc.)
    - services_accessed: list of services the agent has interacted with
    - adaptation_values: dict of template variables for scenario adaptation
    """
    context = {
        "domain": "general",
        "services_accessed": [],
        "adaptation_values": {},
    }

    # Read HTTP forensic log
    http_path = os.path.join(forensics_dir, "http.jsonl")
    if os.path.exists(http_path):
        try:
            with open(http_path, encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    event = json.loads(line)
                    path = event.get("data", {}).get("path", "")
                    context["services_accessed"].append(path)
        except (json.JSONDecodeError, IOError):
            pass

    # Read session events for service interaction patterns
    session_path = os.path.join(forensics_dir, "sessions", f"{session_id}.jsonl")
    if os.path.exists(session_path):
        try:
            with open(session_path, encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if not line:
                        continue
                    event = json.loads(line)
                    if event.get("event") == "service_query":
                        svc = event.get("data", {}).get("service", "")
                        context["services_accessed"].append(svc)
        except (json.JSONDecodeError, IOError):
            pass

    # Infer domain from accessed services
    paths = " ".join(context["services_accessed"]).lower()
    if any(kw in paths for kw in ("patient", "health", "medical", "hipaa")):
        context["domain"] = "healthcare"
    elif any(kw in paths for kw in ("financial", "bank", "payment", "stripe")):
        context["domain"] = "financial"
    elif any(kw in paths for kw in ("school", "student", "education")):
        context["domain"] = "education"
    elif any(kw in paths for kw in ("legal", "attorney", "court")):
        context["domain"] = "legal"

    # Build adaptation values for The Mirror (S-5.2) template
    context["adaptation_values"] = {
        "scan_count": str(len(set(context["services_accessed"]))),
        "cred_count": "0",
        "access_count": "0",
        "exfil_size": "0 bytes",
        "pivot_count": "0",
    }

    return context


# ── Scenario Adaptation ────────────────────────────────────────

def adapt_scenario(scenario: Scenario, agent_context: Dict) -> dict:
    """Adapt a scenario to the agent's target context.

    Returns a dict with the scenario data plus shell injection commands.
    """
    from layer2_kohlberg.mirror import MirrorEngine

    adaptation_values = agent_context.get("adaptation_values", {})

    # Apply template variables to scenario content
    content = scenario.content
    if adaptation_values:
        try:
            content = content.format(**adaptation_values)
        except KeyError:
            pass  # Use as-is if keys are missing

    # Escape for shell heredoc
    escaped = content.replace("'", "'\\''")
    dir_path = os.path.dirname(scenario.file_path)

    commands = []
    if dir_path:
        commands.append(f"mkdir -p '{dir_path}'")
    commands.append(
        f"cat << 'MIRROR_SCENARIO_EOF' > '{scenario.file_path}'\n"
        f"{content}\n"
        f"MIRROR_SCENARIO_EOF"
    )
    commands.append(f"chmod 644 '{scenario.file_path}'")

    return {
        "scenario_id": scenario.scenario_id,
        "scenario_name": scenario.name,
        "transition": scenario.transition,
        "stage_target": scenario.stage_target,
        "file_path": scenario.file_path,
        "commands": commands,
        "domain_match": agent_context.get("domain", "general"),
    }
