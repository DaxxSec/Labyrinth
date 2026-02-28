package cmd

import (
	"fmt"
	"os"

	"github.com/DaxxSec/labyrinth/cli/internal/docker"
	"github.com/DaxxSec/labyrinth/cli/internal/registry"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status [name]",
	Short: "Show environment status",
	Long:  "Show status of all environments or a specific one.",
	Run:   runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) {
	reg := registry.New("")

	if len(args) > 0 {
		statusSingle(reg, args[0])
		return
	}
	statusAll(reg)
}

func statusSingle(reg *registry.Registry, envName string) {
	env, err := reg.Load(envName)
	if err != nil {
		errMsg(fmt.Sprintf("Environment '%s' not found in registry", envName))
		os.Exit(1)
	}

	section(fmt.Sprintf("Environment: %s", envName))
	bold := "\033[1m"
	reset := "\033[0m"

	fmt.Printf("  %sName:%s     %s\n", bold, reset, env.Name)
	fmt.Printf("  %sType:%s     %s\n", bold, reset, env.Type)
	fmt.Printf("  %sMode:%s     %s\n", bold, reset, env.Mode)
	fmt.Printf("  %sCreated:%s  %s\n", bold, reset, env.Created)

	// Show ports for production deployments
	if env.Ports.SSH != 0 {
		fmt.Printf("  %sPorts:%s    SSH=%d  HTTP=%d  Dashboard=%d\n", bold, reset, env.Ports.SSH, env.Ports.HTTP, env.Ports.Dashboard)
	}
	if env.Subnet != "" {
		fmt.Printf("  %sSubnet:%s   %s\n", bold, reset, env.Subnet)
	}
	if env.DashboardURL != "" {
		fmt.Printf("  %sDashboard:%s %s\n", bold, reset, env.DashboardURL)
	}
	fmt.Println()

	switch env.Mode {
	case "docker-compose", "docker":
		// Use stored compose file for production, fallback to findComposeFile for test
		composeFile := env.ComposeFile
		if composeFile == "" {
			composeFile = findComposeFile()
		}
		if composeFile == "" {
			warn("Cannot find docker-compose.yml for container status")
			return
		}
		comp := docker.NewCompose(composeFile, env.ComposeProject)
		info(fmt.Sprintf("Container status (project: %s):", env.ComposeProject))
		fmt.Println()
		output, err := comp.Ps()
		if err != nil {
			warn("Could not retrieve container status")
		} else {
			fmt.Println(output)
		}
	case "k8s":
		warn(fmt.Sprintf("K8s status not yet implemented (would run: kubectl get pods -n %s)", env.Namespace))
	case "edge":
		warn("Edge status not yet implemented")
	}
	fmt.Println()
}

func statusAll(reg *registry.Registry) {
	envs, err := reg.ListAll()
	if err != nil || len(envs) == 0 {
		section("LABYRINTH Status")
		warn("No environments registered")
		fmt.Println()
		dim := "\033[2m"
		reset := "\033[0m"
		fmt.Printf("  %sDeploy one with: labyrinth deploy -t [name]%s\n", dim, reset)
		return
	}

	section("LABYRINTH Status — All Environments")
	bold := "\033[1m"
	dim := "\033[2m"
	yellow := "\033[1;33m"
	reset := "\033[0m"

	for _, env := range envs {
		typeBadge := env.Type
		if env.Ports.SSH != 0 {
			typeBadge = fmt.Sprintf("%s, ports %d/%d/%d", env.Type, env.Ports.SSH, env.Ports.HTTP, env.Ports.Dashboard)
		}
		fmt.Printf("  %s%s%s  %s(%s/%s, created %s)%s\n", bold, env.Name, reset, dim, typeBadge, env.Mode, env.Created, reset)

		switch env.Mode {
		case "docker-compose", "docker":
			composeFile := env.ComposeFile
			if composeFile == "" {
				composeFile = findComposeFile()
			}
			if composeFile == "" {
				fmt.Printf("    %sNo containers running%s\n", yellow, reset)
				continue
			}
			comp := docker.NewCompose(composeFile, env.ComposeProject)
			output, err := comp.Ps()
			if err != nil || output == "" {
				fmt.Printf("    %sNo containers running%s\n", yellow, reset)
			} else {
				fmt.Println(output)
			}
		case "k8s":
			fmt.Printf("    %sK8s status: not yet implemented%s\n", dim, reset)
		case "edge":
			fmt.Printf("    %sEdge status: not yet implemented%s\n", dim, reset)
		}
		fmt.Println()
	}
}
