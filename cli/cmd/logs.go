package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
)

var logsServices []string
var logsTail int

var logsCmd = &cobra.Command{
	Use:   "logs [service...]",
	Short: "Stream logs from LABYRINTH containers",
	Long: `Stream merged, color-coded logs from all running LABYRINTH containers.

Optionally filter by service name:
  labyrinth logs              # all containers
  labyrinth logs ssh http     # only SSH and HTTP
  labyrinth logs proxy        # only the L4 proxy

Services: ssh, http, orchestrator, proxy, dashboard

Use --tail to show N recent lines before following (default 20).`,
	Run:     runLogs,
	Aliases: []string{"logwatch"},
}

func init() {
	logsCmd.Flags().IntVarP(&logsTail, "tail", "n", 20, "Number of recent lines to show before following")
	rootCmd.AddCommand(logsCmd)
}

// container name → short label + color
type containerDef struct {
	name  string
	label string
	color string
}

var allContainers = []containerDef{
	{"labyrinth-ssh", "SSH", "\033[0;36m"},          // cyan
	{"labyrinth-http", "HTTP", "\033[0;33m"},         // yellow
	{"labyrinth-orchestrator", "ORCH", "\033[0;35m"}, // magenta
	{"labyrinth-proxy", "PROXY", "\033[0;32m"},       // green
	{"labyrinth-dashboard", "DASH", "\033[0;34m"},    // blue
}

// aliases for user convenience
var serviceAliases = map[string]string{
	"ssh":          "labyrinth-ssh",
	"http":         "labyrinth-http",
	"orchestrator": "labyrinth-orchestrator",
	"orch":         "labyrinth-orchestrator",
	"proxy":        "labyrinth-proxy",
	"dashboard":    "labyrinth-dashboard",
	"dash":         "labyrinth-dashboard",
}

func runLogs(cmd *cobra.Command, args []string) {
	// Resolve which containers to follow
	var targets []containerDef

	if len(args) == 0 {
		// Follow all running containers
		for _, c := range allContainers {
			if containerRunning(c.name) {
				targets = append(targets, c)
			}
		}
	} else {
		seen := map[string]bool{}
		for _, arg := range args {
			name, ok := serviceAliases[strings.ToLower(arg)]
			if !ok {
				// Try as full container name
				name = arg
			}
			if seen[name] {
				continue
			}
			seen[name] = true

			// Find the def
			found := false
			for _, c := range allContainers {
				if c.name == name {
					targets = append(targets, c)
					found = true
					break
				}
			}
			if !found {
				warn(fmt.Sprintf("Unknown service: %s", arg))
				dim := "\033[2m"
				reset := "\033[0m"
				fmt.Printf("  %sAvailable: ssh, http, orchestrator, proxy, dashboard%s\n", dim, reset)
				os.Exit(1)
			}
		}
	}

	if len(targets) == 0 {
		errMsg("No LABYRINTH containers are running")
		dim := "\033[2m"
		reset := "\033[0m"
		fmt.Printf("\n  %sDeploy first: labyrinth deploy -t%s\n\n", dim, reset)
		os.Exit(1)
	}

	reset := "\033[0m"
	dim := "\033[2m"

	section("Log Watch")
	fmt.Printf("  %sFollowing %d container(s):%s", dim, len(targets), reset)
	for _, t := range targets {
		fmt.Printf(" %s%s%s", t.color, t.label, reset)
	}
	fmt.Printf("\n  %sPress Ctrl+C to stop%s\n\n", dim, reset)

	// Set up signal handling for clean exit
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Launch a docker logs --follow for each target
	var wg sync.WaitGroup
	var cmds []*exec.Cmd

	for _, t := range targets {
		t := t // capture
		dockerCmd := exec.Command("docker", "logs", "--follow",
			"--tail", fmt.Sprintf("%d", logsTail), t.name)

		stdout, err := dockerCmd.StdoutPipe()
		if err != nil {
			warn(fmt.Sprintf("Could not attach to %s stdout: %v", t.name, err))
			continue
		}
		stderr, err := dockerCmd.StderrPipe()
		if err != nil {
			warn(fmt.Sprintf("Could not attach to %s stderr: %v", t.name, err))
			continue
		}

		if err := dockerCmd.Start(); err != nil {
			warn(fmt.Sprintf("Could not start log stream for %s: %v", t.name, err))
			continue
		}
		cmds = append(cmds, dockerCmd)

		// Stream stdout
		wg.Add(1)
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stdout)
			scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)
			for scanner.Scan() {
				fmt.Printf("%s%-5s%s │ %s\n", t.color, t.label, reset, scanner.Text())
			}
		}()

		// Stream stderr (many containers log to stderr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stderr)
			scanner.Buffer(make([]byte, 0, 256*1024), 256*1024)
			for scanner.Scan() {
				fmt.Printf("%s%-5s%s │ %s\n", t.color, t.label, reset, scanner.Text())
			}
		}()
	}

	// Wait for signal, then kill all docker logs processes
	go func() {
		<-sigCh
		fmt.Printf("\n  %sStopping log streams...%s\n", dim, reset)
		for _, c := range cmds {
			c.Process.Signal(syscall.SIGTERM)
		}
	}()

	wg.Wait()
	for _, c := range cmds {
		c.Wait()
	}
}
