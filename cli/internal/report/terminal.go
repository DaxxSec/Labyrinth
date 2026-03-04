package report

import (
	"fmt"
	"strings"
)

// ANSI helpers (matching root.go style)
const (
	green   = "\033[0;32m"
	yellow  = "\033[1;33m"
	red     = "\033[0;31m"
	magenta = "\033[0;35m"
	cyan    = "\033[0;36m"
	bold    = "\033[1m"
	dim     = "\033[2m"
	reset   = "\033[0m"
)

// RenderTerminal produces an ANSI-formatted report for the terminal.
func RenderTerminal(r *Report) string {
	var b strings.Builder

	// Header
	b.WriteString(fmt.Sprintf("\n  %s━━━ LABYRINTH Forensic Report ━━━%s\n", magenta, reset))
	b.WriteString(fmt.Sprintf("  Session: %s%s%s | Generated: %s%s%s\n\n",
		bold, r.SessionID, reset, dim, r.GeneratedAt, reset))

	// Executive Summary
	b.WriteString(fmt.Sprintf("  %s━━━ Executive Summary ━━━%s\n\n", magenta, reset))
	b.WriteString(fmt.Sprintf("  %sDuration:%s    %s\n", bold, reset, r.Summary.Duration))
	b.WriteString(fmt.Sprintf("  %sAttacker:%s    %s\n", bold, reset, r.Summary.AttackerType))
	b.WriteString(fmt.Sprintf("  %sLayers:%s      %s\n", bold, reset, formatLayers(r.Summary.LayersReached, r.Summary.L3Activated)))
	b.WriteString(fmt.Sprintf("  %sMax Depth:%s   %d\n", bold, reset, r.Summary.MaxDepth))
	b.WriteString(fmt.Sprintf("  %sConfusion:%s   %s %d/100\n", bold, reset, confusionBar(r.Summary.ConfusionScore), r.Summary.ConfusionScore))
	b.WriteString(fmt.Sprintf("  %sRisk Level:%s  %s\n", bold, reset, colorRiskLevel(r.Summary.RiskLevel)))
	b.WriteString("\n")

	// Attack Timeline
	b.WriteString(fmt.Sprintf("  %s━━━ Attack Timeline (MITRE ATT&CK) ━━━%s\n\n", magenta, reset))
	b.WriteString(fmt.Sprintf("  %s%-10s %-5s %-20s %-20s %-10s%s\n",
		dim, "TIME", "LAYER", "EVENT", "TACTIC", "TECHNIQUE", reset))
	b.WriteString(fmt.Sprintf("  %s%s%s\n", dim, strings.Repeat("─", 70), reset))

	displayed := 0
	maxDisplay := 20
	for _, entry := range r.Timeline {
		if displayed >= maxDisplay {
			remaining := len(r.Timeline) - maxDisplay
			if remaining > 0 {
				b.WriteString(fmt.Sprintf("  %s...        ...   (%d more events)%s\n", dim, remaining, reset))
			}
			break
		}
		ts := formatTimestamp(entry.Timestamp)
		layer := fmt.Sprintf("L%d", entry.Layer)
		event := truncate(entry.Event, 18)
		tactic := truncate(entry.MITRETactic, 18)
		tech := entry.MITRETechID

		layerColor := layerColorCode(entry.Layer)
		b.WriteString(fmt.Sprintf("  %-10s %s%-5s%s %-20s %-20s %-10s\n",
			ts, layerColor, layer, reset, event, tactic, tech))
		displayed++
	}
	b.WriteString("\n")

	// Credentials
	b.WriteString(fmt.Sprintf("  %s━━━ Credentials ━━━%s\n\n", magenta, reset))
	if len(r.Credentials.BaitCreds) > 0 {
		b.WriteString(fmt.Sprintf("  %s%-6s %-25s %s%s\n", dim, "SVC", "PLANTED", "STATUS", reset))
		for _, bc := range r.Credentials.BaitCreds {
			status := fmt.Sprintf("%s[NOT USED]%s", dim, reset)
			if bc.WasUsed {
				status = fmt.Sprintf("%s[USED ✓]%s", green, reset)
			}
			b.WriteString(fmt.Sprintf("  %-6s %-25s %s\n", bc.Service, bc.Username, status))
		}
	}
	if len(r.Credentials.CapturedAuth) > 0 {
		b.WriteString(fmt.Sprintf("\n  Captured auth attempts: %s%d%s", bold, len(r.Credentials.CapturedAuth), reset))
		b.WriteString(fmt.Sprintf("  |  Novel attempts: %s%d%s\n", bold, r.Credentials.NovelAttempts, reset))
	}
	b.WriteString("\n")

	// Services Explored
	if len(r.Services) > 0 {
		b.WriteString(fmt.Sprintf("  %s━━━ Services Explored ━━━%s\n\n", magenta, reset))
		b.WriteString(fmt.Sprintf("  %s%-14s %-6s %-5s %-5s %-8s %s%s\n",
			dim, "SERVICE", "PORT", "CONN", "AUTH", "QUERIES", "SAMPLE", reset))
		for _, svc := range r.Services {
			sample := "—"
			if len(svc.SampleQueries) > 0 {
				sample = truncate(svc.SampleQueries[0], 30)
			}
			b.WriteString(fmt.Sprintf("  %-14s %-6d %-5d %-5d %-8d %s\n",
				svc.Protocol, svc.Port, svc.Connections, svc.AuthAttempts, svc.Queries, sample))
		}
		b.WriteString("\n")
	}

	// Tools Analysis
	if r.Tools.UserAgent != "" || len(r.Tools.ToolInventory) > 0 {
		b.WriteString(fmt.Sprintf("  %s━━━ Tools Analysis ━━━%s\n\n", magenta, reset))
		if r.Tools.UserAgent != "" {
			b.WriteString(fmt.Sprintf("  %sUser Agent:%s  %s\n", bold, reset, r.Tools.UserAgent))
		}
		if r.Tools.SDKDetected != "" {
			b.WriteString(fmt.Sprintf("  %sSDK:%s         %s\n", bold, reset, r.Tools.SDKDetected))
		}
		if len(r.Tools.APIKeys) > 0 {
			b.WriteString(fmt.Sprintf("  %sAPI Keys:%s    %s\n", bold, reset, strings.Join(r.Tools.APIKeys, ", ")))
		}
		if len(r.Tools.Models) > 0 {
			b.WriteString(fmt.Sprintf("  %sModels:%s      %s\n", bold, reset, strings.Join(r.Tools.Models, ", ")))
		}
		if len(r.Tools.ToolInventory) > 0 {
			b.WriteString(fmt.Sprintf("\n  %sTop commands:%s\n", bold, reset))
			limit := 10
			if len(r.Tools.ToolInventory) < limit {
				limit = len(r.Tools.ToolInventory)
			}
			for _, t := range r.Tools.ToolInventory[:limit] {
				b.WriteString(fmt.Sprintf("    %-20s %dx\n", t.Name, t.Count))
			}
		}
		b.WriteString("\n")
	}

	// Captured Prompts
	if len(r.Prompts) > 0 {
		b.WriteString(fmt.Sprintf("  %s━━━ Captured Prompts (%d) ━━━%s\n\n", magenta, len(r.Prompts), reset))
		for i, p := range r.Prompts {
			b.WriteString(fmt.Sprintf("  %s[%d]%s %s | %s\n", cyan, i+1, reset, p.Timestamp, p.Domain))
			text := p.Text
			if len(text) > 200 {
				text = text[:197] + "..."
			}
			b.WriteString(fmt.Sprintf("  %s%s%s\n\n", dim, text, reset))
		}
	}

	// Attack Graph
	b.WriteString(fmt.Sprintf("  %s━━━ Attack Graph ━━━%s\n\n", magenta, reset))
	for _, line := range strings.Split(r.AttackGraph, "\n") {
		b.WriteString(fmt.Sprintf("  %s%s%s\n", dim, line, reset))
	}
	b.WriteString("\n")

	// Effectiveness Assessment
	b.WriteString(fmt.Sprintf("  %s━━━ Effectiveness Assessment ━━━%s\n\n", magenta, reset))
	for _, item := range r.Effectiveness.DeceptionWorked {
		b.WriteString(fmt.Sprintf("  %s[+]%s %s\n", green, reset, item))
	}
	for _, item := range r.Effectiveness.DeceptionFailed {
		b.WriteString(fmt.Sprintf("  %s[!]%s %s\n", yellow, reset, item))
	}
	if len(r.Effectiveness.IntelligenceGained) > 0 {
		b.WriteString(fmt.Sprintf("\n  %sIntelligence Captured:%s\n", bold, reset))
		for _, item := range r.Effectiveness.IntelligenceGained {
			b.WriteString(fmt.Sprintf("  %s[+]%s %s\n", green, reset, item))
		}
	}
	b.WriteString("\n")

	return b.String()
}

