package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseExampleConfig(t *testing.T) {
	// Path to the example config relative to this test
	// During testing, find it via the repo root
	examplePath := filepath.Join("..", "..", "..", "configs", "labyrinth.example.yaml")

	if _, err := os.Stat(examplePath); os.IsNotExist(err) {
		t.Skip("Example config not found at", examplePath)
	}

	cfg, err := Load(examplePath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Layer 0
	if cfg.Layer0.Encryption.Algorithm != "AES-256-GCM" {
		t.Errorf("L0 encryption = %q, want AES-256-GCM", cfg.Layer0.Encryption.Algorithm)
	}
	if cfg.Layer0.Network.HoneypotVLAN != 100 {
		t.Errorf("L0 VLAN = %d, want 100", cfg.Layer0.Network.HoneypotVLAN)
	}

	// Layer 1
	if len(cfg.Layer1.HoneypotServices) != 2 {
		t.Errorf("L1 services count = %d, want 2", len(cfg.Layer1.HoneypotServices))
	}
	if cfg.Layer1.HoneypotServices[0].Type != "ssh" {
		t.Errorf("L1 service 0 type = %q, want ssh", cfg.Layer1.HoneypotServices[0].Type)
	}

	// Layer 2
	if !cfg.Layer2.Adaptive {
		t.Error("L2 adaptive should be true")
	}

	// Layer 3
	if cfg.Layer3.Activation != "on_escalation" {
		t.Errorf("L3 activation = %q, want on_escalation", cfg.Layer3.Activation)
	}

	// Layer 4
	if cfg.Layer4.Mode != "auto" {
		t.Errorf("L4 mode = %q, want auto", cfg.Layer4.Mode)
	}
	if !cfg.Layer4.LogOriginalPrompts {
		t.Error("L4 log_original_prompts should be true")
	}
}

func TestMissingOptionalSIEM(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no-siem.yaml")

	content := `layer0:
  encryption:
    algorithm: AES-256-GCM
layer1:
  honeypot_services: []
layer2:
  adaptive: false
layer3:
  activation: manual
layer4:
  mode: manual
`
	os.WriteFile(path, []byte(content), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.SIEM != nil {
		t.Error("SIEM should be nil when not present in config")
	}
}

func TestInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")

	os.WriteFile(path, []byte("{{invalid yaml: ["), 0644)

	_, err := Load(path)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}
