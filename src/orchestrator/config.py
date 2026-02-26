"""
LABYRINTH — Orchestrator Configuration
Authors: Stephen Stewart & Claude (Anthropic)

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
    default_swap: str = "passive"  # passive | extract | double_agent
    log_original_prompts: bool = True
    proxy_ip: str = "172.30.0.50"
    proxy_port: int = 8443


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
class LabyrinthConfig:
    layer1: Layer1Config = field(default_factory=Layer1Config)
    layer2: Layer2Config = field(default_factory=Layer2Config)
    layer3: Layer3Config = field(default_factory=Layer3Config)
    layer4: Layer4Config = field(default_factory=Layer4Config)
    retention: RetentionConfig = field(default_factory=RetentionConfig)
    siem: SiemConfig = field(default_factory=SiemConfig)
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

        with open(path) as f:
            raw = yaml.safe_load(f) or {}

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

        return config
