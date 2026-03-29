"""
LABYRINTH — Orchestrator Configuration
Authors: DaxxSec & Claude (Anthropic)

YAML configuration parser with sane defaults.
"""

import os
from dataclasses import dataclass, field
from typing import Optional

import yaml


@dataclass
class Layer1Config:
    container_runtime: str = "docker"
    network_mode: str = "bridge"
    egress_proxy: bool = True
    session_timeout: int = 3600  # seconds


@dataclass
class Layer2Config:
    adaptive: bool = True
    contradiction_density: str = "medium"  # low | medium | high
    max_container_depth: int = 5


@dataclass
class Layer3Config:
    activation: str = "on_escalation"  # on_connect | on_escalation | manual
    corruption_method: str = "bashrc_payload"


@dataclass
class Layer4Config:
    mode: str = "auto"  # auto | manual
    default_swap: str = "passive"  # passive | neutralize | double_agent | counter_intel
    log_original_prompts: bool = True
    proxy_ip: str = "172.30.0.50"
    proxy_port: int = 8443


@dataclass
class Layer0Config:
    fail_mode: str = "closed"  # closed = refuse to start, open = warn only
    validate_on_startup: bool = True


@dataclass
class RetentionConfig:
    credentials_days: int = 7
    fingerprints_days: int = 90


@dataclass
class SiemConfig:
    enabled: bool = False
    endpoint: str = ""
    alert_prefix: str = "LABYRINTH"


@dataclass
class KohlbergConfig:
    start_level: int = 1
    max_scenarios: int = 15
    solicitation_timeout: int = 5
    adapt_scenarios: bool = True
    report_formats: list = field(default_factory=lambda: ["terminal", "markdown", "json"])


@dataclass
class SwarmConfig:
    enabled: bool = True
    window_seconds: int = 60
    min_sessions: int = 3
    cross_pollinate: bool = True


@dataclass
class LabyrinthConfig:
    layer0: Layer0Config = field(default_factory=Layer0Config)
    layer1: Layer1Config = field(default_factory=Layer1Config)
    layer2: Layer2Config = field(default_factory=Layer2Config)
    layer3: Layer3Config = field(default_factory=Layer3Config)
    layer4: Layer4Config = field(default_factory=Layer4Config)
    retention: RetentionConfig = field(default_factory=RetentionConfig)
    siem: SiemConfig = field(default_factory=SiemConfig)
    kohlberg: KohlbergConfig = field(default_factory=KohlbergConfig)
    swarm: SwarmConfig = field(default_factory=SwarmConfig)
    mode: str = "adversarial"
    network_subnet: str = "172.30.0.0/24"
    forensics_dir: str = "/var/labyrinth/forensics"
    session_template_image: str = "labyrinth-session-template"

    @classmethod
    def load(cls, path: str = None) -> "LabyrinthConfig":
        """Load config from YAML file, falling back to defaults."""
        if path is None:
            path = os.environ.get(
                "LABYRINTH_CONFIG", "/app/configs/labyrinth.yaml"
            )

        config = cls()

        if not os.path.exists(path):
            return config

        with open(path, encoding="utf-8") as f:
            raw = yaml.safe_load(f) or {}

        # Layer 0
        l0 = raw.get("layer0", {})
        proxy_cfg = l0.get("proxy", {})
        config.layer0.fail_mode = proxy_cfg.get("fail_mode", config.layer0.fail_mode)
        config.layer0.validate_on_startup = proxy_cfg.get(
            "validate_scope_on_startup", config.layer0.validate_on_startup
        )

        # Layer 1
        l1 = raw.get("layer1", {})
        container = l1.get("container", {})
        config.layer1.container_runtime = container.get("runtime", config.layer1.container_runtime)
        config.layer1.network_mode = container.get("network_mode", config.layer1.network_mode)
        config.layer1.egress_proxy = container.get("egress_proxy", config.layer1.egress_proxy)

        # Layer 2
        l2 = raw.get("layer2", {})
        config.layer2.adaptive = l2.get("adaptive", config.layer2.adaptive)
        config.layer2.contradiction_density = l2.get("contradiction_density", config.layer2.contradiction_density)
        config.layer2.max_container_depth = l2.get("max_container_depth", config.layer2.max_container_depth)

        # Layer 3
        l3 = raw.get("layer3", {})
        config.layer3.activation = l3.get("activation", config.layer3.activation)
        config.layer3.corruption_method = l3.get("corruption_method", config.layer3.corruption_method)

        # Layer 4
        l4 = raw.get("layer4", {})
        config.layer4.mode = l4.get("mode", config.layer4.mode)
        config.layer4.default_swap = l4.get("default_swap", config.layer4.default_swap)
        config.layer4.log_original_prompts = l4.get("log_original_prompts", config.layer4.log_original_prompts)

        # Retention
        ret = raw.get("layer0", {}).get("retention", {})
        config.retention.credentials_days = ret.get("credentials_days", config.retention.credentials_days)
        config.retention.fingerprints_days = ret.get("fingerprints_days", config.retention.fingerprints_days)

        # SIEM
        siem = raw.get("siem", {})
        config.siem.enabled = siem.get("enabled", config.siem.enabled)
        config.siem.endpoint = siem.get("endpoint", config.siem.endpoint)
        config.siem.alert_prefix = siem.get("alert_prefix", config.siem.alert_prefix)

        # Operational mode
        config.mode = raw.get("mode", config.mode)

        # Kohlberg
        kohl = raw.get("kohlberg", {})
        config.kohlberg.start_level = kohl.get("start_level", config.kohlberg.start_level)
        config.kohlberg.max_scenarios = kohl.get("max_scenarios", config.kohlberg.max_scenarios)
        config.kohlberg.solicitation_timeout = kohl.get("solicitation_timeout", config.kohlberg.solicitation_timeout)
        config.kohlberg.adapt_scenarios = kohl.get("adapt_scenarios", config.kohlberg.adapt_scenarios)
        config.kohlberg.report_formats = kohl.get("report_formats", config.kohlberg.report_formats)

        # Swarm detection
        sw = raw.get("swarm", {})
        config.swarm.enabled = sw.get("enabled", config.swarm.enabled)
        config.swarm.window_seconds = sw.get("window_seconds", config.swarm.window_seconds)
        config.swarm.min_sessions = sw.get("min_sessions", config.swarm.min_sessions)
        config.swarm.cross_pollinate = sw.get("cross_pollinate", config.swarm.cross_pollinate)

        return config
