package report

import (
	"fmt"
	"strings"
)

// RenderMarkdown produces a Markdown report suitable for GitHub/wikis/sharing.
func RenderMarkdown(r *Report) string {
	var b strings.Builder

	b.WriteString("# LABYRINTH Forensic Report\n\n")
	b.WriteString(fmt.Sprintf("**Session:** `%s`  \n", r.SessionID))
	b.WriteString(fmt.Sprintf("**Generated:** %s\n\n", r.GeneratedAt))

	// Executive Summary
	b.WriteString("## Executive Summary\n\n")
	b.WriteString("| Metric | Value |\n")
	b.WriteString("|--------|-------|\n")
	b.WriteString(fmt.Sprintf("| Duration | %s |\n", r.Summary.Duration))
	b.WriteString(fmt.Sprintf("| Attacker Type | %s |\n", r.Summary.AttackerType))
	b.WriteString(fmt.Sprintf("| Layers Reached | %s |\n", formatLayersMD(r.Summary.LayersReached)))
	b.WriteString(fmt.Sprintf("| Max Depth | %d |\n", r.Summary.MaxDepth))
	b.WriteString(fmt.Sprintf("| Confusion Score | %d/100 |\n", r.Summary.ConfusionScore))
	b.WriteString(fmt.Sprintf("| Risk Level | **%s** |\n", r.Summary.RiskLevel))
	b.WriteString(fmt.Sprintf("| Total Events | %d |\n", r.Summary.TotalEvents))
	b.WriteString(fmt.Sprintf("| L3 Activated | %s |\n", boolStr(r.Summary.L3Activated)))
	b.WriteString(fmt.Sprintf("| L4 Active | %s |\n", boolStr(r.Summary.L4Active)))
	b.WriteString(fmt.Sprintf("| First Seen | %s |\n", r.Summary.FirstSeen))
	b.WriteString(fmt.Sprintf("| Last Seen | %s |\n", r.Summary.LastSeen))
	b.WriteString("\n")

	// Attack Timeline
	b.WriteString("## Attack Timeline (MITRE ATT&CK)\n\n")
	b.WriteString("| Time | Layer | Event | Description | Tactic | Technique |\n")
	b.WriteString("|------|-------|-------|-------------|--------|-----------|\n")
	for _, entry := range r.Timeline {
		ts := formatTimestamp(entry.Timestamp)
		desc := escMD(entry.Description)
		if len(desc) > 50 {
			desc = desc[:47] + "..."
		}
		b.WriteString(fmt.Sprintf("| %s | L%d | %s | %s | %s | %s |\n",
			ts, entry.Layer, entry.Event, desc, entry.MITRETactic, entry.MITRETechID))
	}
	b.WriteString("\n")

	// Credentials
	b.WriteString("## Credentials\n\n")
	if len(r.Credentials.BaitCreds) > 0 {
		b.WriteString("### Planted Credentials\n\n")
		b.WriteString("| Service | Username | Status |\n")
		b.WriteString("|---------|----------|--------|\n")
		for _, bc := range r.Credentials.BaitCreds {
			status := "Not Used"
			if bc.WasUsed {
				status = "**USED**"
			}
			b.WriteString(fmt.Sprintf("| %s | %s | %s |\n", bc.Service, bc.Username, status))
		}
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("- **Matched bait credentials:** %d\n", r.Credentials.MatchedBait))
	b.WriteString(fmt.Sprintf("- **Novel attempts:** %d\n", r.Credentials.NovelAttempts))
	b.WriteString(fmt.Sprintf("- **Total auth events captured:** %d\n\n", len(r.Credentials.CapturedAuth)))

	// Services Explored
	if len(r.Services) > 0 {
		b.WriteString("## Services Explored\n\n")
		b.WriteString("| Service | Port | Connections | Auth Attempts | Queries | Sample Query |\n")
		b.WriteString("|---------|------|-------------|---------------|---------|--------------|\n")
		for _, svc := range r.Services {
			sample := "—"
			if len(svc.SampleQueries) > 0 {
				sample = "`" + escMD(svc.SampleQueries[0]) + "`"
				if len(sample) > 40 {
					sample = sample[:37] + "...`"
				}
			}
			b.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %d | %s |\n",
				svc.Protocol, svc.Port, svc.Connections, svc.AuthAttempts, svc.Queries, sample))
		}
		b.WriteString("\n")
	}

	// Tools Analysis
	if r.Tools.UserAgent != "" || len(r.Tools.ToolInventory) > 0 {
		b.WriteString("## Tools Analysis\n\n")
		if r.Tools.UserAgent != "" {
			b.WriteString(fmt.Sprintf("- **User Agent:** `%s`\n", r.Tools.UserAgent))
		}
		if r.Tools.SDKDetected != "" {
			b.WriteString(fmt.Sprintf("- **SDK Detected:** %s\n", r.Tools.SDKDetected))
		}
		if len(r.Tools.APIKeys) > 0 {
			b.WriteString(fmt.Sprintf("- **API Keys:** `%s`\n", strings.Join(r.Tools.APIKeys, "`, `")))
		}
		if len(r.Tools.Models) > 0 {
			b.WriteString(fmt.Sprintf("- **Models:** %s\n", strings.Join(r.Tools.Models, ", ")))
		}
		if r.Tools.ToolCount > 0 {
			b.WriteString(fmt.Sprintf("- **Tool Count:** %d\n", r.Tools.ToolCount))
		}
		if len(r.Tools.ToolInventory) > 0 {
			b.WriteString("\n### Top Commands\n\n")
			b.WriteString("| Command | Count |\n")
			b.WriteString("|---------|-------|\n")
			limit := 15
			if len(r.Tools.ToolInventory) < limit {
				limit = len(r.Tools.ToolInventory)
			}
			for _, t := range r.Tools.ToolInventory[:limit] {
				b.WriteString(fmt.Sprintf("| `%s` | %d |\n", t.Name, t.Count))
			}
		}
		b.WriteString("\n")
	}

	// Captured Prompts
	if len(r.Prompts) > 0 {
		b.WriteString(fmt.Sprintf("## Captured Prompts (%d)\n\n", len(r.Prompts)))
		for i, p := range r.Prompts {
			b.WriteString(fmt.Sprintf("### Prompt %d\n\n", i+1))
			b.WriteString(fmt.Sprintf("- **Timestamp:** %s\n", p.Timestamp))
			b.WriteString(fmt.Sprintf("- **Domain:** %s\n\n", p.Domain))
			b.WriteString("```\n")
			b.WriteString(p.Text)
			b.WriteString("\n```\n\n")
		}
	}

	// Attack Graph
	b.WriteString("## Attack Graph\n\n")
	b.WriteString("```mermaid\n")
	b.WriteString(r.AttackGraph)
	b.WriteString("```\n\n")

	// Effectiveness Assessment
	b.WriteString("## Effectiveness Assessment\n\n")
	if len(r.Effectiveness.DeceptionWorked) > 0 {
		b.WriteString("### Deception Successes\n\n")
		for _, item := range r.Effectiveness.DeceptionWorked {
			b.WriteString(fmt.Sprintf("- [x] %s\n", item))
		}
		b.WriteString("\n")
	}
	if len(r.Effectiveness.DeceptionFailed) > 0 {
		b.WriteString("### Deception Gaps\n\n")
		for _, item := range r.Effectiveness.DeceptionFailed {
			b.WriteString(fmt.Sprintf("- [ ] %s\n", item))
		}
		b.WriteString("\n")
	}
	if len(r.Effectiveness.IntelligenceGained) > 0 {
		b.WriteString("### Intelligence Captured\n\n")
		for _, item := range r.Effectiveness.IntelligenceGained {
			b.WriteString(fmt.Sprintf("- %s\n", item))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func formatLayersMD(layers []int) string {
	if len(layers) == 0 {
		return "None"
	}
	var parts []string
	for _, l := range layers {
		parts = append(parts, fmt.Sprintf("L%d", l))
	}
	return strings.Join(parts, " → ")
}

func boolStr(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func escMD(s string) string {
	s = strings.ReplaceAll(s, "|", "\\|")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}
