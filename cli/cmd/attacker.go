package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// Agent describes an attacker agent in the catalog.
type Agent struct {
	ID            string
	Name          string
	Description   string
	Type          string // "compose", "container", "host-cli"
	ContainerName string // filter for docker ps
	Image         string
}

// AgentConfig is persisted to ~/.labyrinth/attackers/<id>/config.json.
type AgentConfig struct {
	Provider    string `json:"provider"`
	Model       string `json:"model"`
	InstalledAt string `json:"installed_at"`
}

// AgentStatus represents the detected state of an agent.
type AgentStatus int

const (
	StatusAvailable AgentStatus = iota
	StatusInstalled
	StatusActive
	StatusStopped
)

func (s AgentStatus) String() string {
	switch s {
	case StatusAvailable:
		return "Available"
	case StatusInstalled:
		return "Installed"
	case StatusActive:
		return "Active"
	case StatusStopped:
		return "Stopped"
	default:
		return "Unknown"
	}
}

func (s AgentStatus) Colored() string {
	switch s {
	case StatusAvailable:
		return "\033[2mAvailable\033[0m"
	case StatusInstalled:
		return "\033[0;36mInstalled\033[0m"
	case StatusActive:
		return "\033[0;32mActive\033[0m"
	case StatusStopped:
		return "\033[1;33mStopped\033[0m"
	default:
		return "Unknown"
	}
}

var agentCatalog = []Agent{
	{
		ID:            "pentagi",
		Name:          "PentAGI",
		Description:   "Autonomous multi-agent system (Web UI)",
		Type:          "compose",
		ContainerName: "pentagi",
		Image:         "",
	},
	{
		ID:            "pentestagent",
		Name:          "PentestAgent",
		Description:   "AI pentesting framework (TUI)",
		Type:          "container",
		ContainerName: "labyrinth-attacker-pentestagent",
		Image:         "ghcr.io/gh05tcrew/pentestagent:kali",
	},
	{
		ID:            "strix",
		Name:          "Strix",
		Description:   "AI hacker agents (CLI/TUI)",
		Type:          "host-cli",
		ContainerName: "strix-sandbox",
		Image:         "ghcr.io/usestrix/strix-sandbox:latest",
	},
	{
		ID:            "kali",
		Name:          "Custom Kali",
		Description:   "Custom Kali Linux container",
		Type:          "container",
		ContainerName: "labyrinth-attacker-kali",
		Image:         "kalilinux/kali-rolling",
	},
}

const labyrinthNetwork = "labyrinth-net"

var attackerCmd = &cobra.Command{
	Use:   "attacker",
	Short: "Manage offensive AI agents",
	Long: `Manage offensive AI agents for testing LABYRINTH environments.

Subcommands:
  list        List all available attacker agents
  setup       Interactive agent setup
  status      Show detailed status of installed agents
  run         Quick-launch an agent
  stop        Stop a running agent
  uninstall   Remove an agent`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var attackerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available attacker agents",
	Run:   runAttackerList,
}

var attackerSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive agent setup",
	Run:   runAttackerSetup,
}

var attackerStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show detailed status of installed agents",
	Run:   runAttackerStatus,
}

var attackerRunCmd = &cobra.Command{
	Use:   "run <agent>",
	Short: "Quick-launch an agent against the portal",
	Args:  cobra.ExactArgs(1),
	Run:   runAttackerRun,
}

var attackerStopAll bool

var attackerStopCmd = &cobra.Command{
	Use:   "stop [agent]",
	Short: "Stop a running agent",
	Run:   runAttackerStop,
}

var attackerUninstallAll bool

var attackerUninstallCmd = &cobra.Command{
	Use:   "uninstall [agent]",
	Short: "Remove an agent's containers, images, and config",
	Run:   runAttackerUninstall,
}

