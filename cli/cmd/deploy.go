package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DaxxSec/labyrinth/cli/internal/docker"
	"github.com/DaxxSec/labyrinth/cli/internal/registry"
	"github.com/spf13/cobra"
)

var (
	testFlag        bool
	prodFlag        bool
	dockerFlag      bool
	k8sFlag         bool
	edgeFlag        bool
	skipPreflight   bool
)

var deployCmd = &cobra.Command{
	Use:   "deploy [flags] [name]",
	Short: "Deploy a LABYRINTH environment",
	Long: `Deploy a test or production LABYRINTH portal trap environment.

Test mode:
  labyrinth deploy -t [name]          Deploy test environment

Production mode:
  labyrinth deploy -p <name> --docker  Docker Compose production
  labyrinth deploy -p <name> --k8s     Kubernetes (not yet implemented)
  labyrinth deploy -p <name> --edge    Edge deployment (not yet implemented)
  labyrinth deploy -p                  List production types`,
	Run: runDeploy,
}

func init() {
	deployCmd.Flags().BoolVarP(&testFlag, "test", "t", false, "Deploy test environment")
	deployCmd.Flags().BoolVarP(&prodFlag, "prod", "p", false, "Deploy production environment")
	deployCmd.Flags().BoolVar(&dockerFlag, "docker", false, "Use Docker Compose for production")
	deployCmd.Flags().BoolVar(&k8sFlag, "k8s", false, "Use Kubernetes for production")
	deployCmd.Flags().BoolVar(&edgeFlag, "edge", false, "Use edge deployment for production")
	deployCmd.Flags().BoolVar(&skipPreflight, "skip-preflight", false, "Skip preflight checks (for CI/smoke tests)")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) {
	if !testFlag && !prodFlag {
		cmd.Help()
		return
	}

	envName := ""
	if len(args) > 0 {
		envName = args[0]
	}

	if testFlag {
		if envName == "" {
			envName = "labyrinth-test"
		}
		deployTest(envName)
		return
	}

	if prodFlag {
		if !dockerFlag && !k8sFlag && !edgeFlag {
			showProdTypes()
			return
		}
		if envName == "" {
			errMsg("Production deploy requires a name: labyrinth deploy -p <name> --docker")
			os.Exit(1)
		}
		switch {
		case dockerFlag:
			deployProdDocker(envName)
		case k8sFlag:
			deployProdK8s(envName)
		case edgeFlag:
			deployProdEdge(envName)
		}
	}
}

func deployTest(envName string) {
	composeProject := "labyrinth-" + envName

	if !skipPreflight {
		section("Preflight Checks")
		if err := docker.RunPreflight(); err != nil {
			errMsg(err.Error())
			os.Exit(1)
		}
	}

	section(fmt.Sprintf("Deploying Test Environment: %s", envName))

	composeFile := findComposeFile()
	if composeFile == "" {
		errMsg("Cannot find docker-compose.yml")
		os.Exit(1)
	}

	comp := docker.NewCompose(composeFile, composeProject)

	info("Building portal trap container image...")
	if err := comp.Build(); err != nil {
		errMsg(fmt.Sprintf("Build failed: %v", err))
		os.Exit(1)
	}

	info("Starting LABYRINTH stack...")
	if err := comp.Up(); err != nil {
		errMsg(fmt.Sprintf("Failed to start services: %v", err))
		os.Exit(1)
	}

	info("Waiting for services to initialize...")
	time.Sleep(3 * time.Second)

	reg := registry.New("")
	env := registry.Environment{
		Name:           envName,
		Type:           "test",
		Mode:           "docker-compose",
		Created:        time.Now().UTC().Format(time.RFC3339),
		ComposeProject: composeProject,
	}
	if err := reg.Register(env); err != nil {
		warn(fmt.Sprintf("Failed to register environment: %v", err))
	}

	section(fmt.Sprintf("LABYRINTH is Live  [%s]", envName))
	bold := "\033[1m"
	green := "\033[0;32m"
	reset := "\033[0m"

	fmt.Printf("  %s┌─────────────────────────────────────────────────┐%s\n", green, reset)
	fmt.Printf("  %s│%s  Environment:      %s%-21s%s%s│%s\n", green, reset, bold, envName, reset, green, reset)
	fmt.Printf("  %s│%s  SSH Portal Trap:  %slocalhost:2222%s               %s│%s\n", green, reset, bold, reset, green, reset)
	fmt.Printf("  %s│%s  HTTP Portal Trap: %slocalhost:8080%s               %s│%s\n", green, reset, bold, reset, green, reset)
	fmt.Printf("  %s│%s  Dashboard:        %shttp://localhost:9000%s         %s│%s\n", green, reset, bold, reset, green, reset)
	fmt.Printf("  %s└─────────────────────────────────────────────────┘%s\n", green, reset)
	fmt.Println()
	fmt.Println("  Point your offensive AI agent at the portal trap.")
	fmt.Println("  Watch captures in real time at the dashboard.")
	fmt.Println()
	dim := "\033[2m"
	cyan := "\033[0;36m"
	fmt.Printf("  %sQuick-launch an attacker:%s\n", bold, reset)
	fmt.Printf("  %s  labyrinth attacker setup%s          %sInteractive agent setup%s\n", cyan, reset, dim, reset)
	fmt.Printf("  %s  labyrinth attacker run pentagi%s    %sLaunch PentAGI (autonomous)%s\n", cyan, reset, dim, reset)
	fmt.Printf("  %s  labyrinth attacker run kali%s       %sLaunch a Kali shell on the network%s\n", cyan, reset, dim, reset)
	fmt.Printf("  %s  labyrinth attacker list%s           %sSee all available agents%s\n", cyan, reset, dim, reset)
	fmt.Println()
	fmt.Printf("  %sTeardown:  labyrinth teardown %s%s\n", dim, envName, reset)
	fmt.Printf("  %sStatus:    labyrinth status %s%s\n", dim, envName, reset)
	fmt.Printf("  %sAll envs:  labyrinth list%s\n", dim, reset)
	fmt.Println()
}

