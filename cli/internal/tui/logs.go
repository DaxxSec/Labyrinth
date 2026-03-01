package tui

import (
	"fmt"
	"strings"

	"github.com/DaxxSec/labyrinth/cli/internal/api"
	"charm.land/lipgloss/v2"
)

func renderLogs(a *App, height int) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(StyleBold.Render("  Event Log"))

	// Get filtered events
	var events []api.ForensicEvent
	if a.logFilterType != "" {
		for _, ev := range a.allEvents {
			if ev.Event == a.logFilterType {
				events = append(events, ev)
			}
		}
	} else {
		events = a.allEvents
	}

	total := len(events)
	if total == 0 && len(a.allEvents) == 0 {
		b.WriteString("\n\n")
		b.WriteString(StyleDim.Render("  No log data available\n\n"))
		b.WriteString(StyleDim.Render("  Logs will stream here when sessions are active.\n"))
		b.WriteString(StyleDim.Render("  Deploy an environment and point an agent at the portal trap.\n"))
		return b.String()
	}

	if total == 0 {
		b.WriteString(fmt.Sprintf("  (%d total, filter: %s — no matches)\n\n", len(a.allEvents), a.logFilterType))
		return b.String()
	}

	maxLines := height - 5
	if maxLines < 1 {
		maxLines = 10
	}

	// Clamp scroll offset
	if a.logScrollOffset > total-maxLines {
		a.logScrollOffset = maxInt(0, total-maxLines)
	}

	endIdx := a.logScrollOffset + maxLines
	if endIdx > total {
		endIdx = total
	}

	// Status line
	b.WriteString(StyleDim.Render(fmt.Sprintf("  Showing %d-%d of %d events",
		a.logScrollOffset+1, endIdx, total)))
	if a.logFilterType != "" {
		b.WriteString(StyleDim.Render(fmt.Sprintf("  [filter: %s]", a.logFilterType)))
	}
	b.WriteString("\n\n")

	// Header
	b.WriteString(fmt.Sprintf("  %s\n",
		StyleSubtle.Render(fmt.Sprintf("%-20s %-6s %-22s %-14s %s", "TIMESTAMP", "LAYER", "EVENT", "SESSION", "DETAILS"))))
	b.WriteString(fmt.Sprintf("  %s\n",
		StyleDim.Render(strings.Repeat("─", a.width-4))))

	// Event rows
	for i := a.logScrollOffset; i < endIdx && i < total; i++ {
		ev := events[i]

		// Color by layer
		lStyle := getLayerStyle(ev.Layer)
		layerStr := lStyle.Render(fmt.Sprintf("L%d", ev.Layer))

		// Timestamp — show just time portion
		ts := ev.Timestamp
		if len(ts) > 11 {
			ts = ts[11:]
		}
		if len(ts) > 8 {
			ts = ts[:8]
		}

		// Session ID — truncate
		sessionStr := truncate(ev.SessionID, 12)
		if sessionStr == "" {
			sessionStr = "-"
		}

		// Event type
		eventStr := truncate(ev.Event, 20)

		// Details from Data map
		details := formatEventData(ev.Event, ev.Data)

		b.WriteString(fmt.Sprintf("  %-20s %s  %-22s %-14s %s\n",
			StyleDim.Render(ts),
			layerStr,
			lStyle.Render(eventStr),
			StyleValueCyan.Render(sessionStr),
			StyleDim.Render(truncate(details, maxInt(1, a.width-70))),
		))
	}

	return b.String()
}

func getLayerStyle(layer int) lipgloss.Style {
	switch layer {
	case 0:
		return StyleValueGreen
	case 1:
		return StyleValueCyan
	case 2:
		return StyleValuePurple
	case 3:
		return StyleValueRed
	case 4:
		return lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
	default:
		return StyleDim
	}
}

func formatEventData(eventType string, data map[string]interface{}) string {
	if len(data) == 0 {
		return ""
	}
	switch eventType {
	case "http_access":
		return fmt.Sprintf("%s %s → %v", str(data, "method"), str(data, "path"), data["status"])
	case "auth_attempt", "auth":
		return fmt.Sprintf("%s %s@%s", str(data, "service"), str(data, "username"), str(data, "src_ip"))
	case "connection":
		return fmt.Sprintf("%s from %s", str(data, "service"), str(data, "source_ip"))
	case "command":
		return str(data, "command")
	case "api_intercepted":
		return fmt.Sprintf("%s %s", str(data, "method"), str(data, "domain"))
	case "blindfold_activated":
		return fmt.Sprintf("depth=%v", data["depth"])
	case "container_spawned":
		return fmt.Sprintf("container=%s", str(data, "container_name"))
	case "depth_increase":
		return fmt.Sprintf("depth %v → %v", data["old_depth"], data["new_depth"])
	default:
		var parts []string
		for k, v := range data {
			s := fmt.Sprintf("%v", v)
			if len(s) > 30 {
				s = s[:27] + "..."
			}
			parts = append(parts, fmt.Sprintf("%s=%s", k, s))
		}
		return strings.Join(parts, " ")
	}
}

func str(data map[string]interface{}, key string) string {
	if v, ok := data[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
