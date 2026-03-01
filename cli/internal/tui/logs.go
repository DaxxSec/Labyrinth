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

		// Highlight bait and deception events
		detailStyle := StyleDim
		tag := ""
		if isBaitEvent(ev) {
			detailStyle = lipgloss.NewStyle().Foreground(ColorYellow).Bold(true)
			tag = " [BAIT]"
		} else if isDeceptionEvent(ev) {
			detailStyle = StyleValuePurple
			tag = " [DECEPTION]"
		} else if ev.Event == "api_intercepted" {
			detailStyle = lipgloss.NewStyle().Foreground(ColorYellow)
		}

		detailStr := truncate(details+tag, maxInt(1, a.width-70))

		b.WriteString(fmt.Sprintf("  %-20s %s  %-22s %-14s %s\n",
			StyleDim.Render(ts),
			layerStr,
			lStyle.Render(eventStr),
			StyleValueCyan.Render(sessionStr),
			detailStyle.Render(detailStr),
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
		s := fmt.Sprintf("%s %s", str(data, "method"), str(data, "domain"))
		if str(data, "prompt_swapped") == "true" {
			s += " (prompt swapped: " + str(data, "mode") + ")"
		}
		return s
	case "api_response":
		s := str(data, "domain")
		if model := str(data, "model"); model != "" {
			s += " model=" + model
		}
		if tc := str(data, "tool_call_count"); tc != "" && tc != "0" {
			s += " tools=" + tc
		}
		return s
	case "blindfold_activated":
		return fmt.Sprintf("depth=%v", data["depth"])
	case "proxy_interception_activated":
		return fmt.Sprintf("proxy=%s depth=%v", str(data, "proxy_ip"), data["depth"])
	case "container_spawned":
		s := fmt.Sprintf("ip=%s depth=%v", str(data, "container_ip"), data["depth"])
		if density := str(data, "contradiction_density"); density != "" {
			s += " contradictions=" + density
		}
		return s
	case "depth_increase":
		s := fmt.Sprintf("depth → %v", data["new_depth"])
		if density := str(data, "density"); density != "" {
			s += " contradictions=" + density
		}
		return s
	case "escalation_detected", "escalation":
		s := str(data, "type")
		if file := str(data, "file"); file != "" {
			s += " " + file
		}
		return s
	case "container_ready":
		if c := str(data, "contradictions"); c != "" {
			return fmt.Sprintf("contradictions=%s", c)
		}
		return ""
	case "session_end":
		return fmt.Sprintf("duration=%vs depth=%v cmds=%v",
			data["duration_seconds"], data["final_depth"], data["command_count"])
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

// isBaitEvent returns true if the event involves bait file interaction.
func isBaitEvent(ev api.ForensicEvent) bool {
	if ev.Event == "escalation" || ev.Event == "escalation_detected" {
		return true
	}
	if ev.Event == "http_access" {
		path := str(ev.Data, "path")
		for _, bp := range []string{"/.env", "/api/config", "/api/users", "/backup/"} {
			if path == bp || strings.HasPrefix(path, bp) {
				return true
			}
		}
	}
	return false
}

// isDeceptionEvent returns true if the event is L2 deception related.
func isDeceptionEvent(ev api.ForensicEvent) bool {
	if ev.Layer == 2 {
		return true
	}
	if ev.Event == "container_spawned" {
		if d := str(ev.Data, "contradiction_density"); d != "" {
			return true
		}
	}
	return false
}

func str(data map[string]interface{}, key string) string {
	if v, ok := data[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return ""
}
