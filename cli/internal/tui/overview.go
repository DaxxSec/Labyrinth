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

	b.WriteString(StyleCardLabel.Render("AUTH ATTEMPTS: "))
	b.WriteString(lipgloss.NewStyle().Foreground(ColorYellow).Bold(true).Render(fmt.Sprintf("%d", a.stats.AuthAttempts)))
	b.WriteString("\n")

	b.WriteString(StyleCardLabel.Render("CONTAINERS: "))
	containerCount := a.stats.ActiveContainers
	if a.containers != nil {
		containerCount = len(a.containers.Infrastructure) + len(a.containers.Sessions)
	}
	b.WriteString(StyleValueCyan.Render(fmt.Sprintf("%d", containerCount)))
	if a.containers != nil && len(a.containers.Sessions) > 0 {
		b.WriteString(StyleDim.Render(fmt.Sprintf(" (%d session)", len(a.containers.Sessions))))
	}

	return b.String()
}

func renderServiceTable(a *App) string {
	var b strings.Builder

	headerStyle := StyleBold
	b.WriteString(fmt.Sprintf("  %s\n",
		headerStyle.Render(fmt.Sprintf("%-24s %-14s %-10s %s", "SERVICE", "STATUS", "LAYER", "PORTS"))))
	b.WriteString(fmt.Sprintf("  %s\n",
		StyleDim.Render(fmt.Sprintf("%-24s %-14s %-10s %s", "────────────────────────", "──────────────", "──────────", "─────────────"))))

	// Use real container data if available
	if a.containers != nil && len(a.containers.Infrastructure) > 0 {
		for _, c := range a.containers.Infrastructure {
			statusIcon := "●"
			statusStyle := StyleStatusRunning
			statusText := c.State
			if c.State != "running" {
				statusIcon = "○"
				statusStyle = StyleStatusStopped
				statusText = c.State
			}

			layerStr := c.Layer
			if layerStr == "" {
				layerStr = "-"
			}

			b.WriteString(fmt.Sprintf("  %-24s %s  %-10s %s\n",
				StyleValueCyan.Render(truncate(c.Name, 22)),
				statusStyle.Render(fmt.Sprintf("%s %-10s", statusIcon, statusText)),
				StyleSubtle.Render(layerStr),
				StyleDim.Render(c.Ports),
			))
		}

		// Session containers
		if len(a.containers.Sessions) > 0 {
			b.WriteString("\n")
			b.WriteString(StyleSubtle.Render(fmt.Sprintf("  Session Containers (%d):\n", len(a.containers.Sessions))))
			for _, c := range a.containers.Sessions {
				statusIcon := "●"
				statusStyle := StyleStatusRunning
				if c.State != "running" {
					statusIcon = "○"
					statusStyle = StyleStatusStopped
				}
				b.WriteString(fmt.Sprintf("  %-24s %s\n",
					StyleDim.Render(truncate(c.Name, 22)),
					statusStyle.Render(fmt.Sprintf("%s %s", statusIcon, c.State)),
				))
			}
		}
	} else {
		// Fallback to static service list
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
			statusStr := StyleStatusRunning.Render("● " + svc.status)
			if a.dataSource == SourceNone {
				statusStr = StyleDim.Render("○ unknown")
			}
			b.WriteString(fmt.Sprintf("  %-24s %-14s %-10s %s\n",
				StyleValueCyan.Render(svc.name),
				statusStr,
				StyleSubtle.Render(svc.layer),
				StyleDim.Render(svc.ports),
			))
		}
	}

	return b.String()
}
