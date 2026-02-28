package tui

import (
	"fmt"
	"strings"

	"github.com/DaxxSec/labyrinth/cli/internal/registry"
	"charm.land/lipgloss/v2"
)

// renderEnvSelector renders the environment selector bar.
// Only shown when there are multiple environments.
func (a *App) renderEnvSelector() string {
	if len(a.environments) <= 1 && !a.envSelectorOpen {
		return ""
	}

	var tabs []string

	// "All" tab
	allLabel := " All "
	if a.selectedEnv == -1 {
		tabs = append(tabs, StyleActiveTab.Render(allLabel))
	} else {
		tabs = append(tabs, StyleInactiveTab.Render(allLabel))
	}

	// Per-environment tabs
	for i, env := range a.environments {
		badge := envBadge(env)
		label := fmt.Sprintf(" %s %s ", env.Name, badge)
		if i == a.selectedEnv {
			tabs = append(tabs, StyleActiveTab.Render(label))
		} else {
			tabs = append(tabs, StyleInactiveTab.Render(label))
		}
	}

	return "│ " + strings.Join(tabs, " │ ") + " │"
}

// envBadge returns a colored PROD or DEV badge string for an environment.
func envBadge(env registry.Environment) string {
	if env.Type == "production" {
		return lipgloss.NewStyle().Foreground(ColorRed).Bold(true).Render("PROD")
	}
	return lipgloss.NewStyle().Foreground(ColorGreen).Bold(true).Render("DEV")
}

// modeBadge returns a header badge for the current environment mode.
func (a *App) modeBadge() string {
	if a.activeEnvName == "" && len(a.environments) > 0 {
		// "All" mode — show aggregated
		return lipgloss.NewStyle().Foreground(ColorCyan).Bold(true).Render("[ALL]")
	}

	for _, env := range a.environments {
		if env.Name == a.activeEnvName {
			if env.Type == "production" {
				return lipgloss.NewStyle().
					Foreground(ColorRed).
					Bold(true).
					Render("[PROD]")
			}
			return lipgloss.NewStyle().
				Foreground(ColorGreen).
				Bold(true).
				Render("[DEV]")
		}
	}

	return ""
}
