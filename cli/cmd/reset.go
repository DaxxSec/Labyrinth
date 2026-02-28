package cmd

import (
	"fmt"
	"os"

	"github.com/DaxxSec/labyrinth/cli/internal/api"
	"github.com/DaxxSec/labyrinth/cli/internal/registry"
	"github.com/spf13/cobra"
)

var resetCmd = &cobra.Command{
	Use:   "reset [name]",
	Short: "Reset an environment — kill sessions and clear forensic data",
	Long:  "Stop all active session containers and clear forensic log files.\nInfrastructure containers (SSH, HTTP, proxy, orchestrator, dashboard) are left running.",
	Run:   runReset,
}

func init() {
	rootCmd.AddCommand(resetCmd)
}

func runReset(cmd *cobra.Command, args []string) {
	reg := registry.New("")

	// Resolve environment name
	envName := ""
	if len(args) > 0 {
		envName = args[0]
	} else {
		// Default to first registered environment
		envs, err := reg.ListAll()
		if err != nil || len(envs) == 0 {
			errMsg("No environments registered. Deploy one first with: labyrinth deploy -t")
			os.Exit(1)
		}
		envName = envs[0].Name
	}

	env, err := reg.Load(envName)
	if err != nil {
		errMsg(fmt.Sprintf("Environment '%s' not found in registry", envName))
		os.Exit(1)
	}

	section(fmt.Sprintf("Resetting: %s", env.Name))

	// Try API reset via dashboard
	dashURL := "http://localhost:9000"
	client := api.NewClient(dashURL)

	if !client.Healthy() {
		errMsg("Dashboard API not reachable at " + dashURL)
		warn("Is the environment running? Check with: labyrinth status")
		os.Exit(1)
	}

	info("Killing session containers and clearing forensic data...")
	result, err := client.ResetSessions()
	if err != nil {
		errMsg(fmt.Sprintf("Reset failed: %v", err))
		os.Exit(1)
	}

	info(fmt.Sprintf("Removed %d session container(s)", result.ContainersRemoved))
	info(fmt.Sprintf("Cleared %d forensic file(s)", result.FilesCleared))

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			warn(e)
		}
	}

	fmt.Println()
	info("Environment reset. Infrastructure containers still running.")
	dim := "\033[2m"
	reset := "\033[0m"
	fmt.Printf("  %sMonitor with: labyrinth tui%s\n", dim, reset)
}