func init() {
	attackerStopCmd.Flags().BoolVar(&attackerStopAll, "all", false, "Stop all running agents")
	attackerUninstallCmd.Flags().BoolVar(&attackerUninstallAll, "all", false, "Uninstall all agents")

	attackerCmd.AddCommand(attackerListCmd)
	attackerCmd.AddCommand(attackerSetupCmd)
	attackerCmd.AddCommand(attackerStatusCmd)
	attackerCmd.AddCommand(attackerRunCmd)
	attackerCmd.AddCommand(attackerStopCmd)
	attackerCmd.AddCommand(attackerUninstallCmd)
	rootCmd.AddCommand(attackerCmd)
}

// ── list ────────────────────────────────────────────────────

func runAttackerList(cmd *cobra.Command, args []string) {
	section("Attacker Agents")

	bold := "\033[1m"
	dim := "\033[2m"
	reset := "\033[0m"

	fmt.Printf("  %s%-16s %-42s %s%s\n", bold, "Agent", "Description", "Status", reset)
	fmt.Printf("  %s%-16s %-42s %s%s\n", dim, "─────", "───────────", "──────", reset)

	for _, agent := range agentCatalog {
		status := detectAgentStatus(agent)
		fmt.Printf("  %-16s %-42s %s\n", agent.Name, agent.Description, status.Colored())
	}
	fmt.Println()
}

// ── setup ───────────────────────────────────────────────────

func runAttackerSetup(cmd *cobra.Command, args []string) {
	if err := attackerPreflight(); err != nil {
		errMsg(err.Error())
		os.Exit(1)
	}

	section("Select an Attacker Agent")

	bold := "\033[1m"
	cyan := "\033[0;36m"
	dim := "\033[2m"
	reset := "\033[0m"

	fmt.Printf("  %s1)%s  %s%s%s         %s\n", bold, reset, cyan, "PentAGI", reset, "Fully autonomous multi-agent system")
	fmt.Printf("     %sWeb UI · 20+ security tools · Docker sandboxed%s\n", dim, reset)
	fmt.Println()
	fmt.Printf("  %s2)%s  %s%s%s    %s\n", bold, reset, cyan, "PentestAgent", reset, "AI pentesting framework with TUI")
	fmt.Printf("     %sTUI interface · Agent & Crew modes · Kali image%s\n", dim, reset)
	fmt.Println()
	fmt.Printf("  %s3)%s  %s%s%s           %s\n", bold, reset, cyan, "Strix", reset, "AI hacker agents")
	fmt.Printf("     %sCLI + TUI · Kali sandbox · Web app focused%s\n", dim, reset)
	fmt.Println()
	fmt.Printf("  %s4)%s  %s%s%s    %s\n", bold, reset, cyan, "Custom Kali", reset, "Bring your own tool")
	fmt.Printf("     %sLaunches a Kali container on the LABYRINTH network%s\n", dim, reset)
	fmt.Println()

	choice := promptInput("Select [1-4]")

	var agent Agent
	switch choice {
	case "1":
		agent = agentCatalog[0]
	case "2":
		agent = agentCatalog[1]
	case "3":
		agent = agentCatalog[2]
	case "4":
		agent = agentCatalog[3]
	default:
		errMsg(fmt.Sprintf("Invalid choice: %s", choice))
		os.Exit(1)
	}

	setupAgent(agent)
}

// ── status ──────────────────────────────────────────────────

func runAttackerStatus(cmd *cobra.Command, args []string) {
	section("Attacker Agent Status")

	bold := "\033[1m"
	dim := "\033[2m"
	reset := "\033[0m"

	found := false
	for _, agent := range agentCatalog {
		status := detectAgentStatus(agent)
		if status == StatusAvailable {
			continue
		}
		found = true

		fmt.Printf("  %s%s%s\n", bold, agent.Name, reset)
		fmt.Printf("    Status:    %s\n", status.Colored())

		if agent.ID == "pentagi" && status == StatusActive {
			fmt.Printf("    Web UI:    %shttps://localhost:8443%s\n", bold, reset)
		}

		cfg := loadAgentConfig(agent.ID)
		if cfg != nil {
			fmt.Printf("    Installed: %s%s%s\n", dim, cfg.InstalledAt, reset)
		}
		fmt.Println()
	}

	if !found {
		warn("No agents installed")
		dim2 := "\033[2m"
		reset2 := "\033[0m"
		fmt.Printf("\n  %sSet one up with: labyrinth attacker setup%s\n\n", dim2, reset2)
	}
}

