package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the health of a running LABYRINTH deployment",
	Long: `Run diagnostic checks on a LABYRINTH deployment to verify all
services are running correctly and provide helpful tips for common issues.`,
	Run: runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

type checkResult struct {
	name   string
	status string // "pass", "fail", "warn"
	detail string
	tip    string
}

func runDoctor(cmd *cobra.Command, args []string) {
	green := "\033[0;32m"
	red := "\033[0;31m"
	yellow := "\033[1;33m"
	dim := "\033[2m"
	bold := "\033[1m"
	reset := "\033[0m"

	section("LABYRINTH Doctor")

	var results []checkResult
	passes, fails, warns := 0, 0, 0

	emit := func(r checkResult) {
		results = append(results, r)
		switch r.status {
		case "pass":
			passes++
			fmt.Printf("  %s[PASS]%s %s", green, reset, r.name)
			if r.detail != "" {
				fmt.Printf(" %s— %s%s", dim, r.detail, reset)
			}
			fmt.Println()
		case "fail":
			fails++
			fmt.Printf("  %s[FAIL]%s %s", red, reset, r.name)
			if r.detail != "" {
				fmt.Printf(" %s— %s%s", dim, r.detail, reset)
			}
			fmt.Println()
			if r.tip != "" {
				fmt.Printf("         %sTip: %s%s\n", yellow, r.tip, reset)
			}
		case "warn":
			warns++
			fmt.Printf("  %s[WARN]%s %s", yellow, reset, r.name)
			if r.detail != "" {
				fmt.Printf(" %s— %s%s", dim, r.detail, reset)
			}
			fmt.Println()
			if r.tip != "" {
				fmt.Printf("         %sTip: %s%s\n", dim, r.tip, reset)
			}
		}
	}

	// ── 1. Docker daemon ──
	if err := exec.Command("docker", "info").Run(); err != nil {
		emit(checkResult{"Docker daemon", "fail", "not running", "Start Docker Desktop or run: sudo systemctl start docker"})
	} else {
		out, _ := exec.Command("docker", "--version").Output()
		ver := strings.TrimSpace(string(out))
		emit(checkResult{"Docker daemon", "pass", ver, ""})
	}

	// ── 2. Docker Compose ──
	if err := exec.Command("docker", "compose", "version").Run(); err != nil {
		emit(checkResult{"Docker Compose", "fail", "not found", "Install Docker Compose v2: https://docs.docker.com/compose/install/"})
	} else {
		out, _ := exec.Command("docker", "compose", "version", "--short").Output()
		emit(checkResult{"Docker Compose", "pass", "v" + strings.TrimSpace(string(out)), ""})
	}

	// ── 3. Core containers ──
	coreContainers := []struct {
		name string
		desc string
		tip  string
	}{
		{"labyrinth-ssh", "SSH portal trap (L1)", "Run: labyrinth deploy -t"},
		{"labyrinth-http", "HTTP portal trap (L1)", "Run: labyrinth deploy -t"},
		{"labyrinth-orchestrator", "Session orchestrator", "Run: labyrinth deploy -t"},
		{"labyrinth-proxy", "L4 proxy + phantom services", "Run: labyrinth deploy -t"},
		{"labyrinth-dashboard", "Web dashboard", "Run: labyrinth deploy -t"},
	}

	containerStates := map[string]string{}
	out, err := exec.Command("docker", "ps", "-a", "--filter", "label=project=labyrinth",
		"--format", "{{.Names}}\t{{.State}}").Output()
	if err == nil {
		for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
			parts := strings.SplitN(line, "\t", 2)
			if len(parts) == 2 {
				containerStates[parts[0]] = parts[1]
			}
		}
	}

	allContainersUp := true
	for _, c := range coreContainers {
		state, exists := containerStates[c.name]
		if !exists {
			emit(checkResult{c.desc, "fail", c.name + " not found", c.tip})
			allContainersUp = false
		} else if state != "running" {
			emit(checkResult{c.desc, "fail", c.name + " is " + state, "Check logs: docker logs " + c.name})
			allContainersUp = false
		} else {
			emit(checkResult{c.desc, "pass", c.name + " running", ""})
		}
	}

	// ── 4. Port bindings ──
	portOutput, _ := exec.Command("docker", "ps", "--filter", "label=project=labyrinth",
		"--format", "{{.Ports}}").Output()
	portStr := string(portOutput)

	portChecks := []struct {
		port int
		desc string
	}{
		{22, "SSH (port 22)"},
		{8080, "HTTP (port 8080)"},
		{9000, "Dashboard (port 9000)"},
	}

	for _, pc := range portChecks {
		if strings.Contains(portStr, fmt.Sprintf(":%d->", pc.port)) {
			emit(checkResult{pc.desc, "pass", "bound", ""})
		} else if allContainersUp {
			emit(checkResult{pc.desc, "warn", "binding not detected", fmt.Sprintf("Check: docker ps | grep %d", pc.port)})
		} else {
			emit(checkResult{pc.desc, "fail", "not bound", "Deploy first: labyrinth deploy -t"})
		}
	}

	// ── 5. Dashboard health ──
	dashOK := false
	if allContainersUp {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("http://localhost:9000/api/health")
		if err != nil {
			emit(checkResult{"Dashboard API", "fail", "connection refused", "Check: docker logs labyrinth-dashboard"})
		} else {
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			var health map[string]interface{}
			if json.Unmarshal(body, &health) == nil && health["status"] == "ok" {
				uptime, _ := health["uptime_seconds"].(float64)
				emit(checkResult{"Dashboard API", "pass", fmt.Sprintf("healthy (up %.0fs)", uptime), ""})
				dashOK = true
			} else {
				emit(checkResult{"Dashboard API", "fail", "unhealthy response", "Check: curl http://localhost:9000/api/health"})
			}
		}
	}

	// ── 6. Proxy entrypoint (start.sh vs old mitmdump) ──
	if state, exists := containerStates["labyrinth-proxy"]; exists && state == "running" {
		out, err := exec.Command("docker", "inspect", "--format",
			"{{index .Config.Cmd 0}}", "labyrinth-proxy").Output()
		entrypoint := strings.TrimSpace(string(out))
		if err != nil {
			emit(checkResult{"Proxy entrypoint", "warn", "could not inspect", ""})
		} else if strings.Contains(entrypoint, "start.sh") {
			emit(checkResult{"Proxy entrypoint", "pass", "start.sh (phantom + interceptor)", ""})
		} else if strings.Contains(entrypoint, "mitmdump") {
			emit(checkResult{"Proxy entrypoint", "fail", "old mitmdump-only CMD",
				"Rebuild: docker compose build proxy && docker compose up -d proxy"})
		} else {
			emit(checkResult{"Proxy entrypoint", "warn", entrypoint, ""})
		}
	}

	// ── 7. Phantom services listening ──
	if state, exists := containerStates["labyrinth-proxy"]; exists && state == "running" {
		out, err := exec.Command("docker", "logs", "--tail", "50", "labyrinth-proxy").CombinedOutput()
		logs := string(out)
		if err != nil {
			emit(checkResult{"Phantom services", "warn", "could not read proxy logs", ""})
		} else if strings.Contains(logs, "All services ready") {
			// Count how many services are mentioned
			serviceCount := 0
			for _, svc := range []string{"PostgreSQL", "Redis", "Elasticsearch", "Consul", "Jenkins", "SSH relay"} {
				if strings.Contains(logs, svc+" listening") {
					serviceCount++
				}
			}
			emit(checkResult{"Phantom services", "pass", fmt.Sprintf("%d/6 services listening", serviceCount), ""})
		} else if strings.Contains(logs, "Starting network service handler") {
			emit(checkResult{"Phantom services", "warn", "starting (may need a moment)", "Wait a few seconds and run doctor again"})
		} else {
			emit(checkResult{"Phantom services", "fail", "not detected in proxy logs",
				"The proxy image may be outdated. Rebuild: docker compose build --no-cache proxy"})
		}
	}

	// ── 8. Identity loaded ──
	if state, exists := containerStates["labyrinth-proxy"]; exists && state == "running" {
		out, _ := exec.Command("docker", "logs", "--tail", "50", "labyrinth-proxy").CombinedOutput()
		logs := string(out)
		if strings.Contains(logs, "Configuration data loaded") {
			// Extract company name
			idx := strings.Index(logs, "Configuration data loaded: ")
			company := ""
			if idx >= 0 {
				rest := logs[idx+len("Configuration data loaded: "):]
				if nl := strings.Index(rest, "\n"); nl >= 0 {
					company = strings.TrimSpace(rest[:nl])
				}
			}
			if company != "" {
				emit(checkResult{"Identity loaded", "pass", company, ""})
			} else {
				emit(checkResult{"Identity loaded", "pass", "config.json read", ""})
			}
		} else {
			emit(checkResult{"Identity loaded", "warn", "no identity detected",
				"Drop bait first: labyrinth bait drop (identity is generated during bait drop)"})
		}
	}

	// ── 9. MITM interceptor ──
	if state, exists := containerStates["labyrinth-proxy"]; exists && state == "running" {
		out, _ := exec.Command("docker", "logs", "--tail", "50", "labyrinth-proxy").CombinedOutput()
		logs := string(out)
		if strings.Contains(logs, "Transparent Proxy listening") || strings.Contains(logs, "PUPPETEER interceptor loaded") {
			emit(checkResult{"MITM interceptor", "pass", "transparent proxy on :8443", ""})
		} else if strings.Contains(logs, "Starting MITM interceptor") {
			emit(checkResult{"MITM interceptor", "pass", "interceptor running (mitmdump)", ""})
		} else {
			emit(checkResult{"MITM interceptor", "warn", "not detected in logs",
				"MITM activates after phantom services start. Check: docker logs labyrinth-proxy"})
		}
	}

	// ── 10. Bait planted ──
	home, _ := os.UserHomeDir()
	baitFile := filepath.Join(home, ".labyrinth", "bait.json")
	if _, err := os.Stat(baitFile); err == nil {
		data, _ := os.ReadFile(baitFile)
		var bait map[string]interface{}
		if json.Unmarshal(data, &bait) == nil {
			company, _ := bait["company"].(string)
			if company != "" {
				emit(checkResult{"Bait credentials", "pass", "planted (" + company + ")", ""})
			} else {
				emit(checkResult{"Bait credentials", "pass", "planted", ""})
			}
		} else {
			emit(checkResult{"Bait credentials", "warn", "bait.json unreadable", "Re-drop: labyrinth bait drop"})
		}
	} else {
		emit(checkResult{"Bait credentials", "warn", "not planted yet",
			"Attacker agents need bait to find the trap. Run: labyrinth bait drop"})
	}

	// ── 11. Bait credential sync ──
	// Cross-check bait.json against config.json in HTTP and proxy containers
	if manifest := loadBaitManifest(); manifest != nil && allContainersUp {
		baitCompany := manifest.Company

		// Read config.json from HTTP container
		httpCfgOut, httpCfgErr := exec.Command("docker", "exec", "labyrinth-http",
			"cat", "/var/log/audit/config.json").Output()
		// Read config.json from proxy container
		proxyCfgOut, proxyCfgErr := exec.Command("docker", "exec", "labyrinth-proxy",
			"cat", "/var/labyrinth/forensics/config.json").Output()

		var httpCfg, proxyCfg map[string]interface{}
		httpOK := httpCfgErr == nil && json.Unmarshal(httpCfgOut, &httpCfg) == nil
		proxyOK := proxyCfgErr == nil && json.Unmarshal(proxyCfgOut, &proxyCfg) == nil

		if !httpOK && !proxyOK {
			emit(checkResult{"Bait sync", "warn", "config.json not found in containers",
				"Run: labyrinth bait drop (writes config.json to shared volumes)"})
		} else {
			mismatches := []string{}

			if httpOK {
				httpCompany, _ := httpCfg["company"].(string)
				if httpCompany != baitCompany {
					mismatches = append(mismatches, fmt.Sprintf("HTTP has \"%s\"", httpCompany))
				}
				httpAWS, _ := httpCfg["aws_key_id"].(string)
				if httpAWS != "" && httpAWS != manifest.AWSKeyID {
					mismatches = append(mismatches, "HTTP AWS key mismatch")
				}
				httpDB, _ := httpCfg["db_pass"].(string)
				if httpDB != "" && httpDB != manifest.DBPass {
					mismatches = append(mismatches, "HTTP db_pass mismatch")
				}
			}
			if proxyOK {
				proxyCompany, _ := proxyCfg["company"].(string)
				if proxyCompany != baitCompany {
					mismatches = append(mismatches, fmt.Sprintf("proxy has \"%s\"", proxyCompany))
				}
				proxyAWS, _ := proxyCfg["aws_key_id"].(string)
				if proxyAWS != "" && proxyAWS != manifest.AWSKeyID {
					mismatches = append(mismatches, "proxy AWS key mismatch")
				}
				proxyDB, _ := proxyCfg["db_pass"].(string)
				if proxyDB != "" && proxyDB != manifest.DBPass {
					mismatches = append(mismatches, "proxy db_pass mismatch")
				}
			}

			if len(mismatches) > 0 {
				emit(checkResult{"Bait sync", "fail",
					"credentials out of sync: " + strings.Join(mismatches, ", "),
					"Re-sync: labyrinth bait drop"})
			} else {
				syncedWith := []string{}
				if httpOK {
					syncedWith = append(syncedWith, "HTTP")
				}
				if proxyOK {
					syncedWith = append(syncedWith, "proxy")
				}
				emit(checkResult{"Bait sync", "pass",
					fmt.Sprintf("bait.json matches %s config", strings.Join(syncedWith, " + ")), ""})
			}
		}

		// Check that bait SSH users exist in the SSH container
		if state, exists := containerStates["labyrinth-ssh"]; exists && state == "running" {
			missingUsers := []string{}
			for _, u := range manifest.Users {
				err := exec.Command("docker", "exec", "labyrinth-ssh", "id", u.Username).Run()
				if err != nil {
					missingUsers = append(missingUsers, u.Username)
				}
			}
			if len(missingUsers) > 0 {
				emit(checkResult{"Bait SSH users", "fail",
					"missing: " + strings.Join(missingUsers, ", "),
					"Re-drop: labyrinth bait drop"})
			} else {
				emit(checkResult{"Bait SSH users", "pass",
					fmt.Sprintf("%d users present in SSH container", len(manifest.Users)), ""})
			}
		}

		// Check that static bait files are served by HTTP
		if state, exists := containerStates["labyrinth-http"]; exists && state == "running" {
			out, err := exec.Command("docker", "exec", "labyrinth-http",
				"ls", "/var/log/audit/static/.env").Output()
			if err != nil || len(strings.TrimSpace(string(out))) == 0 {
				emit(checkResult{"Bait static files", "fail", ".env not in STATIC_DIR",
					"Re-drop: labyrinth bait drop"})
			} else {
				emit(checkResult{"Bait static files", "pass", "present in /var/log/audit/static/", ""})
			}
		}
	}

	// ── 12. L4 services API ──
	if dashOK {
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Get("http://localhost:9000/api/l4/services")
		if err != nil {
			emit(checkResult{"L4 services API", "warn", "endpoint not reachable",
				"Dashboard may need rebuild: docker compose build dashboard && docker compose up -d dashboard"})
		} else {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 404 {
				emit(checkResult{"L4 services API", "warn", "endpoint not found (404)",
					"Dashboard needs rebuild: docker compose build dashboard && docker compose up -d dashboard"})
			} else {
				var svcResp map[string]interface{}
				if json.Unmarshal(body, &svcResp) == nil {
					services, _ := svcResp["services"].([]interface{})
					if len(services) > 0 {
						emit(checkResult{"L4 services API", "pass", fmt.Sprintf("%d service definitions", len(services)), ""})
					} else {
						emit(checkResult{"L4 services API", "warn", "no services returned",
							"Orchestrator may need rebuild: docker compose build orchestrator"})
					}
				} else {
					emit(checkResult{"L4 services API", "fail", "invalid response",
						"Check: curl http://localhost:9000/api/l4/services"})
				}
			}
		}
	}

	// ── 12. Container network ──
	out, err = exec.Command("docker", "network", "ls", "--filter", "label=project=labyrinth",
		"--format", "{{.Name}}\t{{.Driver}}").Output()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		emit(checkResult{"Docker network", "pass", fmt.Sprintf("%d network(s)", len(lines)), ""})
	} else {
		// Try broader search
		out, _ = exec.Command("docker", "network", "ls", "--format", "{{.Name}}").Output()
		found := false
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "labyrinth") {
				found = true
				break
			}
		}
		if found {
			emit(checkResult{"Docker network", "pass", "labyrinth network found", ""})
		} else {
			emit(checkResult{"Docker network", "fail", "no labyrinth network",
				"Deploy creates the network: labyrinth deploy -t"})
		}
	}

	// ── Summary ──
	fmt.Println()
	fmt.Printf("  %s━━━ Summary ━━━%s\n\n", "\033[0;35m", reset)

	total := passes + fails + warns
	fmt.Printf("  %s%d/%d checks passed%s", bold, passes, total, reset)
	if fails > 0 {
		fmt.Printf("  %s%d failed%s", red, fails, reset)
	}
	if warns > 0 {
		fmt.Printf("  %s%d warnings%s", yellow, warns, reset)
	}
	fmt.Println()

	if fails == 0 && warns == 0 {
		fmt.Printf("\n  %sAll systems operational. Ready to engage.%s\n", green, reset)
	} else if fails == 0 {
		fmt.Printf("\n  %sCore systems operational with minor warnings.%s\n", green, reset)
	} else {
		fmt.Printf("\n  %sSome checks failed — review the tips above to fix issues.%s\n", red, reset)
		if !allContainersUp {
			fmt.Printf("  %sMost issues resolve by deploying: %slabyrinth deploy -t%s\n", dim, bold, reset)
		}
	}
	fmt.Println()
}