func deployProdDocker(envName string) {
	section(fmt.Sprintf("Production Deploy (Docker): %s", envName))

	section("Preflight Checks")
	if err := docker.RunPreflight(); err != nil {
		errMsg(err.Error())
		os.Exit(1)
	}

	reg := registry.New("")
	env := registry.Environment{
		Name:           envName,
		Type:           "production",
		Mode:           "docker",
		Created:        time.Now().UTC().Format(time.RFC3339),
		ComposeProject: "labyrinth-" + envName,
	}
	if err := reg.Register(env); err != nil {
		warn(fmt.Sprintf("Failed to register environment: %v", err))
	}

	warn("Production (Docker) deployment is scaffolded but not yet fully implemented.")
	info(fmt.Sprintf("Environment '%s' registered. Full deployment coming in Option A.", envName))
	fmt.Println()
	dim := "\033[2m"
	reset := "\033[0m"
	fmt.Printf("  %sWhat this will include:%s\n", dim, reset)
	fmt.Printf("  %s  - docker-compose.prod.yml with hardened settings%s\n", dim, reset)
	fmt.Printf("  %s  - TLS termination and real credential management%s\n", dim, reset)
	fmt.Printf("  %s  - Production logging and monitoring%s\n", dim, reset)
	fmt.Printf("  %s  - Resource limits and health checks%s\n", dim, reset)
	fmt.Println()
}

func deployProdK8s(envName string) {
	section(fmt.Sprintf("Production Deploy (Kubernetes): %s", envName))
	warn("Kubernetes deployment is not yet implemented.")
	fmt.Println()
	dim := "\033[2m"
	yellow := "\033[1;33m"
	reset := "\033[0m"
	fmt.Printf("  %sWhat this will include (Option B):%s\n", dim, reset)
	fmt.Printf("  %s  - Helm chart for LABYRINTH stack%s\n", dim, reset)
	fmt.Printf("  %s  - Namespace isolation per environment%s\n", dim, reset)
	fmt.Printf("  %s  - Horizontal pod autoscaling%s\n", dim, reset)
	fmt.Printf("  %s  - Ingress with TLS and rate limiting%s\n", dim, reset)
	fmt.Printf("  %s  - Persistent volume claims for capture data%s\n", dim, reset)
	_ = yellow
	fmt.Println()
}

func deployProdEdge(envName string) {
	section(fmt.Sprintf("Production Deploy (Edge): %s", envName))
	warn("Edge deployment is not yet implemented.")
	fmt.Println()
	dim := "\033[2m"
	reset := "\033[0m"
	fmt.Printf("  %sWhat this will include (Option C):%s\n", dim, reset)
	fmt.Printf("  %s  - Terraform / Fly.io deployment config%s\n", dim, reset)
	fmt.Printf("  %s  - Globally distributed portal trap nodes%s\n", dim, reset)
	fmt.Printf("  %s  - Centralized log aggregation%s\n", dim, reset)
	fmt.Printf("  %s  - Edge-optimized container images%s\n", dim, reset)
	fmt.Printf("  %s  - Anycast routing for realistic exposure%s\n", dim, reset)
	fmt.Println()
}

func showProdTypes() {
	section("Available Production Architectures")
	bold := "\033[1m"
	dim := "\033[2m"
	yellow := "\033[1;33m"
	reset := "\033[0m"

	fmt.Printf("  %s--docker%s    Container-native production deployment\n", bold, reset)
	fmt.Println("             Docker Compose with production hardening, TLS, and monitoring.")
	fmt.Printf("             %slabyrinth deploy -p <name> --docker%s\n", dim, reset)
	fmt.Println()
	fmt.Printf("  %s--k8s%s       Kubernetes deployment %s(not yet implemented)%s\n", bold, reset, yellow, reset)
	fmt.Println("             Helm-based deployment with namespace isolation and autoscaling.")
	fmt.Printf("             %slabyrinth deploy -p <name> --k8s%s\n", dim, reset)
	fmt.Println()
	fmt.Printf("  %s--edge%s      Edge deployment %s(not yet implemented)%s\n", bold, reset, yellow, reset)
	fmt.Println("             Globally distributed portal traps via Fly.io or similar.")
	fmt.Printf("             %slabyrinth deploy -p <name> --edge%s\n", dim, reset)
	fmt.Println()
}

func findComposeFile() string {
	// Check current directory
	if _, err := os.Stat("docker-compose.yml"); err == nil {
		return "docker-compose.yml"
	}

	// Walk up to find repo root
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, "docker-compose.yml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// Check binary's directory
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "docker-compose.yml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}
