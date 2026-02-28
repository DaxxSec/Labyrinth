package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var metaPort string

var metaDashboardCmd = &cobra.Command{
	Use:   "meta-dashboard",
	Short: "Launch the central meta-dashboard",
	Long: `Launch a meta-dashboard that aggregates data from all LABYRINTH environments.

The meta-dashboard runs on port 9999 (configurable with --port) and provides:
  - /api/environments     — lists all registered environments with health status
  - /api/aggregate/stats  — aggregated stats across all environments
  - /api/aggregate/sessions — merged sessions from all environments`,
	Run: runMetaDashboard,
}

func init() {
	metaDashboardCmd.Flags().StringVar(&metaPort, "port", "9999", "Port for the meta-dashboard")
	rootCmd.AddCommand(metaDashboardCmd)
}

func runMetaDashboard(cmd *cobra.Command, args []string) {
	section("Starting Meta-Dashboard")

	// Find meta.py
	metaScript := findMetaScript()
	if metaScript == "" {
		errMsg("Cannot find dashboard/meta.py")
		os.Exit(1)
	}

	info(fmt.Sprintf("Meta-dashboard: http://localhost:%s", metaPort))
	info("Endpoints:")
	dim := "\033[2m"
	reset := "\033[0m"
	fmt.Printf("  %s  GET /api/environments       — list all environments%s\n", dim, reset)
	fmt.Printf("  %s  GET /api/aggregate/stats    — aggregated stats%s\n", dim, reset)
	fmt.Printf("  %s  GET /api/aggregate/sessions — merged sessions%s\n", dim, reset)
	fmt.Println()

	// Run the meta-dashboard
	pyCmd := exec.Command("python3", metaScript)
	pyCmd.Env = append(os.Environ(), fmt.Sprintf("LABYRINTH_META_PORT=%s", metaPort))
	pyCmd.Stdout = os.Stdout
	pyCmd.Stderr = os.Stderr

	if err := pyCmd.Run(); err != nil {
		errMsg(fmt.Sprintf("Meta-dashboard exited: %v", err))
		os.Exit(1)
	}
}

func findMetaScript() string {
	candidates := []string{
		"dashboard/meta.py",
	}

	// Walk up to find repo root
	dir, err := os.Getwd()
	if err == nil {
		for {
			candidate := filepath.Join(dir, "dashboard", "meta.py")
			candidates = append(candidates, candidate)
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// Check binary's directory
	exe, err := os.Executable()
	if err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "dashboard", "meta.py"))
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return ""
}
