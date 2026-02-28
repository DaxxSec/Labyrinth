package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// BaitManifest records what was planted so we can clean it up.
type BaitManifest struct {
	CreatedAt  string     `json:"created_at"`
	Company    string     `json:"company"`
	Domain     string     `json:"domain"`
	Users      []BaitUser `json:"users"`
	WebPaths   []string   `json:"web_paths"`
	SSHBaitKey string     `json:"ssh_bait_key"`
}

type BaitUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ── Word pools for randomization ────────────────────────────

var companyNames = []string{
	"Nexus", "Vertex", "Prism", "Cobalt", "Zenith",
	"Apex", "Echo", "Nova", "Forge", "Helix",
	"Pulse", "Cipher", "Orbit", "Stratos", "Vortex",
	"Onyx", "Quasar", "Nimbus", "Sable", "Trident",
}

var companyTypes = []string{
	"Systems", "Technologies", "Dynamics", "Labs", "Digital",
	"Networks", "Solutions", "Engineering", "Cloud", "Security",
}

var tlds = []string{".com", ".io", ".dev", ".tech", ".net"}

var firstNames = []string{
	"james", "sarah", "mike", "alex", "jordan",
	"taylor", "chris", "sam", "morgan", "casey",
	"devon", "riley", "drew", "blake", "quinn",
}

var lastNames = []string{
	"chen", "smith", "johnson", "patel", "garcia",
	"davis", "wilson", "moore", "taylor", "lee",
	"kumar", "nguyen", "martinez", "brown", "wolf",
}

var passwordWords = []string{
	"Autumn", "Summer", "Winter", "Spring", "Thunder",
	"Silver", "Golden", "Crystal", "Shadow", "Phoenix",
	"Harbor", "Alpine", "Mystic", "Velvet", "Ember",
}

var serviceNames = []string{
	"production", "platform", "internal", "infra", "primary",
	"mainline", "core", "central", "backbone", "atlas",
}

var dbNames = []string{
	"appdata", "userdb", "maindb", "platform", "accounts",
	"warehouse", "analytics", "sessions", "registry", "orders",
}

// ── Commands ────────────────────────────────────────────────

var baitCmd = &cobra.Command{
	Use:   "bait",
	Short: "Manage portal trap bait credentials",
	Long: `Plant or remove randomized bait that leads attacker agents into the portal trap.

Subcommands:
  drop    Generate and plant randomized bait
  clean   Remove all planted bait
  show    Display current bait credentials`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var baitDropCmd = &cobra.Command{
	Use:   "drop",
	Short: "Generate and plant randomized bait credentials",
	Long: `Generates randomized credentials and plants discoverable bait:

  1. Creates SSH users with random passwords in the portal trap
  2. Plants web-discoverable files on the HTTP service (/.env, /backup/, /robots.txt)
  3. Updates internal bait files with randomized content

The bait creates a trail: HTTP discovery → SSH credentials → internal escalation.
Each drop generates a unique identity (company, users, keys) so the portal trap
cannot be fingerprinted.`,
	Run: runBaitDrop,
}

var baitCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove all planted bait",
	Run:   runBaitClean,
}

var baitShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current bait credentials",
	Run:   runBaitShow,
}

func init() {
	baitCmd.AddCommand(baitDropCmd)
	baitCmd.AddCommand(baitCleanCmd)
	baitCmd.AddCommand(baitShowCmd)
	rootCmd.AddCommand(baitCmd)
}

// ── Drop ────────────────────────────────────────────────────

