package docker

import (
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
)

var requiredPorts = []int{2222, 8080, 9000}

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

	// Port availability
	for _, port := range requiredPorts {
		if !PortAvailable(port) {
			return fmt.Errorf("Port %d is already in use. Free it and try again", port)
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

// PortAvailable checks if a TCP port is free to bind.
func PortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}
