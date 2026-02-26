package docker

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Compose wraps docker compose operations for a specific project.
type Compose struct {
	file    string
	project string
}

// NewCompose creates a Compose wrapper.
func NewCompose(file, project string) *Compose {
	return &Compose{file: file, project: project}
}

// Build runs docker compose build.
func (c *Compose) Build() error {
	cmd := c.command("build")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Up runs docker compose up -d.
func (c *Compose) Up() error {
	cmd := c.command("up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Down runs docker compose down -v.
func (c *Compose) Down() error {
	cmd := c.command("down", "-v")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Ps runs docker compose ps and returns the output.
func (c *Compose) Ps() (string, error) {
	cmd := c.command("ps", "--format", "table {{.Name}}\t{{.Status}}\t{{.Ports}}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// BuildComposeArgs returns the argument list for a compose subcommand.
func BuildComposeArgs(file, project, subcommand string, extra ...string) []string {
	args := []string{"compose", "-f", file, "-p", project, subcommand}
	args = append(args, extra...)
	return args
}

func (c *Compose) command(subcommand string, extra ...string) *exec.Cmd {
	args := BuildComposeArgs(c.file, c.project, subcommand, extra...)
	cmd := exec.Command("docker", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", c.project))
	return cmd
}

// RemoveLabyrinthImages removes Docker images with the labyrinth project label.
func RemoveLabyrinthImages() {
	out, err := exec.Command("docker", "images", "--filter", "label=project=labyrinth", "-q").Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return
	}
	ids := strings.Fields(strings.TrimSpace(string(out)))
	args := append([]string{"rmi"}, ids...)
	exec.Command("docker", args...).Run()
}
