package docker

import (
	"fmt"
	"net"

	"github.com/DaxxSec/labyrinth/cli/internal/registry"
)

// PortSet holds the three ports needed for a LABYRINTH deployment.
type PortSet struct {
	SSH       int
	HTTP      int
	Dashboard int
}

// TestPorts are the fixed ports used by test deployments.
var TestPorts = PortSet{SSH: 22, HTTP: 8080, Dashboard: 9000}

const (
	prodPortStart = 10000
	prodPortStep  = 10 // gap between allocations for future services
)

// AllocatePorts finds an available set of 3 ports for a production deployment,
// avoiding ports already used by existing environments.
func AllocatePorts(existing []registry.Environment) (PortSet, error) {
	used := make(map[int]bool)
	// Reserve test ports
	used[TestPorts.SSH] = true
	used[TestPorts.HTTP] = true
	used[TestPorts.Dashboard] = true

	for _, env := range existing {
		if env.Ports.SSH != 0 {
			used[env.Ports.SSH] = true
		}
		if env.Ports.HTTP != 0 {
			used[env.Ports.HTTP] = true
		}
		if env.Ports.Dashboard != 0 {
			used[env.Ports.Dashboard] = true
		}
	}

	// Scan from prodPortStart in steps of prodPortStep
	for base := prodPortStart; base < 65000; base += prodPortStep {
		ssh, http, dash := base, base+1, base+2
		if used[ssh] || used[http] || used[dash] {
			continue
		}
		// Verify ports are actually free on the host
		if !portFree(ssh) || !portFree(http) || !portFree(dash) {
			continue
		}
		return PortSet{SSH: ssh, HTTP: http, Dashboard: dash}, nil
	}

	return PortSet{}, fmt.Errorf("no available port range found (scanned %d–65000)", prodPortStart)
}

func portFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

const (
	subnetBase    = 1  // production starts at 172.30.1.0/24 (test uses .0)
	subnetPrefix  = "172.30"
	proxyHostPart = 50 // proxy IP is always .50 in the subnet
)

// AllocateSubnet assigns a unique 172.30.{N}.0/24 subnet for a production deployment.
// Test uses 172.30.0.0/24. Returns subnet CIDR and proxy IP.
func AllocateSubnet(existing []registry.Environment) (subnet, proxyIP string, err error) {
	used := make(map[string]bool)
	used["172.30.0.0/24"] = true // reserved for test

	for _, env := range existing {
		if env.Subnet != "" {
			used[env.Subnet] = true
		}
	}

	for n := subnetBase; n < 255; n++ {
		candidate := fmt.Sprintf("%s.%d.0/24", subnetPrefix, n)
		if !used[candidate] {
			proxy := fmt.Sprintf("%s.%d.%d", subnetPrefix, n, proxyHostPart)
			return candidate, proxy, nil
		}
	}

	return "", "", fmt.Errorf("no available subnet in %s.0.0/16 range", subnetPrefix)
}
