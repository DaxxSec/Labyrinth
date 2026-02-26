package cmd

import (
	"fmt"

	"github.com/ItzDaxxy/labyrinth/cli/internal/registry"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tracked environments",
	Run:   runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) {
	reg := registry.New("")
	envs, err := reg.ListAll()
	if err != nil || len(envs) == 0 {
		warn("No environments registered")
		fmt.Println()
		dim := "\033[2m"
		reset := "\033[0m"
		fmt.Printf("  %sDeploy one with: labyrinth deploy -t [name]%s\n", dim, reset)
		return
	}

	section("Registered Environments")
	bold := "\033[1m"
	reset := "\033[0m"

	fmt.Printf("  %s%-20s %-14s %-18s %s%s\n", bold, "NAME", "TYPE", "MODE", "CREATED", reset)
	fmt.Printf("  %-20s %-14s %-18s %s\n", "────────────────────", "──────────────", "──────────────────", "───────────────────")
	for _, env := range envs {
		fmt.Printf("  %-20s %-14s %-18s %s\n", env.Name, env.Type, env.Mode, env.Created)
	}
	fmt.Println()
}
