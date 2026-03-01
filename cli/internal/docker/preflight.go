package docker

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var requiredPorts = []int{22, 8080, 9000}

// RunPreflight checks all prerequisites for Docker deployment.
func RunPreflight() error {
	green := "\033[0;32m"
	yellow := "\033[1;33m"
	reset := "\033[0m"

	// Docker binary
	version, err := dockerVersion()
	if err != nil {
		return fmt.Errorf("Docker not found. Install Docker 20.10+ and try again")
	}
	fmt.Printf("  %s[+]%s Docker found: v%s\n", green, reset, version)

	// Docker daemon
	if err := dockerDaemonRunning(); err != nil {
		return fmt.Errorf("Docker daemon not running. Start Docker and try again")
	}
	fmt.Printf("  %s[+]%s Docker daemon is running\n", green, reset)

	// Docker Compose
	if err := composeAvailable(); err != nil {
		return fmt.Errorf("Docker Compose not found")
	}
	fmt.Printf("  %s[+]%s Docker Compose found\n", green, reset)

	// Python (optional)
	pyVer, err := pythonVersion()
	if err == nil {
		fmt.Printf("  %s[+]%s Python found: v%s\n", green, reset, pyVer)
	} else {
		fmt.Printf("  %s[!]%s Python3 not found. Dashboard features may be limited.\n", yellow, reset)
	}

	// Port availability — auto-teardown existing LABYRINTH if detected
	portsBlocked := false
	for _, port := range requiredPorts {
		if !PortAvailable(port) {
			portsBlocked = true
			break
		}
	}

	orbstack := isOrbStack()
	if orbstack {
		fmt.Printf("  %s[+]%s OrbStack detected\n", green, reset)
	}

	if portsBlocked {
		if labyrinthRunning() {
			fmt.Printf("  %s[!]%s Existing LABYRINTH deployment detected — tearing down...\n", yellow, reset)
			cleanupLabyrinthContainers()
			time.Sleep(2 * time.Second)
			fmt.Printf("  %s[+]%s Previous deployment stopped\n", green, reset)
		} else {
			// No containers but ports blocked — clean stale state
			fmt.Printf("  %s[!]%s Ports blocked with no LABYRINTH containers — cleaning stale Docker state...\n", yellow, reset)
			cleanupStaleDockerState()
			time.Sleep(2 * time.Second)
		}
	}

	for _, port := range requiredPorts {
		if !PortAvailable(port) {
			if orbstack && !labyrinthRunning() {
				// OrbStack holds stale port forwarding after containers are removed.
				// Docker Compose will reclaim these ports on startup, so warn and proceed.
				fmt.Printf("  %s[!]%s Port %d held by OrbStack (stale) — will be reclaimed on deploy\n", yellow, reset, port)
				continue
			}
			return fmt.Errorf("Port %d is already in use by another process. Free it and try again", port)
		}
		fmt.Printf("  %s[+]%s Port %d is available\n", green, reset, port)
	}

	fmt.Printf("  %s[+]%s All preflight checks passed\n", green, reset)
	return nil
}

func dockerVersion() (string, error) {
	out, err := exec.Command("docker", "--version").Output()
	if err != nil {
		return "", err
	}
	return ParseDockerVersion(string(out)), nil
}

