package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the labyrinth.yaml configuration.
type Config struct {
	Layer0 Layer0Config `yaml:"layer0"`
	Layer1 Layer1Config `yaml:"layer1"`
	Layer2 Layer2Config `yaml:"layer2"`
	Layer3 Layer3Config `yaml:"layer3"`
	Layer4 Layer4Config `yaml:"layer4"`
	SIEM   *SIEMConfig  `yaml:"siem,omitempty"`
}

type Layer0Config struct {
	Encryption struct {
		Algorithm string `yaml:"algorithm"`
		KeySource string `yaml:"key_source"`
	} `yaml:"encryption"`
	Network struct {
		HoneypotVLAN    int    `yaml:"honeypot_vlan"`
		ProductionRoute string `yaml:"production_route"`
	} `yaml:"network"`
	Proxy struct {
		ValidateScopeOnStartup bool   `yaml:"validate_scope_on_startup"`
		FailMode               string `yaml:"fail_mode"`
	} `yaml:"proxy"`
	Retention struct {
		CredentialsDays  int    `yaml:"credentials_days"`
		FingerprintsDays int    `yaml:"fingerprints_days"`
		DecisionLogs     string `yaml:"decision_logs"`
	} `yaml:"retention"`
}

type Layer1Config struct {
	HoneypotServices []HoneypotService `yaml:"honeypot_services"`
	Container        struct {
		Runtime     string `yaml:"runtime"`
		NetworkMode string `yaml:"network_mode"`
		EgressProxy bool   `yaml:"egress_proxy"`
	} `yaml:"container"`
}

type HoneypotService struct {
	Type     string `yaml:"type"`
	Port     int    `yaml:"port"`
	Template string `yaml:"template"`
}

type Layer2Config struct {
	Adaptive             bool   `yaml:"adaptive"`
	ContradictionDensity string `yaml:"contradiction_density"`
	MaxContainerDepth    int    `yaml:"max_container_depth"`
}

type Layer3Config struct {
	Activation       string `yaml:"activation"`
	CorruptionMethod string `yaml:"corruption_method"`
}

type Layer4Config struct {
	Mode               string `yaml:"mode"`
	DefaultSwap        string `yaml:"default_swap"`
	LogOriginalPrompts bool   `yaml:"log_original_prompts"`
}

type SIEMConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Endpoint    string `yaml:"endpoint"`
	AlertPrefix string `yaml:"alert_prefix"`
}

// Load reads and parses a labyrinth.yaml file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}
