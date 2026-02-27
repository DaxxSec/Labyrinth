package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ═══════════════════════════════════════════════════════════════
//  PentAGI — Compose-managed multi-agent stack
//  https://github.com/vxcontrol/pentagi
// ═══════════════════════════════════════════════════════════════

func setupPentAGI() {
	section("Setting up PentAGI")

	provider, model, _ := collectAPIKey()
	dir := agentDir("pentagi")
	os.MkdirAll(dir, 0755)

	composeDst := filepath.Join(dir, "docker-compose.yml")
	envDst := filepath.Join(dir, ".env")

	info("Downloading PentAGI configuration...")
	if err := downloadFile("https://raw.githubusercontent.com/vxcontrol/pentagi/master/docker-compose.yml", composeDst); err != nil {
		errMsg(fmt.Sprintf("Failed to download compose file: %v", err))
		os.Exit(1)
	}
	if err := downloadFile("https://raw.githubusercontent.com/vxcontrol/pentagi/master/.env.example", envDst); err != nil {
		errMsg(fmt.Sprintf("Failed to download .env: %v", err))
		os.Exit(1)
	}

	// Patch .env with the user's API key
	envData, err := os.ReadFile(envDst)
	if err == nil {
		content := string(envData)
		// Get a fresh key from env or the one already entered
		var key string
		switch provider {
		case "openai":
			key = os.Getenv("OPENAI_API_KEY")
			if key == "" {
				key = promptSecret("OpenAI API key (for PentAGI .env)")
			}
			content = patchEnvVar(content, "OPEN_AI_KEY", key)
		case "anthropic":
			key = os.Getenv("ANTHROPIC_API_KEY")
			if key == "" {
				key = promptSecret("Anthropic API key (for PentAGI .env)")
			}
			content = patchEnvVar(content, "ANTHROPIC_API_KEY", key)
		}
		// Strip inline comments that break docker-compose
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if idx := strings.Index(line, " #"); idx > 0 && !strings.HasPrefix(strings.TrimSpace(line), "#") {
				lines[i] = line[:idx]
			}
		}
		os.WriteFile(envDst, []byte(strings.Join(lines, "\n")), 0644)
	}

	info("Starting PentAGI...")
	cmd := exec.Command("docker", "compose", "-f", composeDst, "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		errMsg(fmt.Sprintf("Failed to start PentAGI: %v", err))
		os.Exit(1)
	}

	saveAgentConfig("pentagi", AgentConfig{
		Provider:    provider,
		Model:       model,
		InstalledAt: nowISO(),
	})

	printPentAGIReady()
}

func launchPentAGI() {
	section("Launching PentAGI")
	composePath := filepath.Join(agentDir("pentagi"), "docker-compose.yml")
	if _, err := os.Stat(composePath); err != nil {
		errMsg("PentAGI compose file not found — run setup first")
		os.Exit(1)
	}

	cmd := exec.Command("docker", "compose", "-f", composePath, "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		errMsg(fmt.Sprintf("Failed to start PentAGI: %v", err))
		os.Exit(1)
	}

	printPentAGIReady()
}

func stopPentAGI() {
	section("Stopping PentAGI")
	composePath := filepath.Join(agentDir("pentagi"), "docker-compose.yml")
	if _, err := os.Stat(composePath); err != nil {
		warn("PentAGI compose file not found")
		return
	}
	cmd := exec.Command("docker", "compose", "-f", composePath, "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		warn(fmt.Sprintf("docker compose down error: %v", err))
	} else {
		info("PentAGI stopped")
	}
}

func uninstallPentAGI() {
	composePath := filepath.Join(agentDir("pentagi"), "docker-compose.yml")
	if _, err := os.Stat(composePath); err == nil {
		cmd := exec.Command("docker", "compose", "-f", composePath, "down", "-v", "--rmi", "all")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}
}

func printPentAGIReady() {
	bold := "\033[1m"
	cyan := "\033[0;36m"
	dim := "\033[2m"
	reset := "\033[0m"

	section("PentAGI is Ready")
	fmt.Printf("  %sWeb UI:%s  %shttps://localhost:8443%s\n", bold, reset, cyan, reset)
	fmt.Printf("  %sLogin:%s   admin@pentagi.com / admin\n\n", bold, reset)
	fmt.Printf("  %sTo test against LABYRINTH, create a Flow and enter:%s\n\n", bold, reset)
	fmt.Printf("  %s  SSH target:%s\n", dim, reset)
	fmt.Printf("  %s  Penetration test the SSH service at %s:2222%s\n\n", dim, targetHost(), reset)
	fmt.Printf("  %s  HTTP target:%s\n", dim, reset)
	fmt.Printf("  %s  Penetration test the web app at http://%s:8080%s\n\n", dim, targetHTTPHost(), reset)
	fmt.Printf("  %sTeardown: labyrinth attacker stop pentagi%s\n\n", dim, reset)
}

// ═══════════════════════════════════════════════════════════════
//  PentestAgent — AI pentesting framework (interactive TUI)
//  https://github.com/GH05TCREW/PentestAgent
// ═══════════════════════════════════════════════════════════════

