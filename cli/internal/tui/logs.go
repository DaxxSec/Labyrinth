package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/DaxxSec/labyrinth/cli/internal/api"
	"charm.land/lipgloss/v2"
)

func renderLogs(a *App, height int) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(StyleBold.Render("  Event Log"))

	// Get filtered events
	events := a.filteredLogEvents()

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

	// Clamp cursor
	if a.logCursorPos >= total {
		a.logCursorPos = total - 1
	}
	if a.logCursorPos < 0 {
		a.logCursorPos = 0
	}

	maxLines := height - 5
	if maxLines < 1 {
		maxLines = 10
	}

	// Compute visual line count for each event
	visualLines := make([]int, total)
	totalVisualLines := 0
	for i, ev := range events {
		if a.logExpandedRows[i] && isExpandable(ev) {
			visualLines[i] = 1 + len(ev.Data)
		} else {
			visualLines[i] = 1
		}
		totalVisualLines += visualLines[i]
	}

	// Auto-scroll viewport to keep cursor visible
	// Find visual line offset of cursor row
	cursorVisualStart := 0
	for i := 0; i < a.logCursorPos; i++ {
		cursorVisualStart += visualLines[i]
	}
	cursorVisualEnd := cursorVisualStart + visualLines[a.logCursorPos]

	// Clamp scroll to valid range
	if a.logScrollOffset > totalVisualLines-maxLines {
		a.logScrollOffset = maxInt(0, totalVisualLines-maxLines)
	}
	if a.logScrollOffset < 0 {
		a.logScrollOffset = 0
	}

	// Ensure cursor is visible
	if cursorVisualStart < a.logScrollOffset {
		a.logScrollOffset = cursorVisualStart
	}
	if cursorVisualEnd > a.logScrollOffset+maxLines {
		a.logScrollOffset = cursorVisualEnd - maxLines
	}

	// Status line
	b.WriteString(StyleDim.Render(fmt.Sprintf("  %d events", total)))
	if a.logFilterType != "" {
		b.WriteString(StyleDim.Render(fmt.Sprintf("  [filter: %s]", a.logFilterType)))
	}
	b.WriteString("\n\n")

	// Header
	b.WriteString(fmt.Sprintf("  %s\n",
		StyleSubtle.Render(fmt.Sprintf("  %-20s %-6s %-22s %-14s %s", "TIMESTAMP", "LAYER", "EVENT", "SESSION", "DETAILS"))))
	b.WriteString(fmt.Sprintf("  %s\n",
		StyleDim.Render(strings.Repeat("─", a.width-4))))

	// Visual-line-aware rendering
	// Skip events until we reach logScrollOffset visual lines
	linesRendered := 0
	visualLineAccum := 0

	for i := 0; i < total && linesRendered < maxLines; i++ {
		evVisLines := visualLines[i]

		// Skip events entirely before scroll offset
		if visualLineAccum+evVisLines <= a.logScrollOffset {
			visualLineAccum += evVisLines
			continue
		}

		ev := events[i]
		isCursor := i == a.logCursorPos
		expanded := a.logExpandedRows[i] && isExpandable(ev)

		// How many lines of this event are above the scroll window?
		skipLines := 0
		if visualLineAccum < a.logScrollOffset {
			skipLines = a.logScrollOffset - visualLineAccum
		}
		visualLineAccum += evVisLines

		// Render summary line (line 0 of this event)
		if skipLines == 0 && linesRendered < maxLines {
			summaryLine := renderSummaryLine(a, ev, i, isCursor, expanded)
			b.WriteString(summaryLine)
			b.WriteString("\n")
			linesRendered++
		} else if skipLines > 0 {
			skipLines--
		}

		// Render detail lines if expanded
		if expanded {
			detailLines := renderDetailLines(ev, a.width)
			for _, dl := range detailLines {
				if skipLines > 0 {
					skipLines--
					continue
				}
				if linesRendered >= maxLines {
					break
				}
				if isCursor {
					b.WriteString(StyleCursorRow.Render(dl))
				} else {
					b.WriteString(dl)
				}
				b.WriteString("\n")
				linesRendered++
			}
		}
	}

	return b.String()
}

// renderSummaryLine renders a single event row (the collapsed/summary line).
func renderSummaryLine(a *App, ev api.ForensicEvent, idx int, isCursor, expanded bool) string {
	// Expand indicator
	indicator := "  "
	if isExpandable(ev) {
		if expanded {
			indicator = "▾ "
		} else {
			indicator = "▸ "
		}
	}

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

	detailStr := truncate(details+tag, maxInt(1, a.width-72))

	line := fmt.Sprintf("  %s%-20s %s  %-22s %-14s %s",
		indicator,
		StyleDim.Render(ts),
		layerStr,
		lStyle.Render(eventStr),
		StyleValueCyan.Render(sessionStr),
		detailStyle.Render(detailStr),
	)

	if isCursor {
		return StyleCursorRow.Render(line)
	}
	return line
}

// isExpandable returns true if the event has 2+ data fields.
func isExpandable(ev api.ForensicEvent) bool {
	return len(ev.Data) >= 2
}

// detailValueStyle returns a color style based on the key name pattern.
func detailValueStyle(key string) lipgloss.Style {
	k := strings.ToLower(key)
	switch {
	case strings.Contains(k, "pass") || strings.Contains(k, "token") ||
		strings.Contains(k, "key") || strings.Contains(k, "secret"):
		return StyleDetailValRed
	case strings.Contains(k, "ip") || strings.Contains(k, "host") ||
		strings.Contains(k, "domain"):
		return StyleValueCyan
	case strings.Contains(k, "path") || strings.Contains(k, "url") ||
		strings.Contains(k, "file"):
		return StyleValueGreen
	case strings.Contains(k, "status") || strings.Contains(k, "port") ||
		strings.Contains(k, "depth") || strings.Contains(k, "count") ||
		strings.Contains(k, "duration"):
		return lipgloss.NewStyle().Foreground(ColorYellow)
	default:
		return StyleDim
	}
}

// renderDetailLines returns the formatted detail lines for an expanded event.
func renderDetailLines(ev api.ForensicEvent, width int) []string {
	if len(ev.Data) == 0 {
		return nil
	}

	// Sort keys for stable output
	keys := make([]string, 0, len(ev.Data))
	for k := range ev.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Padding to align with the DETAILS column (indicator + timestamp + layer + event + session)
	const pad = "                                                      "

	var lines []string
	for _, k := range keys {
		v := fmt.Sprintf("%v", ev.Data[k])
		maxVal := width - 56 - 14 - 4
		if maxVal < 10 {
			maxVal = 10
		}
		if len(v) > maxVal {
			v = v[:maxVal-3] + "..."
		}
		keyStr := StyleDetailKey.Render(k)
		valStr := detailValueStyle(k).Render(v)
		lines = append(lines, fmt.Sprintf("%s│ %s %s", pad, keyStr, valStr))
	}
	return lines
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
	case "service_connection":
		return fmt.Sprintf("%s %s:%s", str(data, "protocol"), str(data, "client_ip"), str(data, "port"))
	case "service_auth":
		s := str(data, "protocol")
		if user := str(data, "username"); user != "" {
			s += " user=" + user
		}
		if token := str(data, "token"); token != "" {
			s += " token=" + token
		}
		return s
	case "service_query":
		s := str(data, "protocol")
		if q := str(data, "query"); q != "" {
			s += " " + q
		} else if cmd := str(data, "command"); cmd != "" {
			s += " " + cmd
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
