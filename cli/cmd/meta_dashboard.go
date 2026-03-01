package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var multiDashPort string

var multiDashboardCmd = &cobra.Command{
	Use:   "multi-dashboard",
	Short: "Launch an aggregated dashboard across all environments",
	Long: `Launch a dashboard that aggregates data from all registered LABYRINTH environments.

Useful when running multiple deployments (e.g. test + production) and you want
a single view across all of them. Runs on port 9999 by default.

Endpoints:
  /api/environments       — list all environments with health status
  /api/aggregate/stats    — combined stats across all environments
  /api/aggregate/sessions — merged session list from all environments`,
	Run: runMultiDashboard,
}

func init() {
	multiDashboardCmd.Flags().StringVar(&multiDashPort, "port", "9999", "Port for the multi-dashboard")
	rootCmd.AddCommand(multiDashboardCmd)
}

func runMultiDashboard(cmd *cobra.Command, args []string) {
	section("Starting Multi-Environment Dashboard")

	script := findMultiDashScript()
	if script == "" {
		errMsg("Cannot find dashboard/meta.py — run from the repo root or install it alongside the binary")
		os.Exit(1)
	}

	info(fmt.Sprintf("Dashboard: http://localhost:%s", multiDashPort))
	info("Endpoints:")
	dim := "\033[2m"
	reset := "\033[0m"
	fmt.Printf("  %s  GET /api/environments       — list all environments%s\n", dim, reset)
	fmt.Printf("  %s  GET /api/aggregate/stats    — aggregated stats%s\n", dim, reset)
	fmt.Printf("  %s  GET /api/aggregate/sessions — merged sessions%s\n", dim, reset)
	fmt.Println()

	pyCmd := exec.Command("python3", script)
	pyCmd.Env = append(os.Environ(), fmt.Sprintf("LABYRINTH_META_PORT=%s", multiDashPort))
	pyCmd.Stdout = os.Stdout
	pyCmd.Stderr = os.Stderr

	if err := pyCmd.Run(); err != nil {
		errMsg(fmt.Sprintf("Multi-dashboard exited: %v", err))
		os.Exit(1)
	}
}

func findMultiDashScript() string {
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