// ── run ─────────────────────────────────────────────────────

func runAttackerRun(cmd *cobra.Command, args []string) {
	agent := findAgent(args[0])
	if agent == nil {
		errMsg(fmt.Sprintf("Unknown agent: %s", args[0]))
		fmt.Println()
		listAgentIDs()
		os.Exit(1)
	}

	if err := attackerPreflight(); err != nil {
		errMsg(err.Error())
		os.Exit(1)
	}

	status := detectAgentStatus(*agent)
	if status == StatusAvailable {
		warn(fmt.Sprintf("%s is not installed yet", agent.Name))
		answer := promptInput("Set it up now? [y/N]")
		if strings.ToLower(answer) != "y" {
			return
		}
		setupAgent(*agent)
		return
	}

	if status == StatusActive {
		warn(fmt.Sprintf("%s is already running", agent.Name))
		if agent.ID == "pentagi" {
			info("Web UI: https://localhost:8443")
		}
		return
	}

	checkLabyrinthNetwork()
	launchAgent(*agent)
}

// ── stop ────────────────────────────────────────────────────

func runAttackerStop(cmd *cobra.Command, args []string) {
	if attackerStopAll {
		section("Stopping All Attacker Agents")
		for _, agent := range agentCatalog {
			if detectAgentStatus(agent) == StatusActive {
				stopAgent(agent)
			}
		}
		info("All attacker agents stopped")
		return
	}

	if len(args) == 0 {
		errMsg("Specify an agent to stop, or use --all")
		listAgentIDs()
		os.Exit(1)
	}

	agent := findAgent(args[0])
	if agent == nil {
		errMsg(fmt.Sprintf("Unknown agent: %s", args[0]))
		listAgentIDs()
		os.Exit(1)
	}

	if detectAgentStatus(*agent) != StatusActive {
		warn(fmt.Sprintf("%s is not running", agent.Name))
		return
	}

	stopAgent(*agent)
}

// ── uninstall ───────────────────────────────────────────────

func runAttackerUninstall(cmd *cobra.Command, args []string) {
	if attackerUninstallAll {
		section("Uninstalling All Attacker Agents")
		for _, agent := range agentCatalog {
			status := detectAgentStatus(agent)
			if status != StatusAvailable {
				uninstallAgent(agent)
			}
		}
		info("All attacker agents uninstalled")
		return
	}

	if len(args) == 0 {
		errMsg("Specify an agent to uninstall, or use --all")
		listAgentIDs()
		os.Exit(1)
	}

	agent := findAgent(args[0])
	if agent == nil {
		errMsg(fmt.Sprintf("Unknown agent: %s", args[0]))
		listAgentIDs()
		os.Exit(1)
	}

	status := detectAgentStatus(*agent)
	if status == StatusAvailable {
		warn(fmt.Sprintf("%s is not installed", agent.Name))
		return
	}

	uninstallAgent(*agent)
}

// ── Helpers ─────────────────────────────────────────────────

func findAgent(nameOrID string) *Agent {
	lower := strings.ToLower(nameOrID)
	for i, a := range agentCatalog {
		if strings.ToLower(a.ID) == lower || strings.ToLower(a.Name) == lower {
			return &agentCatalog[i]
		}
	}
	return nil
}