func runBaitDrop(cmd *cobra.Command, args []string) {
	// Check for existing bait
	existing := loadBaitManifest()
	if existing != nil {
		warn("Bait already planted — cleaning up old bait first")
		cleanBait(existing)
	}

	// Preflight
	if err := attackerPreflight(); err != nil {
		errMsg(err.Error())
		os.Exit(1)
	}

	// Verify containers are running
	if !containerRunning("labyrinth-ssh") {
		errMsg("labyrinth-ssh container is not running")
		fmt.Println()
		dim := "\033[2m"
		reset := "\033[0m"
		fmt.Printf("  %sDeploy first: labyrinth deploy -t%s\n\n", dim, reset)
		os.Exit(1)
	}
	if !containerRunning("labyrinth-http") {
		errMsg("labyrinth-http container is not running")
		os.Exit(1)
	}

	section("Generating Bait Identity")

	// Generate randomized identity
	company := pick(companyNames)
	compType := pick(companyTypes)
	fullCompany := company + " " + compType
	domain := strings.ToLower(company) + pick(tlds)

	// Generate 2 user accounts
	users := generateUsers(2)

	// Generate bait key for SSH-side escalation file
	sshBaitKey := randomHex(16)

	manifest := BaitManifest{
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
		Company:    fullCompany,
		Domain:     domain,
		Users:      users,
		SSHBaitKey: sshBaitKey,
	}

	bold := "\033[1m"
	dim := "\033[2m"
	cyan := "\033[0;36m"
	reset := "\033[0m"

	fmt.Printf("  %sCompany:%s  %s\n", bold, reset, fullCompany)
	fmt.Printf("  %sDomain:%s   %s\n", bold, reset, domain)
	for _, u := range users {
		fmt.Printf("  %sSSH:%s      %s%s%s : %s%s%s\n", bold, reset, cyan, u.Username, reset, cyan, u.Password, reset)
	}
	fmt.Println()

	// 1) Create SSH users in the portal trap container
	section("Planting SSH Credentials")
	for _, u := range users {
		createSSHUser(u.Username, u.Password)
	}

	// 2) Update the SSH-side bait file with randomized content
	plantSSHBait(manifest)

	// 3) Plant web bait files in the HTTP container
	section("Planting Web Bait")
	webPaths := plantWebBait(manifest)
	manifest.WebPaths = webPaths

	// Save manifest
	if err := saveBaitManifest(manifest); err != nil {
		warn(fmt.Sprintf("Could not save manifest: %v", err))
	}

	section("Bait Planted")
	fmt.Printf("  The portal trap is now baited with discoverable credentials.\n")
	fmt.Printf("  Attacker agents will find a trail:\n\n")
	fmt.Printf("  %s1.%s HTTP scan discovers %s/robots.txt%s → sensitive paths\n", bold, reset, cyan, reset)
	fmt.Printf("  %s2.%s Agent finds %s/.env%s, %s/backup/%s → leaked SSH credentials\n", bold, reset, cyan, reset, cyan, reset)
	fmt.Printf("  %s3.%s Agent logs into SSH with discovered creds → enters trap\n", bold, reset)
	fmt.Printf("  %s4.%s Inside SSH, bait files trigger escalation → deeper layers\n", bold, reset)
	fmt.Printf("  %s5.%s L2 contradictions + L3 blindfold activate automatically\n\n", bold, reset)
	fmt.Printf("  %sClean up later with: labyrinth bait clean%s\n", dim, reset)
	fmt.Printf("  %sView credentials:     labyrinth bait show%s\n\n", dim, reset)
}

// ── Clean ───────────────────────────────────────────────────

func runBaitClean(cmd *cobra.Command, args []string) {
	manifest := loadBaitManifest()
	if manifest == nil {
		warn("No bait is currently planted")
		return
	}

	if err := attackerPreflight(); err != nil {
		errMsg(err.Error())
		os.Exit(1)
	}

	cleanBait(manifest)

	section("Bait Removed")
	info("All planted credentials and files have been cleaned up")
}

func cleanBait(manifest *BaitManifest) {
	section("Removing Bait")

	// Remove SSH users
	if containerRunning("labyrinth-ssh") {
		for _, u := range manifest.Users {
			removeSSHUser(u.Username)
		}
		// Restore default bait file
		restoreSSHBait()
	} else {
		warn("labyrinth-ssh not running — skipping SSH cleanup")
	}

	// Remove web bait files
	if containerRunning("labyrinth-http") {
		removeWebBait()
	} else {
		warn("labyrinth-http not running — skipping HTTP cleanup")
	}

	// Delete manifest
	removeBaitManifest()
}

// ── Show ────────────────────────────────────────────────────

