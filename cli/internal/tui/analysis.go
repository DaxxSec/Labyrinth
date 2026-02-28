package tui

import (
	"fmt"
	"strings"
)

func renderAnalysis(a *App, height int) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(StyleBold.Render("  Captured Intelligence"))
	b.WriteString("\n\n")

	if a.stats.CapturedPrompts == 0 && a.stats.TotalEvents == 0 {
		b.WriteString(StyleDim.Render("  No analysis data available\n\n"))
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
		StyleCardLabel.Render("Events:"),
		StyleValuePurple.Render(fmt.Sprintf("%d", a.stats.TotalEvents)),
		StyleCardLabel.Render("Auth:"),
		StyleValueCyan.Render(fmt.Sprintf("%d", a.stats.AuthAttempts)),
	))
	b.WriteString("\n")

	// Event type breakdown from real data
	if len(a.allEvents) > 0 {
		b.WriteString(StyleSubtle.Render("  Event Type Breakdown"))
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
		barWidth := 30
		for i := 0; i < shown; i++ {
			item := sorted[i]
			pct := float64(item.count) / float64(maxCount)
			filled := int(float64(barWidth) * pct)
			if filled < 1 {
				filled = 1
			}
			empty := barWidth - filled

			bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
			b.WriteString(fmt.Sprintf("  %-22s %s %s\n",
				StyleDim.Render(truncate(item.key, 20)),
				StyleValueCyan.Render(bar),
				StyleSubtle.Render(fmt.Sprintf("%d", item.count)),
			))
		}
		b.WriteString("\n")
	}

	// Captured AI system prompts
	if len(a.prompts) > 0 {
		b.WriteString(StyleSubtle.Render("  Captured AI System Prompts"))
		b.WriteString("\n")

		maxPrompts := min(len(a.prompts), 5)
		for i := 0; i < maxPrompts; i++ {
			p := a.prompts[i]
			domain := p.Domain
			if domain == "" {
				domain = "unknown"
			}

			b.WriteString(fmt.Sprintf("  %s %s  %s\n",
				StyleValueRed.Render("▸"),
				StyleValueCyan.Render(truncate(domain, 25)),
				StyleDim.Render(p.Timestamp),
			))

			preview := strings.ReplaceAll(p.Text, "\n", " ")
			preview = truncate(preview, maxInt(20, a.width-10))
			b.WriteString(fmt.Sprintf("    %s\n", StyleDim.Render(preview)))
		}

		if len(a.prompts) > maxPrompts {
			b.WriteString(StyleDim.Render(fmt.Sprintf("  ... %d more prompts\n", len(a.prompts)-maxPrompts)))
		}
	} else {
		b.WriteString(StyleSubtle.Render("  Captured AI System Prompts"))
		b.WriteString("\n")
		b.WriteString(StyleDim.Render("  No prompts captured yet. Prompts appear when L4 intercepts API traffic.\n"))
	}

	return b.String()
}
