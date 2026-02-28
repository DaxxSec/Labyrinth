package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/DaxxSec/labyrinth/cli/internal/api"
	"charm.land/lipgloss/v2"
)

func renderSessions(a *App, height int) string {
	if len(a.sessions) == 0 && len(a.authEvents) == 0 {
		return renderEmptySessions()
	}

	// Detail view — full event timeline
	if a.sessionView == 1 && a.sessionDetail != nil {
		return renderSessionTimeline(a, height)
	}

	listWidth := a.width/2 - 1
	detailWidth := a.width - listWidth - 3
	if listWidth < 30 {
		listWidth = 30
	}

	list := renderSessionList(a, listWidth, height)
	detail := renderSessionSidePanel(a, detailWidth, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, list, " │ ", detail)
}

func renderEmptySessions() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(StyleDim.Render("  No active sessions\n\n"))
	b.WriteString(StyleDim.Render("  Waiting for connections to portal trap services...\n"))
	b.WriteString(StyleDim.Render("  Point an offensive agent at the portal trap: localhost:2222 or localhost:8080\n"))
	return b.String()
}

func renderSessionList(a *App, width, height int) string {
	var b strings.Builder

	header := fmt.Sprintf(" Sessions (%d)", len(a.sessions))
	b.WriteString(StyleBold.Render(header))
	b.WriteString("\n")

	headerLine := fmt.Sprintf("  %-18s %-8s %-6s %s", "ID", "EVENTS", "DEPTH", "LAST")
	b.WriteString(StyleSubtle.Render(headerLine))
	b.WriteString("\n")

	maxRows := height - 4
	// Reserve space for auth captures
	if len(a.authEvents) > 0 {
		maxRows -= min(len(a.authEvents)+3, 8)
	}
	if maxRows < 1 {
		maxRows = 1
	}

	for i, sess := range a.sessions {
		if i >= maxRows {
			break
		}

		prefix := "  "
		nameStyle := StyleValueCyan
		if i == a.selectedSession {
			prefix = "> "
			nameStyle = StyleValueGreen
		}

		name := truncate(strings.TrimSuffix(sess.File, ".jsonl"), 16)
		depth := extractDepth(sess.Last)
		depthStr := "-"
		if depth > 0 {
			depthStr = fmt.Sprintf("%d", depth)
		}

		b.WriteString(fmt.Sprintf("%s%-18s %-8d %-6s %s\n",
			prefix,
			nameStyle.Render(name),
			sess.Events,
			StyleValuePurple.Render(depthStr),
			StyleDim.Render(truncate(extractTimestamp(sess.Last), 12)),
		))
	}

	// Auth captures section
	if len(a.authEvents) > 0 {
		b.WriteString("\n")
		b.WriteString(StyleBold.Render(fmt.Sprintf(" Credentials (%d)", len(a.authEvents))))
		b.WriteString("\n")

		maxAuth := min(len(a.authEvents), 5)
		for i := 0; i < maxAuth; i++ {
			auth := a.authEvents[i]
			svc := truncate(auth.Service, 4)
			user := truncate(auth.Username, 12)
			pass := "****"
			if auth.Password != "" {
				pass = truncate(auth.Password, 8)
			}
			ip := truncate(auth.SrcIP, 15)

			b.WriteString(fmt.Sprintf("  %s %s:%s %s\n",
				StyleDim.Render(svc),
				StyleValueCyan.Render(user),
				StyleValueRed.Render(pass),
				StyleDim.Render(ip),
			))
		}
	}

	return b.String()
}