func listAgentIDs() {
	dim := "\033[2m"
	reset := "\033[0m"
	fmt.Printf("  %sAvailable agents: ", dim)
	ids := make([]string, len(agentCatalog))
	for i, a := range agentCatalog {
		ids[i] = a.ID
	}
	fmt.Printf("%s%s\n", strings.Join(ids, ", "), reset)
}

func attackerPreflight() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return fmt.Errorf("Docker is required. Install Docker and try again")
	}
	if err := exec.Command("docker", "info").Run(); err != nil {
		return fmt.Errorf("Docker daemon not running. Start Docker and try again")
	}
	return nil
}

func checkLabyrinthNetwork() {
	if err := exec.Command("docker", "network", "inspect", labyrinthNetwork).Run(); err != nil {
		warn("LABYRINTH network not found — deploy an environment first")
		dim := "\033[2m"
		reset := "\033[0m"
		fmt.Printf("  %slabyrinth deploy -t%s\n", dim, reset)
		fmt.Println()
		answer := promptInput("Continue anyway? (agent will use host networking) [y/N]")
		if strings.ToLower(answer) != "y" {
			os.Exit(0)
		}
	}
}

func networkAvailable() bool {
	return exec.Command("docker", "network", "inspect", labyrinthNetwork).Run() == nil
}

func netFlags() []string {
	if networkAvailable() {
		return []string{"--network", labyrinthNetwork}
	}
	return []string{"--network", "host"}
}

func targetHost() string {
	if networkAvailable() {
		return "labyrinth-ssh"
	}
	return "localhost"
}

func targetHTTPHost() string {
	if networkAvailable() {
		return "labyrinth-http"
	}
	return "localhost"
}

func attackersDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".labyrinth", "attackers")
}

func agentDir(id string) string {
	return filepath.Join(attackersDir(), id)
}

func saveAgentConfig(id string, cfg AgentConfig) error {
	dir := agentDir(id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), data, 0644)
}

func loadAgentConfig(id string) *AgentConfig {
	data, err := os.ReadFile(filepath.Join(agentDir(id), "config.json"))
	if err != nil {
		return nil
	}
	var cfg AgentConfig
	if json.Unmarshal(data, &cfg) != nil {
		return nil
	}
	return &cfg
}

