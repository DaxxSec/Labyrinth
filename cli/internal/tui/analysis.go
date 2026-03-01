package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func renderAnalysis(a *App, height int) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(StyleBold.Render("  L4 PUPPETEER Intelligence"))
	b.WriteString("\n\n")

	// Current L4 mode — prominent display
	modeLabel := a.l4Mode
	var modeStyle lipgloss.Style
	modeDesc := ""
	switch a.l4Mode {
	case "passive":
		modeStyle = StyleDim
		modeDesc = "Harvesting intelligence without modification"
	case "neutralize":
		modeStyle = lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
		modeDesc = "Replacing attacker prompts with benign instructions"
	case "double_agent":
		modeStyle = lipgloss.NewStyle().Foreground(ColorRed).Bold(true)
		modeDesc = "Feeding deceptive results to waste attacker resources"
	case "counter_intel":
		modeStyle = lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
		modeDesc = "Passive + structured intelligence reports per session"
	}

	b.WriteString(fmt.Sprintf("  Mode: %s  %s\n",
		modeStyle.Render("["+strings.ToUpper(modeLabel)+"]"),
		StyleDim.Render(modeDesc),
	))
	b.WriteString("\n")

	if a.stats.CapturedPrompts == 0 && a.stats.TotalEvents == 0 && len(a.l4Intel) == 0 {
		b.WriteString(StyleDim.Render("  No intelligence data available\n\n"))
		b.WriteString(StyleDim.Render("  Intelligence will appear here when:\n"))
		b.WriteString(StyleDim.Render("  - AI agents interact with portal trap services\n"))
		b.WriteString(StyleDim.Render("  - Prompts are captured via L4 intercept\n"))
		b.WriteString(StyleDim.Render("  - Session behaviors are classified\n"))
		return b.String()
	}

	// Summary stats
	b.WriteString(fmt.Sprintf("  %s %s    %s %s    %s %s    %s %s\n",
		StyleCardLabel.Render("Prompts:"),
		StyleValueRed.Render(fmt.Sprintf("%d", a.stats.CapturedPrompts)),
		StyleCardLabel.Render("Sessions:"),
		StyleValueGreen.Render(fmt.Sprintf("%d", a.stats.ActiveSessions)),
		StyleCardLabel.Render("L4 Intercepts:"),
		lipgloss.NewStyle().Foreground(ColorYellow).Bold(true).Render(fmt.Sprintf("%d", a.stats.L4Interceptions)),
		StyleCardLabel.Render("Auth:"),
		StyleValueCyan.Render(fmt.Sprintf("%d", a.stats.AuthAttempts)),
	))
	b.WriteString("\n")

	// L4 Intelligence Reports — show first with more detail
	if len(a.l4Intel) > 0 {
		b.WriteString(StyleBold.Render("  ── Intercepted Agents "))
		b.WriteString(StyleDim.Render(strings.Repeat("─", maxInt(1, a.width-26))))
		b.WriteString("\n")

		maxIntel := min(len(a.l4Intel), 6)
		for i := 0; i < maxIntel; i++ {
			intel := a.l4Intel[i]

			// API Key badge
			keyInfo := "no key"
			if len(intel.APIKeys) > 0 {
				keyInfo = intel.APIKeys[0]
			}
			if intel.KeyType != "" {
				keyInfo += " (" + intel.KeyType + ")"
			}

			// Session and intercept count
			sessionLabel := ""
			if intel.SessionID != "" {
				sessionLabel = truncate(intel.SessionID, 20) + " "
			}

			b.WriteString(fmt.Sprintf("  %s %s%s  %s\n",
				lipgloss.NewStyle().Foreground(ColorYellow).Bold(true).Render("▸"),
				StyleValueCyan.Render(sessionLabel),
				StyleValueRed.Render(keyInfo),
				StyleDim.Render(fmt.Sprintf("%d intercepts", intel.InterceptCount)),
			))

			// Models
			if len(intel.Models) > 0 {
				b.WriteString(fmt.Sprintf("    %s %s\n",
					StyleCardLabel.Render("Models:"),
					StyleDim.Render(strings.Join(intel.Models, ", ")),
				))
			}

			// Tool count + domains
			var line2 []string
			if intel.ToolCount > 0 {
				line2 = append(line2, fmt.Sprintf("%d tools available", intel.ToolCount))
			}
			if len(intel.Domains) > 0 {
				line2 = append(line2, "domains: "+strings.Join(intel.Domains, ", "))
			}
			if intel.OpenAIOrg != "" {
				line2 = append(line2, "org: "+intel.OpenAIOrg)
			}
			if intel.OpenAIProject != "" {
				line2 = append(line2, "project: "+intel.OpenAIProject)
			}
			if len(line2) > 0 {
				b.WriteString(fmt.Sprintf("    %s\n", StyleDim.Render(strings.Join(line2, " | "))))
			}

			// SDK fingerprint
			if intel.UserAgent != "" {
				b.WriteString(fmt.Sprintf("    %s %s\n",
					StyleCardLabel.Render("SDK:"),
					StyleDim.Render(truncate(intel.UserAgent, maxInt(20, a.width-16))),
				))
			}

			// Time range
			if intel.FirstSeen != "" {
				firstTs := intel.FirstSeen
				if len(firstTs) > 19 {
					firstTs = firstTs[:19]
				}
				lastTs := intel.LastSeen
				if len(lastTs) > 19 {
					lastTs = lastTs[:19]
				}
				b.WriteString(fmt.Sprintf("    %s %s → %s\n",
					StyleCardLabel.Render("Active:"),
					StyleDim.Render(firstTs),
					StyleDim.Render(lastTs),
				))
			}

			if i < maxIntel-1 {
				b.WriteString("\n")
			}
		}

		if len(a.l4Intel) > maxIntel {
			b.WriteString(StyleDim.Render(fmt.Sprintf("  ... %d more agents\n", len(a.l4Intel)-maxIntel)))
		}
		b.WriteString("\n")
	}

	// Captured AI system prompts
	b.WriteString(StyleBold.Render("  ── Captured System Prompts "))
	b.WriteString(StyleDim.Render(strings.Repeat("─", maxInt(1, a.width-31))))
	b.WriteString("\n")

	if len(a.prompts) > 0 {
		maxPrompts := min(len(a.prompts), 5)
		for i := 0; i < maxPrompts; i++ {
			p := a.prompts[i]
			domain := p.Domain
			if domain == "" {
				domain = "unknown"
			}

			ts := p.Timestamp
			if len(ts) > 19 {
				ts = ts[:19]
			}

			b.WriteString(fmt.Sprintf("  %s %s  %s\n",
				StyleValueRed.Render("▸"),
				StyleValueCyan.Render(truncate(domain, 30)),
				StyleDim.Render(ts),
			))

			preview := strings.ReplaceAll(p.Text, "\n", " ")
			preview = truncate(preview, maxInt(20, a.width-10))
			b.WriteString(fmt.Sprintf("    %s\n", StyleDim.Render(preview)))
		}

		if len(a.prompts) > maxPrompts {
			b.WriteString(StyleDim.Render(fmt.Sprintf("  ... %d more prompts\n", len(a.prompts)-maxPrompts)))
		}
	} else {
		b.WriteString(StyleDim.Render("  No prompts captured yet. Prompts appear when L4 intercepts AI API traffic.\n"))
	}

	b.WriteString("\n")

	// Event type breakdown from real data
	if len(a.allEvents) > 0 {
		b.WriteString(StyleBold.Render("  ── Event Breakdown "))
		b.WriteString(StyleDim.Render(strings.Repeat("─", maxInt(1, a.width-24))))
		b.WriteString("\n")

		// Count event types
		typeCounts := make(map[string]int)
		for _, ev := range a.allEvents {
			typeCounts[ev.Event]++
		}

		// Sort by count desc — show top 8
		type kv struct {
			key   string
			count int
		}
		var sorted []kv
		for k, v := range typeCounts {
			sorted = append(sorted, kv{k, v})
		}
		for i := 0; i < len(sorted); i++ {
			for j := i + 1; j < len(sorted); j++ {
				if sorted[j].count > sorted[i].count {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}

		maxCount := 0
		if len(sorted) > 0 {
			maxCount = sorted[0].count
		}

		shown := min(len(sorted), 8)
		barWidth := 25
		for i := 0; i < shown; i++ {
			item := sorted[i]
			pct := float64(item.count) / float64(maxCount)
			filled := int(float64(barWidth) * pct)
			if filled < 1 {
				filled = 1
			}
			empty := barWidth - filled

			bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
			b.WriteString(fmt.Sprintf("  %-24s %s %s\n",
				StyleDim.Render(truncate(item.key, 22)),
				StyleValueCyan.Render(bar),
				StyleSubtle.Render(fmt.Sprintf("%d", item.count)),
			))
		}
	}

	return b.String()
}
