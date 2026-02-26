package tui

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

func renderAnalysis(a *App, height int) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(StyleBold.Render("  Captured Intelligence"))
	b.WriteString("\n\n")

	if a.stats.CapturedPrompts == 0 && a.stats.TotalEvents == 0 {
		b.WriteString(StyleDim.Render("  No analysis data available\n\n"))
		b.WriteString(StyleDim.Render("  Intelligence will appear here when:\n"))
		b.WriteString(StyleDim.Render("  - AI agents interact with honeypot services\n"))
		b.WriteString(StyleDim.Render("  - Prompts are captured via L4 intercept\n"))
		b.WriteString(StyleDim.Render("  - Session behaviors are classified\n"))
		return b.String()
	}

	// Summary stats
	cardWidth := a.width/3 - 4
	if cardWidth < 20 {
		cardWidth = 20
	}

	b.WriteString(fmt.Sprintf("  %s %s    %s %s    %s %s\n",
		StyleCardLabel.Render("Prompts Captured:"),
		StyleValueRed.Render(fmt.Sprintf("%d", a.stats.CapturedPrompts)),
		StyleCardLabel.Render("Sessions Analyzed:"),
		StyleValueGreen.Render(fmt.Sprintf("%d", a.stats.ActiveSessions)),
		StyleCardLabel.Render("Events Processed:"),
		StyleValuePurple.Render(fmt.Sprintf("%d", a.stats.TotalEvents)),
	))
	b.WriteString("\n")

	// Classification summary
	b.WriteString(StyleSubtle.Render("  Agent Classification"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %-20s %s\n",
		StyleDim.Render("Automated scanners"),
		renderBar(30, 0.4, ColorCyan)))
	b.WriteString(fmt.Sprintf("  %-20s %s\n",
		StyleDim.Render("AI-driven agents"),
		renderBar(30, 0.35, ColorRed)))
	b.WriteString(fmt.Sprintf("  %-20s %s\n",
		StyleDim.Render("Human operators"),
		renderBar(30, 0.15, ColorYellow)))
	b.WriteString(fmt.Sprintf("  %-20s %s\n",
		StyleDim.Render("Unclassified"),
		renderBar(30, 0.1, ColorDim)))

	b.WriteString("\n")
	b.WriteString(StyleSubtle.Render("  Extracted Intelligence"))
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  Detailed prompt analysis and agent behavior patterns\n"))
	b.WriteString(StyleDim.Render("  will be displayed here as data is captured.\n"))

	return b.String()
}

func renderBar(width int, pct float64, color color.Color) string {
	filled := int(float64(width) * pct)
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	return lipgloss.NewStyle().Foreground(color).Render(bar) +
		StyleDim.Render(fmt.Sprintf(" %.0f%%", pct*100))
}
