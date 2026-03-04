package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/DaxxSec/labyrinth/cli/internal/api"
	"github.com/DaxxSec/labyrinth/cli/internal/forensics"
)

// Build assembles a full Report for the given session. Exactly one of client
// or reader should be non-nil.
func Build(client *api.Client, reader *forensics.Reader, sessionID string) (*Report, error) {
	r := &Report{
		SessionID:   sessionID,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// 1. Fetch session detail (events, layers, depth, timestamps)
	detail, err := fetchDetail(client, reader, sessionID)
	if err != nil {
		return nil, fmt.Errorf("fetch session detail: %w", err)
	}

	// 2. Fetch or compute analysis (confusion score, phases)
	analysis := fetchOrComputeAnalysis(client, detail, sessionID)

	// 3. Fetch auth events
	authEvents := fetchAuthEvents(client, reader, detail)

	// 4. Fetch bait identity
	bait := fetchBaitIdentity(client, reader)

	// 5. Fetch L4 intel
	intel := fetchL4Intel(client, reader, sessionID)

	// 6. Fetch captured prompts
	prompts := fetchPrompts(client, reader, sessionID)
	r.Prompts = prompts

	// Build executive summary
	r.Summary = buildSummary(detail, analysis, prompts, intel)

	// Build timeline with MITRE mapping
	r.Timeline = buildTimeline(detail)

	// Build credential report
	r.Credentials = buildCredentialReport(authEvents, bait)

	// Build tools analysis
	r.Tools = buildToolsAnalysis(detail, intel)

	// Build service interactions
	r.Services = buildServiceInteractions(detail)

	// Generate attack graph
	r.AttackGraph = GenerateGraph(detail, r.Services)

	// Build effectiveness assessment
	r.Effectiveness = buildEffectiveness(r)

	return r, nil
}

func fetchDetail(client *api.Client, reader *forensics.Reader, sessionID string) (*api.SessionDetail, error) {
	if client != nil {
		return client.FetchSessionDetail(sessionID)
	}
	return reader.ReadSessionDetail(sessionID)
}

func fetchOrComputeAnalysis(client *api.Client, detail *api.SessionDetail, sessionID string) *api.SessionAnalysis {
	if client != nil {
		a, err := client.FetchSessionAnalysis(sessionID)
		if err == nil {
			return a
		}
	}
	// Compute locally from events
	return computeAnalysis(detail)
}

func computeAnalysis(detail *api.SessionDetail) *api.SessionAnalysis {
	a := &api.SessionAnalysis{
		SessionID:     detail.SessionID,
		TotalEvents:   len(detail.Events),
		LayersReached: detail.LayersTriggered,
		MaxDepth:      detail.MaxDepth,
		L3Activated:   detail.L3Activated,
	}

	if len(detail.Events) >= 2 {
		first, _ := time.Parse(time.RFC3339Nano, detail.FirstSeen)
		last, _ := time.Parse(time.RFC3339Nano, detail.LastSeen)
		if !first.IsZero() && !last.IsZero() {
			a.DurationSeconds = last.Sub(first).Seconds()
		}
	}

	// Confusion score — ported from dashboard/app.py
	a.ConfusionScore = computeConfusionScore(detail.Events)

	// Check L4 activity
	for _, ev := range detail.Events {
		switch ev.Event {
		case "api_intercepted", "service_connection", "service_auth", "service_query":
			a.L4Active = true
		}
	}

	// Event breakdown
	a.EventBreakdown = make(map[string]int)
	for _, ev := range detail.Events {
		a.EventBreakdown[ev.Event]++
	}

	return a
}

func computeConfusionScore(events []api.ForensicEvent) int {
	score := 0
	commandCounts := make(map[string]int)
	authCount := 0
	depthChanges := 0
	prevDepth := 0.0
	pathCounts := make(map[string]int)

	for _, ev := range events {
		data := ev.Data
		if data == nil {
			data = make(map[string]interface{})
		}

		switch ev.Event {
		case "command":
			if cmd, ok := data["command"].(string); ok {
				commandCounts[cmd]++
			}
		case "auth_attempt", "auth":
			authCount++
		case "depth_increase":
			depthChanges++
			if newDepth, ok := data["new_depth"].(float64); ok {
				if newDepth < prevDepth {
					score += 5
				}
				prevDepth = newDepth
			}
		case "http_access":
			if path, ok := data["path"].(string); ok {
				pathCounts[path]++
			}
		}
	}

	// Repeated commands (same command > 3 times)
	repeatedCmds := 0
	for _, c := range commandCounts {
		if c > 3 {
			repeatedCmds++
		}
	}
	score += min(repeatedCmds*8, 30)

	// Auth loops
	if authCount > 5 {
		score += min((authCount-5)*3, 15)
	}

	// Depth oscillation bonus
	if depthChanges > 5 {
		score += min((depthChanges-5)*2, 15)
	}

	// Repeated path fetches
	repeatedPaths := 0
	for _, c := range pathCounts {
		if c > 3 {
			repeatedPaths++
		}
	}
	score += min(repeatedPaths*5, 20)

	// Blindfold activation
	for _, ev := range events {
		if ev.Event == "encoding_activated" || ev.Event == "blindfold_activated" {
			score += 15
			break
		}
	}

	// API interception
	for _, ev := range events {
		if ev.Event == "api_intercepted" {
			score += 10
			break
		}
	}

	// Service credential use
	for _, ev := range events {
		if ev.Event == "service_auth" {
			score += 15
			break
		}
	}

	if score > 100 {
		score = 100
	}
	return score
}

func fetchAuthEvents(client *api.Client, reader *forensics.Reader, detail *api.SessionDetail) []api.AuthEvent {
	var all []api.AuthEvent

	if client != nil {
		events, err := client.FetchAuthEvents(1000)
		if err == nil {
			all = events
		}
	} else if reader != nil {
		events, err := reader.ReadAuthEvents()
		if err == nil {
			all = events
		}
	}

	// Filter to session's source IP
	srcIP := ""
	for _, ev := range detail.Events {
		if ev.Event == "connection" {
			if ip, ok := ev.Data["src_ip"].(string); ok {
				srcIP = ip
				break
			}
		}
	}
	if srcIP == "" {
		return all
	}

	var filtered []api.AuthEvent
	for _, ae := range all {
		if ae.SrcIP == srcIP {
			filtered = append(filtered, ae)
		}
	}
	return filtered
}

func fetchBaitIdentity(client *api.Client, reader *forensics.Reader) *api.BaitIdentity {
	if client != nil {
		bait, err := client.FetchBaitIdentity()
		if err == nil {
			return bait
		}
	}
	// File-based fallback: read config.json
	if reader != nil {
		configPath := "/var/labyrinth/forensics/config.json"
		data, err := os.ReadFile(configPath)
		if err == nil {
			var bait api.BaitIdentity
			if json.Unmarshal(data, &bait) == nil {
				return &bait
			}
		}
	}
	return nil
}

func fetchL4Intel(client *api.Client, reader *forensics.Reader, sessionID string) *api.L4IntelSummary {
	if client != nil {
		intelList, err := client.FetchL4Intel()
		if err == nil {
			for _, i := range intelList {
				if i.SessionID == sessionID {
					return &i
				}
			}
			// If no session-specific match, return first if available
			if len(intelList) > 0 {
				return &intelList[0]
			}
		}
	}
	// File-based fallback: read intel/{session_id}.json
	if reader != nil {
		intelPath := filepath.Join("/var/labyrinth/forensics", "intel", sessionID+".json")
		data, err := os.ReadFile(intelPath)
		if err == nil {
			var intel api.L4IntelSummary
			if json.Unmarshal(data, &intel) == nil {
				return &intel
			}
		}
	}
	return nil
}

func fetchPrompts(client *api.Client, reader *forensics.Reader, sessionID string) []api.CapturedPrompt {
	var all []api.CapturedPrompt
	if client != nil {
		p, err := client.FetchPrompts()
		if err == nil {
			all = p
		}
	} else if reader != nil {
		p, err := reader.ReadPrompts()
		if err == nil {
			all = p
		}
	}

	var filtered []api.CapturedPrompt
	for _, p := range all {
		if p.SessionID == sessionID {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func buildSummary(detail *api.SessionDetail, analysis *api.SessionAnalysis, prompts []api.CapturedPrompt, intel *api.L4IntelSummary) ExecutiveSummary {
	s := ExecutiveSummary{
		DurationSecs:  analysis.DurationSeconds,
		Duration:      formatDuration(analysis.DurationSeconds),
		LayersReached: analysis.LayersReached,
		MaxDepth:      analysis.MaxDepth,
		ConfusionScore: analysis.ConfusionScore,
		TotalEvents:   analysis.TotalEvents,
		L3Activated:   analysis.L3Activated,
		L4Active:      analysis.L4Active,
		FirstSeen:     detail.FirstSeen,
		LastSeen:      detail.LastSeen,
	}

	// Risk level
	switch {
	case analysis.ConfusionScore >= 60 && analysis.L4Active:
		s.RiskLevel = "Critical"
	case analysis.ConfusionScore >= 40 || analysis.L3Activated:
		s.RiskLevel = "High"
	case analysis.ConfusionScore >= 20:
		s.RiskLevel = "Medium"
	default:
		s.RiskLevel = "Low"
	}

	// Attacker type
	hasAPIKeys := intel != nil && len(intel.APIKeys) > 0
	hasPrompts := len(prompts) > 0
	if hasPrompts || hasAPIKeys {
		s.AttackerType = "AI Agent"
		if intel != nil && intel.UserAgent != "" {
			// Try to extract agent name from user agent
			ua := intel.UserAgent
			if idx := strings.Index(ua, "/"); idx > 0 {
				s.AttackerType = fmt.Sprintf("AI Agent (%s)", ua[:idx])
			}
		}
	} else if analysis.TotalEvents > 0 && analysis.DurationSeconds > 0 {
		velocity := float64(analysis.TotalEvents) / analysis.DurationSeconds
		if velocity > 10 {
			s.AttackerType = "Automated Scanner"
		} else {
			s.AttackerType = "Human/Unknown"
		}
	} else {
		s.AttackerType = "Human/Unknown"
	}

	return s
}

func buildTimeline(detail *api.SessionDetail) []TimelineEntry {
	var timeline []TimelineEntry
	for _, ev := range detail.Events {
		tactic, techID := MapEvent(ev.Event, ev.Data)
		entry := TimelineEntry{
			Timestamp:   ev.Timestamp,
			Layer:       ev.Layer,
			Event:       ev.Event,
			Description: describeEvent(ev),
			MITRETactic: tactic,
			MITRETechID: techID,
			Data:        ev.Data,
		}
		timeline = append(timeline, entry)
	}
	return timeline
}

func describeEvent(ev api.ForensicEvent) string {
	data := ev.Data
	if data == nil {
		data = make(map[string]interface{})
	}

	switch ev.Event {
	case "connection":
		ip, _ := data["src_ip"].(string)
		return fmt.Sprintf("SSH connection from %s", ip)
	case "auth":
		user, _ := data["username"].(string)
		return fmt.Sprintf("Authentication as %s", user)
	case "container_spawned":
		depth, _ := data["depth"].(float64)
		return fmt.Sprintf("Container spawned at depth %d", int(depth))
	case "depth_increase":
		nd, _ := data["new_depth"].(float64)
		return fmt.Sprintf("Depth increased to %d", int(nd))
	case "command":
		cmd, _ := data["command"].(string)
		if len(cmd) > 80 {
			cmd = cmd[:77] + "..."
		}
		return cmd
	case "blindfold_activated":
		return "L3 BLINDFOLD activated — output encoding enabled"
	case "service_connection":
		proto, _ := data["protocol"].(string)
		port, _ := data["port"].(float64)
		return fmt.Sprintf("%s connection to port %d", proto, int(port))
	case "service_auth":
		proto, _ := data["protocol"].(string)
		user, _ := data["username"].(string)
		return fmt.Sprintf("%s auth as %s", proto, user)
	case "service_query":
		proto, _ := data["protocol"].(string)
		query, _ := data["query"].(string)
		if len(query) > 60 {
			query = query[:57] + "..."
		}
		return fmt.Sprintf("%s query: %s", proto, query)
	case "api_intercepted":
		domain, _ := data["domain"].(string)
		return fmt.Sprintf("API call intercepted (%s)", domain)
	case "http_access":
		path, _ := data["path"].(string)
		method, _ := data["method"].(string)
		return fmt.Sprintf("%s %s", method, path)
	default:
		return ev.Event
	}
}

func buildCredentialReport(authEvents []api.AuthEvent, bait *api.BaitIdentity) CredentialReport {
	cr := CredentialReport{
		CapturedAuth: authEvents,
	}

	if bait == nil {
		cr.NovelAttempts = len(authEvents)
		return cr
	}

	// Build set of bait usernames
	baitUsers := make(map[string]bool)
	for _, u := range bait.Users {
		baitUsers[u.Uname] = true
		cr.BaitCreds = append(cr.BaitCreds, BaitCredStatus{
			Service:  "SSH",
			Username: u.Uname,
		})
	}

	// Add service credentials as bait
	if bait.DBPass != "" {
		cr.BaitCreds = append(cr.BaitCreds, BaitCredStatus{
			Service:  "PostgreSQL",
			Username: "admin",
		})
	}
	if bait.RedisToken != "" {
		cr.BaitCreds = append(cr.BaitCreds, BaitCredStatus{
			Service:  "Redis",
			Username: "(token)",
		})
	}

	// Match auth events against bait
	matched := make(map[string]bool)
	for _, ae := range authEvents {
		if baitUsers[ae.Username] {
			matched[ae.Username] = true
		}
		// Check if password matches service creds
		if ae.Password == bait.DBPass || ae.Password == bait.RedisToken {
			key := ae.Service + ":" + ae.Password
			matched[key] = true
		}
	}

	// Update BaitCreds with usage status
	for i := range cr.BaitCreds {
		bc := &cr.BaitCreds[i]
		if bc.Service == "SSH" && matched[bc.Username] {
			bc.WasUsed = true
			cr.MatchedBait++
		} else if bc.Service == "PostgreSQL" && matched["PostgreSQL:"+bait.DBPass] {
			bc.WasUsed = true
			cr.MatchedBait++
		} else if bc.Service == "Redis" && matched["Redis:"+bait.RedisToken] {
			bc.WasUsed = true
			cr.MatchedBait++
		}
	}

	cr.NovelAttempts = len(authEvents) - cr.MatchedBait

	return cr
}

func buildToolsAnalysis(detail *api.SessionDetail, intel *api.L4IntelSummary) ToolsAnalysis {
	ta := ToolsAnalysis{}

	if intel != nil {
		ta.UserAgent = intel.UserAgent
		ta.SDKDetected = parseSDK(intel.UserAgent)
		ta.Models = intel.Models
		ta.ToolCount = intel.ToolCount
		ta.Domains = intel.Domains

		// Mask API keys (first 8 chars + ****)
		for _, key := range intel.APIKeys {
			if len(key) > 8 {
				ta.APIKeys = append(ta.APIKeys, key[:8]+"****")
			} else {
				ta.APIKeys = append(ta.APIKeys, key)
			}
		}
	}

	// Analyse commands
	commandCounts := make(map[string]int)
	categories := make(map[string]map[string]int) // category -> pattern -> count

	for _, ev := range detail.Events {
		if ev.Event != "command" {
			continue
		}
		cmd, _ := ev.Data["command"].(string)
		if cmd == "" {
			continue
		}
		commandCounts[cmd]++

		// Categorise
		cat := categoriseCommand(cmd)
		if categories[cat] == nil {
			categories[cat] = make(map[string]int)
		}
		// Use first word as pattern
		parts := strings.Fields(cmd)
		if len(parts) > 0 {
			categories[cat][parts[0]]++
		}
	}

	// Build tool inventory from command base names
	baseCounts := make(map[string]int)
	for cmd, count := range commandCounts {
		parts := strings.Fields(cmd)
		if len(parts) > 0 {
			baseCounts[parts[0]] += count
		}
	}

	type kv struct {
		name  string
		count int
	}
	var sorted []kv
	for name, count := range baseCounts {
		sorted = append(sorted, kv{name, count})
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].count > sorted[j].count })
	for _, s := range sorted {
		ta.ToolInventory = append(ta.ToolInventory, ToolEntry{Name: s.name, Count: s.count})
	}

	// Build command patterns
	for cat, patterns := range categories {
		for pattern, count := range patterns {
			ta.CommandPatterns = append(ta.CommandPatterns, CommandPattern{
				Pattern:  pattern,
				Count:    count,
				Category: cat,
			})
		}
	}
	sort.Slice(ta.CommandPatterns, func(i, j int) bool {
		return ta.CommandPatterns[i].Count > ta.CommandPatterns[j].Count
	})

	return ta
}

func parseSDK(userAgent string) string {
	lower := strings.ToLower(userAgent)
	switch {
	case strings.Contains(lower, "openai/python"):
		return "OpenAI Python SDK"
	case strings.Contains(lower, "openai-python"):
		return "OpenAI Python SDK"
	case strings.Contains(lower, "anthropic-python"):
		return "Anthropic Python SDK"
	case strings.Contains(lower, "langchain"):
		return "LangChain"
	case strings.Contains(lower, "openai/node"):
		return "OpenAI Node SDK"
	case strings.Contains(lower, "anthropic/node"):
		return "Anthropic Node SDK"
	case strings.Contains(lower, "python-requests"):
		return "Python Requests"
	case strings.Contains(lower, "curl"):
		return "cURL"
	default:
		if userAgent != "" {
			return userAgent
		}
		return ""
	}
}

func categoriseCommand(cmd string) string {
	lower := strings.ToLower(cmd)
	parts := strings.Fields(lower)
	if len(parts) == 0 {
		return "other"
	}

	base := parts[0]

	// Credential access
	if base == "env" || base == "printenv" {
		return "credential_access"
	}
	if (base == "cat" || base == "head" || base == "tail") &&
		(strings.Contains(lower, ".env") || strings.Contains(lower, "secret") ||
			strings.Contains(lower, "password") || strings.Contains(lower, "credential") ||
			strings.Contains(lower, ".aws") || strings.Contains(lower, "shadow") ||
			strings.Contains(lower, ".ssh")) {
		return "credential_access"
	}

	// Lateral movement
	if base == "ssh" || base == "curl" || base == "wget" || base == "psql" ||
		base == "mysql" || base == "redis-cli" || base == "nc" || base == "mongo" {
		return "lateral_movement"
	}

	// Enumeration / discovery
	if base == "ls" || base == "find" || base == "cat" || base == "tree" ||
		base == "whoami" || base == "id" || base == "uname" || base == "hostname" ||
		base == "ifconfig" || base == "ip" || base == "netstat" || base == "ss" ||
		base == "ps" || base == "nmap" || base == "which" || base == "file" ||
		base == "stat" || base == "du" || base == "df" || base == "mount" {
		return "enumeration"
	}

	return "other"
}

func buildServiceInteractions(detail *api.SessionDetail) []ServiceInteraction {
	type svcKey struct {
		protocol string
		port     int
	}

	services := make(map[svcKey]*ServiceInteraction)
	order := []svcKey{}

	for _, ev := range detail.Events {
		if ev.Event != "service_connection" && ev.Event != "service_auth" && ev.Event != "service_query" {
			continue
		}
		proto, _ := ev.Data["protocol"].(string)
		port := 0
		if p, ok := ev.Data["port"].(float64); ok {
			port = int(p)
		}
		key := svcKey{proto, port}

		svc, exists := services[key]
		if !exists {
			svc = &ServiceInteraction{
				Protocol: proto,
				Port:     port,
			}
			services[key] = svc
			order = append(order, key)
		}

		switch ev.Event {
		case "service_connection":
			svc.Connections++
		case "service_auth":
			svc.AuthAttempts++
			if user, ok := ev.Data["username"].(string); ok {
				svc.Credentials = appendUnique(svc.Credentials, user)
			}
		case "service_query":
			svc.Queries++
			if query, ok := ev.Data["query"].(string); ok && len(svc.SampleQueries) < 3 {
				svc.SampleQueries = append(svc.SampleQueries, query)
			}
		}
	}

	var result []ServiceInteraction
	for _, key := range order {
		result = append(result, *services[key])
	}
	return result
}

func buildEffectiveness(r *Report) EffectivenessAssessment {
	ea := EffectivenessAssessment{
		TimeWasted:          r.Summary.Duration,
		CredentialsCaptured: len(r.Credentials.CapturedAuth),
	}

	// Deception that worked
	if r.Credentials.MatchedBait > 0 {
		ea.DeceptionWorked = append(ea.DeceptionWorked,
			fmt.Sprintf("%d/%d planted credential pairs accepted and used",
				r.Credentials.MatchedBait, len(r.Credentials.BaitCreds)))
	}
	if len(r.Services) > 0 {
		engaged := 0
		for _, svc := range r.Services {
			if svc.Connections > 0 || svc.Queries > 0 {
				engaged++
			}
		}
		if engaged > 0 {
			ea.DeceptionWorked = append(ea.DeceptionWorked,
				fmt.Sprintf("%d/%d phantom services engaged attacker", engaged, len(r.Services)))
		}
	}
	if r.Summary.DurationSecs > 60 {
		ea.DeceptionWorked = append(ea.DeceptionWorked,
			fmt.Sprintf("%s of attacker time consumed", r.Summary.Duration))
	}
	if r.Tools.ToolCount > 0 {
		ea.DeceptionWorked = append(ea.DeceptionWorked,
			fmt.Sprintf("Full tool inventory captured (%d tools)", r.Tools.ToolCount))
	}

	// Deception that failed
	if !r.Summary.L3Activated {
		ea.DeceptionFailed = append(ea.DeceptionFailed,
			"L3 BLINDFOLD never triggered (depth stayed at "+fmt.Sprintf("%d", r.Summary.MaxDepth)+")")
	}
	if len(r.Prompts) == 0 && r.Summary.L4Active {
		ea.DeceptionFailed = append(ea.DeceptionFailed,
			"No system prompts captured (MITM rule was gated behind L3)")
	}
	if r.Summary.MaxDepth <= 1 {
		ea.DeceptionFailed = append(ea.DeceptionFailed,
			"Attacker never attempted deeper SSH nesting")
	}

	// Intelligence gained
	if r.Tools.UserAgent != "" {
		ea.IntelligenceGained = append(ea.IntelligenceGained,
			"Attacker user agent identified")
	}
	if len(r.Credentials.CapturedAuth) > 0 {
		ea.IntelligenceGained = append(ea.IntelligenceGained,
			fmt.Sprintf("%d+ credential attempts logged", len(r.Credentials.CapturedAuth)))
	}
	if len(r.Services) > 0 {
		ea.IntelligenceGained = append(ea.IntelligenceGained,
			"Full service query patterns documented")
	}
	if len(r.Tools.APIKeys) > 0 {
		ea.IntelligenceGained = append(ea.IntelligenceGained,
			fmt.Sprintf("%d API key(s) captured", len(r.Tools.APIKeys)))
	}
	if r.Summary.TotalEvents > 10 {
		ea.IntelligenceGained = append(ea.IntelligenceGained,
			"Attack methodology fully reconstructed")
	}

	return ea
}

func formatDuration(secs float64) string {
	if secs <= 0 {
		return "0s"
	}
	d := time.Duration(secs * float64(time.Second))
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func appendUnique(slice []string, val string) []string {
	for _, v := range slice {
		if v == val {
			return slice
		}
	}
	return append(slice, val)
}
