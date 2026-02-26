package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ItzDaxxy/labyrinth/cli/internal/api"
	"charm.land/lipgloss/v2"
)

func renderSessions(a *App, height int) string {
	if len(a.sessions) == 0 {
		return renderEmptySessions()
	}

	listWidth := a.width/2 - 1
	detailWidth := a.width - listWidth - 3
	if listWidth < 30 {
		listWidth = 30
	}

	list := renderSessionList(a, listWidth, height)
	detail := renderSessionDetail(a, detailWidth, height)

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

	headerLine := fmt.Sprintf("  %-18s %-8s %s", "ID", "EVENTS", "LAST")
	b.WriteString(StyleSubtle.Render(headerLine))
	b.WriteString("\n")

	maxRows := height - 4
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

		name := truncate(sess.File, 16)
		b.WriteString(fmt.Sprintf("%s%-18s %-8d %s\n",
			prefix,
			nameStyle.Render(name),
			sess.Events,
			StyleDim.Render(truncate(extractTimestamp(sess.Last), 20)),
		))
	}

	return b.String()
}

func renderSessionDetail(a *App, width, height int) string {
	var b strings.Builder

	if a.selectedSession >= len(a.sessions) {
		return StyleDim.Render("  Select a session")
	}

	sess := a.sessions[a.selectedSession]
	b.WriteString(StyleBold.Render(fmt.Sprintf(" %s", sess.File)))
	b.WriteString("\n\n")

	b.WriteString(StyleCardLabel.Render("  Events: "))
	b.WriteString(StyleValueCyan.Render(fmt.Sprintf("%d", sess.Events)))
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
