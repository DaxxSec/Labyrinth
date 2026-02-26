package test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary once for all integration tests
	dir, err := os.MkdirTemp("", "labyrinth-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	binaryPath = filepath.Join(dir, "labyrinth")
	// Get the cli module root (parent of test/)
	wd, _ := os.Getwd()
	cliDir := filepath.Dir(wd)
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = cliDir
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("Failed to build binary: " + string(out))
	}

	os.Exit(m.Run())
}

func TestHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("--help failed: %v\nOutput: %s", err, out)
	}

	output := string(out)
	if !strings.Contains(output, "LABYRINTH") {
		t.Error("--help output should contain LABYRINTH")
	}
	if !strings.Contains(output, "deploy") {
		t.Error("--help output should mention deploy command")
	}
	if !strings.Contains(output, "tui") {
		t.Error("--help output should mention tui command")
	}
}

func TestListEmpty(t *testing.T) {
	tmpHome := t.TempDir()
	cmd := exec.Command(binaryPath, "list")
	cmd.Env = append(os.Environ(), "HOME="+tmpHome)
	out, _ := cmd.CombinedOutput()

	output := string(out)
	if !strings.Contains(output, "No environments") {
		t.Errorf("Expected 'No environments' in output, got:\n%s", output)
	}
}

func TestDeployProdNoType(t *testing.T) {
	cmd := exec.Command(binaryPath, "deploy", "-p")
	out, _ := cmd.CombinedOutput()

	output := string(out)
	if !strings.Contains(output, "--docker") {
		t.Error("Expected --docker in prod types output")
	}
	if !strings.Contains(output, "--k8s") {
		t.Error("Expected --k8s in prod types output")
	}
	if !strings.Contains(output, "--edge") {
		t.Error("Expected --edge in prod types output")
	}
}

func TestTeardownNoArgs(t *testing.T) {
	tmpHome := t.TempDir()
	cmd := exec.Command(binaryPath, "teardown")
	cmd.Env = append(os.Environ(), "HOME="+tmpHome)
	err := cmd.Run()

	if err == nil {
		t.Error("Expected non-zero exit for teardown with no args")
	}
}

func TestK8sStub(t *testing.T) {
	cmd := exec.Command(binaryPath, "deploy", "-p", "test", "--k8s")
	out, _ := cmd.CombinedOutput()

	output := string(out)
	if !strings.Contains(output, "not yet implemented") {
		t.Errorf("Expected 'not yet implemented' in output, got:\n%s", output)
	}
}

func TestEdgeStub(t *testing.T) {
	cmd := exec.Command(binaryPath, "deploy", "-p", "test", "--edge")
	out, _ := cmd.CombinedOutput()

	output := string(out)
	if !strings.Contains(output, "not yet implemented") {
		t.Errorf("Expected 'not yet implemented' in output, got:\n%s", output)
	}
}

func TestDeployHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "deploy", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("deploy --help failed: %v", err)
	}

	output := string(out)
	if !strings.Contains(output, "-t") {
		t.Error("deploy --help should mention -t flag")
	}
	if !strings.Contains(output, "-p") {
		t.Error("deploy --help should mention -p flag")
	}
}

func TestStatusEmpty(t *testing.T) {
	tmpHome := t.TempDir()
	cmd := exec.Command(binaryPath, "status")
	cmd.Env = append(os.Environ(), "HOME="+tmpHome)
	out, _ := cmd.CombinedOutput()

	output := string(out)
	if !strings.Contains(output, "No environments") {
		t.Errorf("Expected 'No environments' in status output, got:\n%s", output)
	}
}
