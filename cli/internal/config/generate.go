package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerateEnvConfig copies the example config to an environment directory,
// patching the proxy IP and subnet values.
func GenerateEnvConfig(examplePath, outputDir, proxyIP, subnet string) (string, error) {
	data, err := os.ReadFile(examplePath)
	if err != nil {
		return "", fmt.Errorf("read example config: %w", err)
	}

	content := string(data)

	// Patch network subnet if a placeholder or default exists
	if subnet != "" {
		content = strings.Replace(content, "honeypot_vlan: 100", fmt.Sprintf("honeypot_vlan: 100  # subnet: %s", subnet), 1)
	}

	// Patch proxy IP as a comment for reference
	if proxyIP != "" {
		content = strings.Replace(content, "egress_proxy: true", fmt.Sprintf("egress_proxy: true  # proxy: %s", proxyIP), 1)
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}

	outPath := filepath.Join(outputDir, "labyrinth.yaml")
	if err := os.WriteFile(outPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("write config: %w", err)
	}

	return outPath, nil
}

// FindExampleConfig searches for configs/labyrinth.example.yaml relative to
// the compose file location or current directory.
func FindExampleConfig(composeFile string) string {
	candidates := []string{}

	// Relative to compose file directory
	if composeFile != "" {
		dir := filepath.Dir(composeFile)
		candidates = append(candidates, filepath.Join(dir, "configs", "labyrinth.example.yaml"))
	}

	// Current directory
	candidates = append(candidates, filepath.Join("configs", "labyrinth.example.yaml"))

	// Walk up from current directory
	dir, err := os.Getwd()
	if err == nil {
		for {
			candidate := filepath.Join(dir, "configs", "labyrinth.example.yaml")
			candidates = append(candidates, candidate)
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return ""
}