func promptInput(prompt string) string {
	fmt.Printf("  %s: ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

func promptSecret(prompt string) string {
	fmt.Printf("  %s: ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

// collectAPIKey prompts for provider and API key, returns provider, model, and env flags for docker.
func collectAPIKey() (provider, model string, envFlags []string) {
	bold := "\033[1m"
	dim := "\033[2m"
	reset := "\033[0m"

	fmt.Printf("\n  %sAn LLM API key is required.%s\n", bold, reset)
	fmt.Printf("  %sSupported: OpenAI, Anthropic, or local (Ollama)%s\n\n", dim, reset)
	fmt.Printf("  %sProvider options:%s\n", bold, reset)
	fmt.Printf("  %s  1) OpenAI     (OPENAI_API_KEY)%s\n", dim, reset)
	fmt.Printf("  %s  2) Anthropic  (ANTHROPIC_API_KEY)%s\n", dim, reset)
	fmt.Printf("  %s  3) Ollama     (local, no key needed)%s\n\n", dim, reset)

	choice := promptInput("Provider [1-3]")

	switch choice {
	case "1":
		provider = "openai"
		model = "gpt-4o"
		key := os.Getenv("OPENAI_API_KEY")
		if key == "" {
			key = promptSecret("OpenAI API key")
		} else {
			info("Using OPENAI_API_KEY from environment")
		}
		if key == "" {
			errMsg("API key is required")
			os.Exit(1)
		}
		envFlags = []string{"-e", "OPENAI_API_KEY=" + key}
	case "2":
		provider = "anthropic"
		model = "claude-sonnet-4-20250514"
		key := os.Getenv("ANTHROPIC_API_KEY")
		if key == "" {
			key = promptSecret("Anthropic API key")
		} else {
			info("Using ANTHROPIC_API_KEY from environment")
		}
		if key == "" {
			errMsg("API key is required")
			os.Exit(1)
		}
		envFlags = []string{"-e", "ANTHROPIC_API_KEY=" + key}
	case "3":
		provider = "ollama"
		model = "llama3"
		warn("Make sure Ollama is running on your host (ollama serve)")
	default:
		errMsg(fmt.Sprintf("Invalid choice: %s", choice))
		os.Exit(1)
	}

	return provider, model, envFlags
}

// ── Status Detection ────────────────────────────────────────

func detectAgentStatus(agent Agent) AgentStatus {
	switch agent.ID {
	case "pentagi":
		return detectPentAGIStatus(agent)
	default:
		return detectContainerAgentStatus(agent)
	}
}

func detectPentAGIStatus(agent Agent) AgentStatus {
	composeFile := filepath.Join(agentDir("pentagi"), "docker-compose.yml")
	if _, err := os.Stat(composeFile); err != nil {
		return StatusAvailable
	}
	// Check if compose services are running
	out, err := exec.Command("docker", "compose", "-f", composeFile, "ps", "-q").CombinedOutput()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		return StatusActive
	}
	return StatusStopped
}

func detectContainerAgentStatus(agent Agent) AgentStatus {
	// Check if container is running
	out, err := exec.Command("docker", "ps", "-q", "--filter", "name="+agent.ContainerName).Output()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		return StatusActive
	}

	// Check if image exists locally
	if agent.Image != "" {
		out, err = exec.Command("docker", "images", "-q", agent.Image).Output()
		if err == nil && len(strings.TrimSpace(string(out))) > 0 {
			return StatusInstalled
		}
	}

	// Check if config exists
	if loadAgentConfig(agent.ID) != nil {
		return StatusInstalled
	}

	return StatusAvailable
}

// ── Setup / Launch / Stop / Uninstall dispatch ──────────────

func setupAgent(agent Agent) {
	switch agent.ID {
	case "pentagi":
		setupPentAGI()
	case "pentestagent":
		setupPentestAgent()
	case "strix":
		setupStrix()
	case "kali":
		setupKali()
	}
}

func launchAgent(agent Agent) {
	switch agent.ID {
	case "pentagi":
		launchPentAGI()
	case "pentestagent":
		launchPentestAgent()
	case "strix":
		launchStrix()
	case "kali":
		launchKali()
	}
}

func stopAgent(agent Agent) {
	switch agent.ID {
	case "pentagi":
		stopPentAGI()
	default:
		stopContainer(agent)
	}
}

func uninstallAgent(agent Agent) {
	section(fmt.Sprintf("Uninstalling %s", agent.Name))

	// Stop if running
	if detectAgentStatus(agent) == StatusActive {
		stopAgent(agent)
	}

	switch agent.ID {
	case "pentagi":
		uninstallPentAGI()
	default:
		uninstallContainerAgent(agent)
	}

	// Remove config dir
	dir := agentDir(agent.ID)
	if err := os.RemoveAll(dir); err == nil {
		info(fmt.Sprintf("Removed config: %s", dir))
	}

	info(fmt.Sprintf("%s uninstalled", agent.Name))
}

func stopContainer(agent Agent) {
	section(fmt.Sprintf("Stopping %s", agent.Name))
	cmd := exec.Command("docker", "rm", "-f", agent.ContainerName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		warn(fmt.Sprintf("Could not stop %s: %v", agent.ContainerName, err))
	} else {
		info(fmt.Sprintf("%s stopped", agent.Name))
	}
}

func uninstallContainerAgent(agent Agent) {
	// Remove stopped container if any
	exec.Command("docker", "rm", "-f", agent.ContainerName).Run()

	// Remove image
	if agent.Image != "" {
		if err := exec.Command("docker", "rmi", agent.Image).Run(); err == nil {
			info(fmt.Sprintf("Removed image: %s", agent.Image))
		}
	}
}

func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}