func confusionBar(score int) string {
	filled := score / 5 // 20 chars total
	if filled > 20 {
		filled = 20
	}
	empty := 20 - filled

	color := green
	if score >= 60 {
		color = red
	} else if score >= 30 {
		color = yellow
	}

	return fmt.Sprintf("%s%s%s%s", color, strings.Repeat("█", filled), reset+dim, strings.Repeat("░", empty))
}

func colorRiskLevel(level string) string {
	switch level {
	case "Critical":
		return fmt.Sprintf("%s%sCRITICAL%s", bold, red, reset)
	case "High":
		return fmt.Sprintf("%sHIGH%s", red, reset)
	case "Medium":
		return fmt.Sprintf("%sMEDIUM%s", yellow, reset)
	case "Low":
		return fmt.Sprintf("%sLOW%s", green, reset)
	default:
		return level
	}
}

func formatLayers(layers []int, l3 bool) string {
	if len(layers) == 0 {
		return "None"
	}

	var parts []string
	for _, l := range layers {
		parts = append(parts, fmt.Sprintf("L%d", l))
	}
	result := strings.Join(parts, " → ")

	// Note missing layers
	layerSet := make(map[int]bool)
	for _, l := range layers {
		layerSet[l] = true
	}
	if !layerSet[3] && !l3 {
		result += fmt.Sprintf("  %s(L3 not reached)%s", dim, reset)
	}

	return result
}

func formatTimestamp(ts string) string {
	// Extract just the time portion HH:MM:SS from ISO timestamp
	if len(ts) >= 19 {
		return ts[11:19]
	}
	return ts
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-1] + "…"
	}
	return s
}

func layerColorCode(layer int) string {
	switch layer {
	case 1:
		return cyan
	case 2:
		return magenta
	case 3:
		return yellow
	case 4:
		return red
	default:
		return ""
	}
}