func renderSessionSidePanel(a *App, width, height int) string {
	var b strings.Builder

	if a.selectedSession >= len(a.sessions) {
		return StyleDim.Render("  Select a session")
	}

	sess := a.sessions[a.selectedSession]
	b.WriteString(StyleBold.Render(fmt.Sprintf(" %s", sess.File)))
	b.WriteString("\n\n")

	b.WriteString(StyleCardLabel.Render("  Events: "))
	b.WriteString(StyleValueCyan.Render(fmt.Sprintf("%d", sess.Events)))
	b.WriteString("\n")

	depth := extractDepth(sess.Last)
	if depth > 0 {
		b.WriteString(StyleCardLabel.Render("  Max Depth: "))
		b.WriteString(StyleValuePurple.Render(fmt.Sprintf("%d", depth)))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString(StyleSubtle.Render("  Press [Enter] for full timeline"))
	b.WriteString("\n\n")

	// Try to parse the last event for detail
	if sess.Last != "" {
		b.WriteString(StyleSubtle.Render("  Last Event:"))
		b.WriteString("\n")

		var event api.ForensicEvent
		if err := json.Unmarshal([]byte(sess.Last), &event); err == nil {
			b.WriteString(fmt.Sprintf("  %s %s\n",
				StyleDim.Render(event.Timestamp),
				StyleValueCyan.Render(event.Event),
			))
			b.WriteString(fmt.Sprintf("  Layer: %s  Session: %s\n",
				StyleValuePurple.Render(fmt.Sprintf("L%d", event.Layer)),
				StyleDim.Render(event.SessionID),
			))
			if event.Data != nil {
				b.WriteString("\n")
				b.WriteString(StyleSubtle.Render("  Data:"))
				b.WriteString("\n")
				for k, v := range event.Data {
					b.WriteString(fmt.Sprintf("    %s: %v\n",
						StyleCardLabel.Render(k),
						StyleDim.Render(fmt.Sprintf("%v", v)),
					))
				}
			}
		} else {
			b.WriteString(fmt.Sprintf("  %s\n", StyleDim.Render(truncate(sess.Last, width-4))))
		}
	}

	return b.String()
}

func renderSessionTimeline(a *App, height int) string {
	var b strings.Builder

	d := a.sessionDetail

	// Header
	b.WriteString("\n")
	b.WriteString(StyleBold.Render(fmt.Sprintf("  %s", d.SessionID)))

	// Summary line
	duration := ""
	if d.FirstSeen != "" && d.LastSeen != "" {
		duration = fmt.Sprintf("  %s → %s", d.FirstSeen, d.LastSeen)
	}

	l3Str := "inactive"
	if d.L3Activated {
		l3Str = StyleValueRed.Render("active")
	}

	b.WriteString(StyleDim.Render(fmt.Sprintf("  Max Depth: %d  L3: %s%s",
		d.MaxDepth, l3Str, duration)))
	b.WriteString("\n")

	// Layers triggered
	if len(d.LayersTriggered) > 0 {
		var layerBadges []string
		for _, l := range d.LayersTriggered {
			style := getLayerStyle(l)
			layerBadges = append(layerBadges, style.Render(fmt.Sprintf("L%d", l)))
		}
		b.WriteString(StyleCardLabel.Render("  Layers: "))
		b.WriteString(strings.Join(layerBadges, " "))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s\n",
		StyleSubtle.Render(fmt.Sprintf("%-10s %-4s %-24s %s", "TIME", "L", "EVENT", "DETAILS"))))
	b.WriteString(fmt.Sprintf("  %s\n",
		StyleDim.Render(strings.Repeat("─", a.width-4))))

	maxLines := height - 8
	if maxLines < 1 {
		maxLines = 10
	}

	for i, ev := range d.Events {
		if i >= maxLines {
			remaining := len(d.Events) - i
			b.WriteString(StyleDim.Render(fmt.Sprintf("  ... %d more events\n", remaining)))
			break
		}

		ts := ev.Timestamp
		if len(ts) > 11 {
			ts = ts[11:]
		}
		if len(ts) > 8 {
			ts = ts[:8]
		}

		lStyle := getLayerStyle(ev.Layer)
		details := formatEventData(ev.Data)

		b.WriteString(fmt.Sprintf("  %-10s %s  %-24s %s\n",
			StyleDim.Render(ts),
			lStyle.Render(fmt.Sprintf("L%d", ev.Layer)),
			lStyle.Render(truncate(ev.Event, 22)),
			StyleDim.Render(truncate(details, maxInt(1, a.width-48))),
		))
	}

	// Show prompt if available
	if d.HasPrompts && d.PromptText != "" {
		b.WriteString("\n")
		b.WriteString(StyleValueRed.Render("  Captured Prompt:"))
		b.WriteString("\n")
		preview := truncate(d.PromptText, 200)
		b.WriteString(StyleDim.Render(fmt.Sprintf("  %s\n", preview)))
	}

	return b.String()
}

func extractDepth(jsonLine string) int {
	var ev struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(jsonLine), &ev); err == nil && ev.Data != nil {
		if d, ok := ev.Data["depth"]; ok {
			if depth, ok := d.(float64); ok {
				return int(depth)
			}
		}
	}
	return 0
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen < 4 {
		return s[:maxLen]
	}
	return s[:maxLen-2] + ".."
}

func extractTimestamp(jsonLine string) string {
	var ev struct {
		Timestamp string `json:"timestamp"`
	}
	if err := json.Unmarshal([]byte(jsonLine), &ev); err == nil && ev.Timestamp != "" {
		// Return just the time portion
		if len(ev.Timestamp) > 11 {
			return ev.Timestamp[11:]
		}
		return ev.Timestamp
	}
	return ""
}