func setupPentestAgent() {
	section("Setting up PentestAgent")

	provider, model, envFlags := collectAPIKey()

	info("Pulling PentestAgent Kali image...")
	pullCmd := exec.Command("docker", "pull", "ghcr.io/gh05tcrew/pentestagent:kali")
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr
	if err := pullCmd.Run(); err != nil {
		errMsg(fmt.Sprintf("Failed to pull image: %v", err))
		os.Exit(1)
	}

	saveAgentConfig("pentestagent", AgentConfig{
		Provider:    provider,
		Model:       model,
		InstalledAt: nowISO(),
	})

	info("PentestAgent installed")
	fmt.Println()

	answer := promptInput("Launch now? [Y/n]")
	if answer == "" || strings.ToLower(answer) == "y" {
		launchPentestAgentWith(provider, model, envFlags)
	}
}

func launchPentestAgent() {
	provider, model, envFlags := resolveAgentAPIKey("pentestagent")
	launchPentestAgentWith(provider, model, envFlags)
}

func launchPentestAgentWith(provider, model string, envFlags []string) {
	section("Launching PentestAgent")

	sshTarget := targetHost()
	httpTarget := targetHTTPHost()

	bold := "\033[1m"
	dim := "\033[2m"
	reset := "\033[0m"

	fmt.Printf("  %sLaunching interactive container with Kali tools...%s\n\n", bold, reset)
	fmt.Printf("  %sInside the TUI, try:%s\n", dim, reset)
	fmt.Printf("  %s  /agent Pentest SSH at %s:2222%s\n", dim, sshTarget, reset)
	fmt.Printf("  %s  /agent Pentest web app at http://%s:8080%s\n", dim, httpTarget, reset)
	fmt.Printf("  %s  /crew Full pentest of %s:2222 and http://%s:8080%s\n\n", dim, sshTarget, httpTarget, reset)
	fmt.Printf("  %sPress Ctrl+D or /quit to exit%s\n\n", dim, reset)

	// Build docker run args
	args := []string{"run", "-it", "--rm", "--name", "labyrinth-attacker-pentestagent"}
	args = append(args, netFlags()...)
	args = append(args, envFlags...)

	switch provider {
	case "openai":
		args = append(args, "-e", "PENTESTAGENT_MODEL=gpt-4o")
	case "anthropic":
		args = append(args, "-e", "PENTESTAGENT_MODEL=claude-sonnet-4-20250514")
	case "ollama":
		args = append(args, "-e", "PENTESTAGENT_MODEL=ollama/"+model)
		args = append(args, "-e", "OLLAMA_BASE_URL=http://host.docker.internal:11434")
	}

	args = append(args, "ghcr.io/gh05tcrew/pentestagent:kali")

	cmd := exec.Command("docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

// ═══════════════════════════════════════════════════════════════
//  Strix — AI hacker agents (host CLI + Docker sandbox)
//  https://github.com/UseStrix/strix
// ═══════════════════════════════════════════════════════════════

func setupStrix() {
	section("Setting up Strix")

	provider, model, _ := collectAPIKey()

	info("Pulling Strix sandbox image...")
	pullCmd := exec.Command("docker", "pull", "ghcr.io/usestrix/strix-sandbox:latest")
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr
	if err := pullCmd.Run(); err != nil {
		warn("Latest tag failed, trying 0.1.12...")
		pullCmd2 := exec.Command("docker", "pull", "ghcr.io/usestrix/strix-sandbox:0.1.12")
		pullCmd2.Stdout = os.Stdout
		pullCmd2.Stderr = os.Stderr
		if err := pullCmd2.Run(); err != nil {
			errMsg(fmt.Sprintf("Failed to pull Strix sandbox image: %v", err))
			os.Exit(1)
		}
	}

	saveAgentConfig("strix", AgentConfig{
		Provider:    provider,
		Model:       model,
		InstalledAt: nowISO(),
	})

	printStrixInstructions(provider, model)
}

func launchStrix() {
	cfg := loadAgentConfig("strix")
	provider := "openai"
	model := "gpt-4o"
	if cfg != nil {
		provider = cfg.Provider
		model = cfg.Model
	}
	printStrixInstructions(provider, model)
}

func printStrixInstructions(provider, model string) {
	bold := "\033[1m"
	dim := "\033[2m"
	cyan := "\033[0;36m"
	yellow := "\033[1;33m"
	reset := "\033[0m"

	section("Strix is Ready")
	fmt.Printf("  %sStrix runs as a host CLI that launches Docker sandboxes.%s\n\n", bold, reset)
	fmt.Printf("  %sInstall Strix:%s\n", bold, reset)
	fmt.Printf("  %scurl -sSL https://strix.ai/install | bash%s\n\n", dim, reset)
	fmt.Printf("  %sThen run against LABYRINTH:%s\n", bold, reset)

	switch provider {
	case "openai":
		fmt.Printf("  %sexport STRIX_LLM=openai/%s%s\n", dim, model, reset)
	case "anthropic":
		fmt.Printf("  %sexport STRIX_LLM=anthropic/%s%s\n", dim, model, reset)
	case "ollama":
		fmt.Printf("  %sexport STRIX_LLM=ollama/%s%s\n", dim, model, reset)
	}
	fmt.Printf("  %sexport LLM_API_KEY=<your-key>%s\n\n", dim, reset)
	fmt.Printf("  %sstrix --target http://localhost:8080%s\n", cyan, reset)
	fmt.Printf("  %sstrix --target localhost --instruction \"Pentest SSH on port 2222\"%s\n\n", cyan, reset)
	fmt.Printf("  %sNote: Strix launches its own Docker sandbox containers.%s\n", yellow, reset)
	fmt.Printf("  %sResults saved to ./strix_runs/ in your working directory.%s\n\n", yellow, reset)
}

// ═══════════════════════════════════════════════════════════════
//  Custom Kali — Interactive Kali Linux container
// ═══════════════════════════════════════════════════════════════

func setupKali() {
	section("Setting up Custom Kali")

	info("Pulling Kali Linux image...")
	pullCmd := exec.Command("docker", "pull", "kalilinux/kali-rolling")
	pullCmd.Stdout = os.Stdout
	pullCmd.Stderr = os.Stderr
	if err := pullCmd.Run(); err != nil {
		errMsg(fmt.Sprintf("Failed to pull Kali image: %v", err))
		os.Exit(1)
	}

	saveAgentConfig("kali", AgentConfig{
		Provider:    "",
		Model:       "",
		InstalledAt: nowISO(),
	})

	info("Kali Linux installed")
	fmt.Println()

	answer := promptInput("Launch now? [Y/n]")
	if answer == "" || strings.ToLower(answer) == "y" {
		launchKali()
	}
}

func launchKali() {
	section("Launching Custom Kali Container")

	sshTarget := targetHost()
	httpTarget := targetHTTPHost()

	bold := "\033[1m"
	dim := "\033[2m"
	reset := "\033[0m"

	fmt.Printf("  %sYou'll get a root shell with common pentest tools.%s\n", bold, reset)
	fmt.Printf("  %sThe container is connected to the LABYRINTH network.%s\n\n", bold, reset)
	fmt.Printf("  %sTargets from inside the container:%s\n", dim, reset)
	fmt.Printf("  %s  SSH:   %s:22   (mapped from host :2222)%s\n", dim, sshTarget, reset)
	fmt.Printf("  %s  HTTP:  %s:80   (mapped from host :8080)%s\n", dim, httpTarget, reset)
	fmt.Printf("  %s  Dash:  labyrinth-dashboard:9000%s\n\n", dim, reset)
	fmt.Printf("  %sExample commands:%s\n", dim, reset)
	fmt.Printf("  %s  nmap -sV %s%s\n", dim, sshTarget, reset)
	fmt.Printf("  %s  ssh root@%s%s\n", dim, sshTarget, reset)
	fmt.Printf("  %s  curl http://%s%s\n", dim, httpTarget, reset)
	fmt.Printf("  %s  hydra -l root -P /usr/share/wordlists/rockyou.txt ssh://%s%s\n\n", dim, sshTarget, reset)
	fmt.Printf("  %sPress Ctrl+D or type 'exit' to leave%s\n\n", dim, reset)

	args := []string{
		"run", "-it", "--rm",
		"--name", "labyrinth-attacker-kali",
		"--hostname", "attacker",
	}
	args = append(args, netFlags()...)
	args = append(args, "kalilinux/kali-rolling")
	args = append(args, "/bin/bash", "-c", `
echo "[*] Updating package list..."
apt-get update -qq 2>/dev/null
echo "[*] Installing core tools..."
DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
    nmap hydra curl wget netcat-openbsd sqlmap nikto dirb sshpass \
    2>/dev/null
echo ""
echo "[+] Tools ready. You are on the LABYRINTH network."
echo ""
exec /bin/bash
`)

	cmd := exec.Command("docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run()
}

// ── Shared Helpers ──────────────────────────────────────────

func downloadFile(url, dst string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func patchEnvVar(content, key, value string) string {
	lines := strings.Split(content, "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") {
			lines[i] = key + "=" + value
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, key+"="+value)
	}
	return strings.Join(lines, "\n")
}

// resolveAgentAPIKey loads saved config and resolves the API key from env vars,
// prompting if not found. Returns provider, model, and docker env flags.
func resolveAgentAPIKey(agentID string) (provider, model string, envFlags []string) {
	cfg := loadAgentConfig(agentID)

	if cfg != nil {
		provider = cfg.Provider
		model = cfg.Model
	}

	switch provider {
	case "openai":
		if model == "" {
			model = "gpt-4o"
		}
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
	case "anthropic":
		if model == "" {
			model = "claude-sonnet-4-20250514"
		}
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
	case "ollama":
		if model == "" {
			model = "llama3"
		}
		warn("Make sure Ollama is running on your host (ollama serve)")
	default:
		// No saved config — go through full collection
		return collectAPIKey()
	}

	return provider, model, envFlags
}
