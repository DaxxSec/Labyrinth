package docker

import (
	"net"
	"testing"
)

func TestPortAvailable(t *testing.T) {
	// Use a high ephemeral port that should be free
	if !PortAvailable(59123) {
		t.Skip("Port 59123 unexpectedly in use, skipping")
	}
}

func TestPortInUse(t *testing.T) {
	// Start a listener on a known port
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to start listener: %v", err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	if PortAvailable(port) {
		t.Errorf("Port %d should be in use but reported as available", port)
	}
}

func TestParseDockerVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Docker version 24.0.7, build afdd53b", "24.0.7"},
		{"Docker version 20.10.21, build baeda1f", "20.10.21"},
		{"Docker version 27.5.1, build 9f9e405", "27.5.1"},
	}

	for _, tt := range tests {
		result := ParseDockerVersion(tt.input)
		if result != tt.expected {
			t.Errorf("ParseDockerVersion(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