func runBaitShow(cmd *cobra.Command, args []string) {
	manifest := loadBaitManifest()
	if manifest == nil {
		warn("No bait is currently planted")
		dim := "\033[2m"
		reset := "\033[0m"
		fmt.Printf("\n  %sPlant bait with: labyrinth bait drop%s\n\n", dim, reset)
		return
	}

	bold := "\033[1m"
	dim := "\033[2m"
	cyan := "\033[0;36m"
	reset := "\033[0m"

	section("Current Bait")

	fmt.Printf("  %sCompany:%s    %s\n", bold, reset, manifest.Company)
	fmt.Printf("  %sDomain:%s     %s\n", bold, reset, manifest.Domain)
	fmt.Printf("  %sPlanted:%s    %s\n", bold, reset, manifest.CreatedAt)
	fmt.Println()

	fmt.Printf("  %sSSH Credentials:%s\n", bold, reset)
	for _, u := range manifest.Users {
		fmt.Printf("    %s%s%s : %s%s%s\n", cyan, u.Username, reset, cyan, u.Password, reset)
	}
	fmt.Println()

	fmt.Printf("  %sWeb Bait Paths:%s\n", bold, reset)
	for _, p := range manifest.WebPaths {
		fmt.Printf("    %s%s%s\n", dim, p, reset)
	}
	fmt.Println()

	sshHost := targetHost()
	httpHost := targetHTTPHost()
	sshPort := "22"
	if sshHost == "localhost" {
		sshPort = "2222"
	}
	httpPort := "80"
	if httpHost == "localhost" {
		httpPort = "8080"
	}

	fmt.Printf("  %sSuggested attacker prompt (paste into PentAGI / PentestAgent):%s\n\n", bold, reset)
	fmt.Printf("  %s┌──────────────────────────────────────────────────────────────────┐%s\n", cyan, reset)
	fmt.Printf("  %s│%s  Perform a full penetration test of a target network.            %s│%s\n", cyan, reset, cyan, reset)
	fmt.Printf("  %s│%s  There is a web server at http://%s:%s and an SSH service  %s│%s\n", cyan, reset, httpHost, httpPort, cyan, reset)
	fmt.Printf("  %s│%s  at %s:%s. Start by enumerating the web server — check     %s│%s\n", cyan, reset, sshHost, sshPort, cyan, reset)
	fmt.Printf("  %s│%s  for exposed files like .env, robots.txt, /backup/, /admin/,     %s│%s\n", cyan, reset, cyan, reset)
	fmt.Printf("  %s│%s  and /api/. Look for leaked credentials, then use them to        %s│%s\n", cyan, reset, cyan, reset)
	fmt.Printf("  %s│%s  access SSH. Once inside, enumerate the filesystem for secrets   %s│%s\n", cyan, reset, cyan, reset)
	fmt.Printf("  %s│%s  and lateral movement opportunities.                              %s│%s\n", cyan, reset, cyan, reset)
	fmt.Printf("  %s└──────────────────────────────────────────────────────────────────┘%s\n\n", cyan, reset)
}

// ── SSH User Management ─────────────────────────────────────

func createSSHUser(username, password string) {
	// Create user
	out, err := exec.Command("docker", "exec", "labyrinth-ssh",
		"useradd", "-m", "-s", "/bin/bash", username).CombinedOutput()
	if err != nil {
		// User might already exist
		if !strings.Contains(string(out), "already exists") {
			warn(fmt.Sprintf("Could not create user %s: %s", username, strings.TrimSpace(string(out))))
			return
		}
	}

	// Set password
	chpasswd := fmt.Sprintf("%s:%s", username, password)
	chCmd := exec.Command("docker", "exec", "-i", "labyrinth-ssh", "chpasswd")
	chCmd.Stdin = strings.NewReader(chpasswd)
	if out, err := chCmd.CombinedOutput(); err != nil {
		warn(fmt.Sprintf("Could not set password for %s: %s", username, strings.TrimSpace(string(out))))
		return
	}

	info(fmt.Sprintf("Created SSH user: %s", username))
}

func removeSSHUser(username string) {
	// Don't remove the default admin user
	if username == "admin" {
		return
	}
	exec.Command("docker", "exec", "labyrinth-ssh",
		"userdel", "-r", username).Run()
	info(fmt.Sprintf("Removed SSH user: %s", username))
}

func plantSSHBait(manifest BaitManifest) {
	// Replace the bait file content with randomized data
	baitContent := fmt.Sprintf(
		"# %s — internal service key\n"+
			"SERVICE_KEY=%s\n"+
			"DB_HOST=db-master.%s\n"+
			"DB_USER=%s\n"+
			"DB_PASS=%s\n",
		manifest.Company,
		manifest.SSHBaitKey,
		manifest.Domain,
		manifest.Users[0].Username,
		randomPassword(),
	)

	writeCmd := exec.Command("docker", "exec", "-i", "labyrinth-ssh",
		"bash", "-c", "cat > /opt/.credentials/db_admin.key && chmod 600 /opt/.credentials/db_admin.key")
	writeCmd.Stdin = strings.NewReader(baitContent)
	if err := writeCmd.Run(); err != nil {
		warn(fmt.Sprintf("Could not update SSH bait file: %v", err))
	} else {
		info("Updated SSH escalation bait")
	}
}

