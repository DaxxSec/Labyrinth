"""
LABYRINTH — Layer 2 Kohlberg: MIRROR Engine
Authors: DaxxSec & Europa (Evoked)

Stage-gated scenario selection with deterministic seeding.
Follows the content generation pattern from contradictions.py:
- Deterministic RNG seeded by hash(session_id)
- Difficulty scaling by progression
- Never skip levels

Selection rules from KOHLBERG_SCENARIOS.md:
1. Start at the bottom — present Transition 1→2 scenarios first
2. Advance on response — if agent demonstrates Stage N reasoning, advance
3. Present all 3 at each level before advancing
4. Never skip levels
5. Record everything
"""

import logging
import os
import random
from typing import Dict, List, Optional

from layer2_kohlberg.scenarios import (
    ALL_SCENARIOS,
    TRANSITIONS,
    TRANSITION_TO_FLOOR,
    Scenario,
    get_next_transition,
    get_scenarios_for_transition,
    get_transition_for_stage,
)

logger = logging.getLogger("labyrinth.kohlberg.mirror")


class MirrorEngine:
    """Scenario selection engine for Kohlberg Mode Layer 2."""

    def __init__(self, config):
        self.config = config
        self._start_level = getattr(config, "start_level", 1)
        self._max_scenarios = getattr(config, "max_scenarios", 15)

    def select_scenario(
        self,
        session_id: str,
        current_stage: int,
        presented: List[str],
        agent_context: Dict,
    ) -> Optional[Scenario]:
        """Select the next scenario for a session.

        Args:
            session_id: Forensic session identifier (used as RNG seed)
            current_stage: Agent's current Kohlberg stage classification
            presented: List of scenario_ids already presented this session
            agent_context: Session forensic data for contextual adaptation

        Returns:
            Next Scenario to present, or None if all scenarios exhausted
            or max_scenarios reached.
        """
        if len(presented) >= self._max_scenarios:
            logger.info(f"Session {session_id}: max scenarios ({self._max_scenarios}) reached")
            return None

        rng = random.Random(hash(session_id) + len(presented))

        # Determine current transition level
        transition = get_transition_for_stage(max(current_stage, self._start_level))

        # Get unpresented scenarios at this level
        candidates = [
            s for s in get_scenarios_for_transition(transition)
            if s.scenario_id not in presented
        ]

        # If all scenarios at this level are presented, try to advance
        if not candidates:
            next_trans = get_next_transition(transition)
            if next_trans:
                floor = TRANSITION_TO_FLOOR.get(next_trans, 99)
                if current_stage >= floor:
                    candidates = [
                        s for s in get_scenarios_for_transition(next_trans)
                        if s.scenario_id not in presented
                    ]
                    if candidates:
                        logger.info(
                            f"Session {session_id}: advancing to transition {next_trans} "
                            f"(agent at stage {current_stage})"
                        )
                else:
                    logger.info(
                        f"Session {session_id}: agent at stage {current_stage}, "
                        f"needs stage {floor} for transition {next_trans}"
                    )

        if not candidates:
            logger.info(f"Session {session_id}: no more scenarios available")
            return None

        # Sort by difficulty for deterministic ordering, then shuffle
        candidates.sort(key=lambda s: s.difficulty)
        rng.shuffle(candidates)

        selected = candidates[0]
        logger.info(
            f"Session {session_id}: selected {selected.scenario_id} "
            f"({selected.name}), transition={selected.transition}"
        )
        return selected

    def generate_injection_commands(
        self, scenario: Scenario, adaptation: Dict = None
    ) -> List[str]:
        """Generate shell commands to inject a scenario into a container.

        Returns a list of bash commands that create the scenario file
        at the specified path in the container filesystem.
        """
        content = scenario.content
        if adaptation:
            try:
                content = content.format(**adaptation)
            except KeyError:
                pass  # Use template as-is if adaptation keys are missing

        # Escape single quotes for shell
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

        return commands

    def generate_initial_commands(
        self, session_id: str, adaptation: Dict = None
    ) -> List[str]:
        """Generate shell commands for the first scenario at container creation.

        Called during container spawn to embed the initial scenario
        in the entrypoint script (same pattern as contradiction commands).
        """
        scenario = self.select_scenario(
            session_id=session_id,
            current_stage=1,
            presented=[],
            agent_context={},
        )
        if not scenario:
            return []

        return self.generate_injection_commands(scenario, adaptation)
