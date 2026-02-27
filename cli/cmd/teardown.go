package cmd

import (
	"fmt"
	"os"

	"github.com/DaxxSec/labyrinth/cli/internal/docker"
	"github.com/DaxxSec/labyrinth/cli/internal/registry"
	"github.com/spf13/cobra"
)

var teardownAll bool

var teardownCmd = &cobra.Command{
	Use:   "teardown [name]",
	Short: "Tear down an environment",
	Long:  "Tear down a specific environment or all environments with --all.",
	Run:   runTeardown,
}

func init() {
	teardownCmd.Flags().BoolVar(&teardownAll, "all", false, "Tear down all environments")
	rootCmd.AddCommand(teardownCmd)
}

func runTeardown(cmd *cobra.Command, args []string) {
	reg := registry.New("")

	if teardownAll {
		teardownAllEnvs(reg)
		return
	}

	if len(args) == 0 {
		errMsg("Specify an environment to tear down, or use --all")
		dim := "\033[2m"
		reset := "\033[0m"
		fmt.Printf("  %slabyrinth teardown <name>%s\n", dim, reset)
		fmt.Printf("  %slabyrinth teardown --all%s\n", dim, reset)
		fmt.Println()
		fmt.Printf("  %sSee registered environments: labyrinth list%s\n", dim, reset)
		os.Exit(1)
	}

	envName := args[0]
	env, err := reg.Load(envName)
	if err != nil {
		errMsg(fmt.Sprintf("Environment '%s' not found in registry", envName))
		os.Exit(1)
	}

	section(fmt.Sprintf("Tearing Down: %s", envName))
	teardownSingle(reg, env)
}

func teardownAllEnvs(reg *registry.Registry) {
	section("Tearing Down All Environments")
	envs, err := reg.ListAll()
	if err != nil || len(envs) == 0 {
		warn("No environments registered")
		return
	}
	for _, env := range envs {
		teardownSingle(reg, env)
	}
	info("All environments torn down")
}

func teardownSingle(reg *registry.Registry, env registry.Environment) {
	switch env.Mode {
	case "docker-compose", "docker":
		composeFile := findComposeFile()
		if composeFile != "" {
			comp := docker.NewCompose(composeFile, env.ComposeProject)
			info(fmt.Sprintf("Stopping containers for %s (project: %s)...", env.Name, env.ComposeProject))
			if err := comp.Down(); err != nil {
			warn(fmt.Sprintf("docker compose down returned an error for %s: %v", env.Name, err))
		}
		}
		info(fmt.Sprintf("Removing LABYRINTH images for %s...", env.Name))
		docker.RemoveLabyrinthImages()
	case "k8s":
		warn(fmt.Sprintf("K8s teardown not yet implemented (would run: kubectl delete namespace %s)", env.Namespace))
	case "edge":
		warn(fmt.Sprintf("Edge teardown not yet implemented for %s", env.Name))
	default:
		warn(fmt.Sprintf("Unknown mode '%s' for %s, removing registry entry only", env.Mode, env.Name))
	}

	if err := reg.Remove(env.Name); err != nil {
		warn(fmt.Sprintf("Failed to remove registry entry: %v", err))
	}
}
