package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func renderOverview(a *App, height int) string {
	var b strings.Builder

	// Environment info + stats cards side by side
	envCard := renderEnvInfo(a)
	statsCard := renderStatsCards(a)

	cardWidth := a.width/2 - 2
	if cardWidth < 30 {
		cardWidth = 30
	}

	envStyle := StyleCard.Width(cardWidth)
	statsStyle := StyleCard.Width(cardWidth)

	row := lipgloss.JoinHorizontal(lipgloss.Top,
		envStyle.Render(envCard),
		statsStyle.Render(statsCard),
	)
	b.WriteString(row)
	b.WriteString("\n")

	// Service table
	b.WriteString(renderServiceTable(a))

	return b.String()
}

func renderEnvInfo(a *App) string {
	var b strings.Builder

	if len(a.environments) > 0 {
		env := a.environments[0] // Show first environment
		b.WriteString(StyleCardLabel.Render("ENV: "))
		b.WriteString(StyleBold.Render(env.Name))
		b.WriteString("\n")
		b.WriteString(StyleCardLabel.Render("Type: "))
		b.WriteString(StyleSubtle.Render(env.Type))
		b.WriteString("\n")
		b.WriteString(StyleCardLabel.Render("Mode: "))
		b.WriteString(StyleSubtle.Render(env.Mode))
		b.WriteString("\n")
		b.WriteString(StyleCardLabel.Render("Health: "))
		if a.dataSource == SourceAPI {
			b.WriteString(StyleStatusRunning.Render("● RUNNING"))
		} else if a.dataSource == SourceFiles {
			b.WriteString(lipgloss.NewStyle().Foreground(ColorYellow).Render("● FILES ONLY"))
		} else {
			b.WriteString(StyleStatusStopped.Render("● NO CONNECTION"))
		}
	} else {
		b.WriteString(StyleDim.Render("No environments deployed\n"))
		b.WriteString(StyleDim.Render("Run: labyrinth deploy -t"))
	}

	return b.String()
}

func renderStatsCards(a *App) string {
	var b strings.Builder

	b.WriteString(StyleCardLabel.Render("ACTIVE SESSIONS: "))
	b.WriteString(StyleValueGreen.Render(fmt.Sprintf("%d", a.stats.ActiveSessions)))
	b.WriteString("\n")

	b.WriteString(StyleCardLabel.Render("CAPTURED PROMPTS: "))
	b.WriteString(StyleValueRed.Render(fmt.Sprintf("%d", a.stats.CapturedPrompts)))
	b.WriteString("\n")

	b.WriteString(StyleCardLabel.Render("TOTAL EVENTS: "))
	b.WriteString(StyleValuePurple.Render(fmt.Sprintf("%d", a.stats.TotalEvents)))
	b.WriteString("\n")

	// Mini sparkline (static representation for now)
	if a.stats.TotalEvents > 0 {
		b.WriteString("\n")
		b.WriteString(StyleDim.Render("▂▃▅▇▅▃▂▃▅▇ conn/min"))
	}

	return b.String()
}

func renderServiceTable(a *App) string {
	var b strings.Builder

	headerStyle := StyleBold
	b.WriteString(fmt.Sprintf("  %s\n",
		headerStyle.Render(fmt.Sprintf("%-22s %-12s %-10s %s", "SERVICE", "STATUS", "LAYER", "PORTS"))))
	b.WriteString(fmt.Sprintf("  %s\n",
		StyleDim.Render(fmt.Sprintf("%-22s %-12s %-10s %s", "──────────────────────", "────────────", "──────────", "─────────────"))))

	services := []struct {
		name   string
		layer  string
		ports  string
		status string
	}{
		{"labyrinth-ssh", "L1", "2222:22", "running"},
		{"labyrinth-http", "L1", "8080:80", "running"},
		{"labyrinth-orchestrator", "orch", "-", "running"},
		{"labyrinth-proxy", "L4", "-", "running"},
		{"labyrinth-dashboard", "dash", "9000:9000", "running"},
	}

	for _, svc := range services {
		statusStr := StyleStatusRunning.Render(svc.status)
		if a.dataSource == SourceNone {
			statusStr = StyleDim.Render("unknown")
		}
		b.WriteString(fmt.Sprintf("  %-22s %s  %-10s %s\n",
			StyleValueCyan.Render(svc.name),
			statusStr,
			StyleSubtle.Render(svc.layer),
			StyleDim.Render(svc.ports),
		))
	}

	return b.String()
}