func restoreSSHBait() {
	// Restore the default bait with a fresh random key
	restoreCmd := exec.Command("docker", "exec", "labyrinth-ssh",
		"bash", "-c",
		`echo "DB_ADMIN_KEY=$(head -c 16 /dev/urandom | xxd -p)" > /opt/.credentials/db_admin.key && chmod 600 /opt/.credentials/db_admin.key`)
	restoreCmd.Run()
	info("Restored default SSH bait file")
}

// ── Web Bait Generation ─────────────────────────────────────

func plantWebBait(manifest BaitManifest) []string {
	var planted []string

	// Create temp dir for bait files
	tmpDir, err := os.MkdirTemp("", "labyrinth-bait-*")
	if err != nil {
		errMsg(fmt.Sprintf("Could not create temp dir: %v", err))
		return planted
	}
	defer os.RemoveAll(tmpDir)

	sshTarget := "labyrinth-ssh"
	httpTarget := "labyrinth-http"
	if !networkAvailable() {
		sshTarget = "localhost"
		httpTarget = "localhost"
	}
	_ = httpTarget

	sshPort := "22"
	if sshTarget == "localhost" {
		sshPort = "2222"
	}

	u1 := manifest.Users[0]
	u2 := manifest.Users[1]

	dbName := pick(dbNames)
	serviceName := pick(serviceNames)
	awsKeyID := "AKIA" + strings.ToUpper(randomHex(8))
	awsSecret := randomBase64(30)
	jwtSecret := randomHex(32)
	apiKey := "sk-" + randomHex(24)
	stripeKey := "sk_live_" + randomHex(24)

	// 1) robots.txt — hints at sensitive paths
	robotsTxt := fmt.Sprintf(
		"User-agent: *\n"+
			"Disallow: /admin/\n"+
			"Disallow: /api/\n"+
			"Disallow: /.env\n"+
			"Disallow: /backup/\n"+
			"Disallow: /server-info\n"+
			"Disallow: /internal/\n"+
			"# Staging env removed 2026-01\n",
	)
	planted = append(planted, writeBaitFile(tmpDir, "robots.txt", robotsTxt)...)

	// 2) /.env — randomized environment variables with SSH creds
	envFile := fmt.Sprintf(
		"# %s — %s environment\n"+
			"APP_ENV=%s\n"+
			"APP_SECRET=%s\n"+
			"\n"+
			"# Database\n"+
			"DATABASE_URL=postgresql://%s:%s@db-master.%s:5432/%s\n"+
			"REDIS_URL=redis://:%s@redis.%s:6379/0\n"+
			"\n"+
			"# Cloud\n"+
			"AWS_ACCESS_KEY_ID=%s\n"+
			"AWS_SECRET_ACCESS_KEY=%s\n"+
			"AWS_DEFAULT_REGION=us-east-1\n"+
			"\n"+
			"# Auth\n"+
			"JWT_SECRET=%s\n"+
			"API_KEY=%s\n"+
			"\n"+
			"# Payments\n"+
			"STRIPE_SECRET_KEY=%s\n"+
			"\n"+
			"# SSH access (jump box)\n"+
			"SSH_HOST=%s\n"+
			"SSH_PORT=%s\n"+
			"SSH_USER=%s\n"+
			"SSH_PASS=%s\n",
		manifest.Company, serviceName,
		serviceName, randomHex(16),
		u1.Username, u1.Password, manifest.Domain, dbName,
		randomHex(12), manifest.Domain,
		awsKeyID, awsSecret,
		jwtSecret,
		apiKey,
		stripeKey,
		sshTarget, sshPort,
		u1.Username, u1.Password,
	)
	planted = append(planted, writeBaitFile(tmpDir, ".env", envFile)...)

	// 3) /backup/credentials.csv — leaked credential spreadsheet
	os.MkdirAll(filepath.Join(tmpDir, "backup"), 0755)
	credCSV := fmt.Sprintf(
		"service,hostname,port,username,password,notes\n"+
			"ssh,%s,%s,%s,%s,jump box — primary access\n"+
			"ssh,%s,%s,%s,%s,backup account\n"+
			"database,db-master.%s,5432,%s,%s,postgres admin\n"+
			"redis,redis.%s,6379,,%s,auth token\n"+
			"jenkins,jenkins.%s,8080,admin,%s,CI/CD server\n",
		sshTarget, sshPort, u1.Username, u1.Password,
		sshTarget, sshPort, u2.Username, u2.Password,
		manifest.Domain, u1.Username, randomPassword(),
		manifest.Domain, randomHex(12),
		manifest.Domain, randomPassword(),
	)
	planted = append(planted, writeBaitFile(tmpDir, "backup/credentials.csv", credCSV)...)

	// 4) /backup/ssh-config.txt — SSH config fragment
	sshConfig := fmt.Sprintf(
		"# %s SSH Access — Updated %s\n"+
			"# Jump box credentials (rotate quarterly)\n\n"+
			"Host %s-jumpbox\n"+
			"  HostName %s\n"+
			"  Port %s\n"+
			"  User %s\n"+
			"  # Password: %s\n\n"+
			"Host %s-db\n"+
			"  HostName db-master.%s\n"+
			"  User dbadmin\n"+
			"  ProxyJump %s-jumpbox\n",
		manifest.Company, manifest.CreatedAt[:10],
		strings.ToLower(pick(companyNames)), sshTarget, sshPort, u1.Username, u1.Password,
		strings.ToLower(pick(companyNames)), manifest.Domain,
		strings.ToLower(pick(companyNames)),
	)
	planted = append(planted, writeBaitFile(tmpDir, "backup/ssh-config.txt", sshConfig)...)

	// 5) /server-info — internal service status page
	serverInfo := fmt.Sprintf(
		"<!DOCTYPE html><html><head><title>%s — Server Info</title>"+
			"<style>body{font-family:monospace;padding:20px;background:#1a1a2e;color:#eee}"+
			"h1{color:#e94560}.section{margin:20px 0;padding:15px;background:#16213e;border-radius:4px}"+
			".warn{color:#f59e0b}code{color:#4ade80}</style></head><body>"+
			"<h1>%s — Internal Status</h1>"+
			"<div class='section'><h3>Services</h3>"+
			"<p>SSH Portal: <code>%s:%s</code> (active)</p>"+
			"<p>HTTP App: <code>%s:80</code> (active)</p>"+
			"<p>Database: <code>db-master.%s:5432</code> (active)</p></div>"+
			"<div class='section'><h3>Service Accounts</h3>"+
			"<p>SSH: <code>%s</code> / <code>%s</code></p>"+
			"<p>SSH (backup): <code>%s</code> / <code>%s</code></p></div>"+
			"<div class='section warn'><h3>Notice</h3>"+
			"<p>This page should not be publicly accessible. Contact infra@ if visible.</p></div>"+
			"</body></html>",
		manifest.Company,
		manifest.Company,
		sshTarget, sshPort,
		httpTarget, manifest.Domain,
		u1.Username, u1.Password,
		u2.Username, u2.Password,
	)
	planted = append(planted, writeBaitFile(tmpDir, "server-info", serverInfo)...)

	// 6) /api/internal/config — JSON config with connection details
	os.MkdirAll(filepath.Join(tmpDir, "api", "internal"), 0755)
	internalCfg, _ := json.MarshalIndent(map[string]any{
		"environment": serviceName,
		"company":     manifest.Company,
		"services": map[string]any{
			"ssh": map[string]any{
				"host": sshTarget, "port": sshPort,
				"accounts": []map[string]string{
					{"user": u1.Username, "pass": u1.Password, "role": "admin"},
					{"user": u2.Username, "pass": u2.Password, "role": "deployer"},
				},
			},
			"database": map[string]any{
				"host": "db-master." + manifest.Domain, "port": 5432, "name": dbName,
			},
			"redis": map[string]any{
				"host": "redis." + manifest.Domain, "port": 6379,
			},
		},
		"api_key": apiKey,
	}, "", "  ")
	planted = append(planted, writeBaitFile(tmpDir, "api/internal/config", string(internalCfg))...)

	// 7) /admin/notes.txt — ops notes with plaintext passwords
	os.MkdirAll(filepath.Join(tmpDir, "admin"), 0755)
	adminNotes := fmt.Sprintf(
		"%s — Operations Notes\n"+
			"Last updated: %s\n"+
			"────────────────────────────────\n\n"+
			"SSH Access:\n"+
			"  ssh %s@%s -p %s\n"+
			"  Password: %s\n\n"+
			"Backup account:\n"+
			"  ssh %s@%s -p %s\n"+
			"  Password: %s\n\n"+
			"TODO:\n"+
			"  - Rotate credentials (overdue)\n"+
			"  - Move to key-based auth\n"+
			"  - Restrict .env exposure\n",
		manifest.Company, manifest.CreatedAt[:10],
		u1.Username, sshTarget, sshPort, u1.Password,
		u2.Username, sshTarget, sshPort, u2.Password,
	)
	planted = append(planted, writeBaitFile(tmpDir, "admin/notes.txt", adminNotes)...)

	// Copy all bait files into the HTTP container
	// First ensure the bait directory exists
	exec.Command("docker", "exec", "labyrinth-http",
		"mkdir", "-p", "/var/labyrinth/bait/web").Run()

	// docker cp the temp dir contents into the container
	copyCmd := exec.Command("docker", "cp", tmpDir+"/.", "labyrinth-http:/var/labyrinth/bait/web/")
	if out, err := copyCmd.CombinedOutput(); err != nil {
		errMsg(fmt.Sprintf("Failed to copy bait files: %s", strings.TrimSpace(string(out))))
		return planted
	}

	info(fmt.Sprintf("Planted %d web bait files", len(planted)))

	// Restart the HTTP container so the server process picks up new bait files
	// and regenerates its anti-fingerprinting identity
	info("Restarting HTTP portal trap...")
	if err := exec.Command("docker", "restart", "labyrinth-http").Run(); err != nil {
		warn(fmt.Sprintf("Could not restart labyrinth-http: %v", err))
	}

	return planted
}