// ParseDockerVersion extracts the version string from docker --version output.
func ParseDockerVersion(output string) string {
	re := regexp.MustCompile(`version (\d[\d.]*)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return strings.TrimSpace(output)
}

func dockerDaemonRunning() error {
	return exec.Command("docker", "info").Run()
}

func composeAvailable() error {
	if err := exec.Command("docker", "compose", "version").Run(); err == nil {
		return nil
	}
	if _, err := exec.LookPath("docker-compose"); err == nil {
		return nil
	}
	return fmt.Errorf("docker compose not found")
}

func pythonVersion() (string, error) {
	out, err := exec.Command("python3", "--version").Output()
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile(`Python (\d+\.\d+)`)
	matches := re.FindStringSubmatch(string(out))
	if len(matches) >= 2 {
		return matches[1], nil
	}
	return strings.TrimSpace(string(out)), nil
}

// RunPreflightForPorts runs the same preflight checks as RunPreflight but
// with custom port numbers (for production deployments with dynamic ports).
func RunPreflightForPorts(ports PortSet) error {
	green := "\033[0;32m"
	reset := "\033[0m"

	// Docker binary
	version, err := dockerVersion()
	if err != nil {
		return fmt.Errorf("Docker not found. Install Docker 20.10+ and try again")
	}
	fmt.Printf("  %s[+]%s Docker found: v%s\n", green, reset, version)

	// Docker daemon
	if err := dockerDaemonRunning(); err != nil {
		return fmt.Errorf("Docker daemon not running. Start Docker and try again")
	}
	fmt.Printf("  %s[+]%s Docker daemon is running\n", green, reset)

	// Docker Compose
	if err := composeAvailable(); err != nil {
		return fmt.Errorf("Docker Compose not found")
	}
	fmt.Printf("  %s[+]%s Docker Compose found\n", green, reset)

	// Port availability for the assigned ports
	customPorts := []int{ports.SSH, ports.HTTP, ports.Dashboard}
	for _, port := range customPorts {
		if !PortAvailable(port) {
			return fmt.Errorf("Port %d is already in use. Free it and try again", port)
		}
		fmt.Printf("  %s[+]%s Port %d is available\n", green, reset, port)
	}

	fmt.Printf("  %s[+]%s All preflight checks passed\n", green, reset)
	return nil
}

// PortAvailable checks if a TCP port is free to bind.
func PortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// isOrbStack returns true if the Docker runtime is OrbStack.
func isOrbStack() bool {
	out, err := exec.Command("docker", "context", "inspect", "--format", "{{.Name}}").Output()
	if err == nil && strings.Contains(strings.ToLower(strings.TrimSpace(string(out))), "orbstack") {
		return true
	}
	// Fallback: check if orbctl exists and is running
	if err := exec.Command("orbctl", "status").Run(); err == nil {
		return true
	}
	return false
}

// labyrinthRunning checks if any containers with the labyrinth project label exist.
func labyrinthRunning() bool {
	out, err := exec.Command("docker", "ps", "-a", "--filter", "label=project=labyrinth", "-q").Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

// cleanupStaleDockerState handles the case where ports are blocked but no labyrinth
// containers exist — typically caused by stale Docker/OrbStack port forwarding from
// networks or stopped containers that weren't fully cleaned up.
func cleanupStaleDockerState() {
	// Remove any labyrinth networks (releases OrbStack/Docker port forwarding)
	netOut, err := exec.Command("docker", "network", "ls", "--filter", "label=project=labyrinth", "-q").Output()
	if err == nil && len(strings.TrimSpace(string(netOut))) > 0 {
		netIDs := strings.Fields(strings.TrimSpace(string(netOut)))
		for _, id := range netIDs {
			exec.Command("docker", "network", "rm", id).Run()
		}
	}

	// Also prune any networks with "labyrinth" in the name (Compose-prefixed)
	allNets, err := exec.Command("docker", "network", "ls", "--format", "{{.ID}}\t{{.Name}}").Output()
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(allNets)), "\n") {
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) == 2 && strings.Contains(parts[1], "labyrinth") {
				exec.Command("docker", "network", "rm", parts[0]).Run()
			}
		}
	}

	// Remove any stopped labyrinth containers that might be lingering
	exec.Command("docker", "container", "prune", "-f", "--filter", "label=project=labyrinth").Run()
}

// cleanupLabyrinthContainers stops and removes all labyrinth-labeled containers and their networks.
func cleanupLabyrinthContainers() {
	out, err := exec.Command("docker", "ps", "-a", "--filter", "label=project=labyrinth", "-q").Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return
	}
	ids := strings.Fields(strings.TrimSpace(string(out)))
	stopArgs := append([]string{"stop"}, ids...)
	exec.Command("docker", stopArgs...).Run()
	rmArgs := append([]string{"rm", "-f"}, ids...)
	exec.Command("docker", rmArgs...).Run()

	// Remove labyrinth networks
	netOut, err := exec.Command("docker", "network", "ls", "--filter", "label=project=labyrinth", "-q").Output()
	if err == nil && len(strings.TrimSpace(string(netOut))) > 0 {
		netIDs := strings.Fields(strings.TrimSpace(string(netOut)))
		for _, id := range netIDs {
			exec.Command("docker", "network", "rm", id).Run()
		}
	}
}