func writeBaitFile(tmpDir, relPath, content string) []string {
	fullPath := filepath.Join(tmpDir, relPath)
	os.MkdirAll(filepath.Dir(fullPath), 0755)
	os.WriteFile(fullPath, []byte(content), 0644)
	return []string{"/" + relPath}
}

func removeWebBait() {
	exec.Command("docker", "exec", "labyrinth-http",
		"rm", "-rf", "/var/labyrinth/bait/web").Run()
	info("Removed web bait files")
}

// ── Manifest Persistence ────────────────────────────────────

func baitManifestPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".labyrinth", "bait.json")
}

func saveBaitManifest(manifest BaitManifest) error {
	dir := filepath.Dir(baitManifestPath())
	os.MkdirAll(dir, 0755)
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(baitManifestPath(), data, 0600)
}

func loadBaitManifest() *BaitManifest {
	data, err := os.ReadFile(baitManifestPath())
	if err != nil {
		return nil
	}
	var m BaitManifest
	if json.Unmarshal(data, &m) != nil {
		return nil
	}
	return &m
}

func removeBaitManifest() {
	os.Remove(baitManifestPath())
}

// ── Helpers ─────────────────────────────────────────────────

func containerRunning(name string) bool {
	out, err := exec.Command("docker", "ps", "-q", "--filter", "name=^/"+name+"$").Output()
	return err == nil && len(strings.TrimSpace(string(out))) > 0
}

func generateUsers(count int) []BaitUser {
	used := map[string]bool{"admin": true, "root": true}
	var users []BaitUser
	for i := 0; i < count; i++ {
		var username string
		for {
			first := pick(firstNames)
			last := pick(lastNames)
			// Generate username in format: first initial + last name (e.g. jchen)
			username = string(first[0]) + last
			if !used[username] {
				used[username] = true
				break
			}
		}
		users = append(users, BaitUser{
			Username: username,
			Password: randomPassword(),
		})
	}
	return users
}

func randomPassword() string {
	word := pick(passwordWords)
	n, _ := rand.Int(rand.Reader, big.NewInt(900))
	num := n.Int64() + 100
	symbols := []string{"!", "@", "#", "$", "%"}
	sym := pick(symbols)
	return fmt.Sprintf("%s%d%s", word, num, sym)
}

func pick[T any](pool []T) T {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(pool))))
	return pool[n.Int64()]
}

func randomHex(bytes int) string {
	b := make([]byte, bytes)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func randomBase64(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	result := make([]byte, length)
	for i, v := range b {
		result[i] = charset[int(v)%len(charset)]
	}
	return string(result)
}
