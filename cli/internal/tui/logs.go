package tui

import (
	"fmt"
	"strings"
)

func renderLogs(a *App, height int) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(StyleBold.Render("  Event Log"))
	b.WriteString("\n\n")

	if len(a.sessions) == 0 {
		b.WriteString(StyleDim.Render("  No log data available\n\n"))
		b.WriteString(StyleDim.Render("  Logs will stream here when sessions are active.\n"))
		b.WriteString(StyleDim.Render("  Deploy an environment and point an agent at the honeypot.\n"))
		return b.String()
	}

	// Show recent events from all sessions as a combined log
	maxLines := height - 4
	if maxLines < 1 {
		maxLines = 10
	}

	b.WriteString(fmt.Sprintf("  %s\n",
		StyleSubtle.Render(fmt.Sprintf("%-22s %-8s %-14s %s", "TIMESTAMP", "LAYER", "SESSION", "EVENT"))))
	b.WriteString(fmt.Sprintf("  %s\n",
		StyleDim.Render(strings.Repeat("─", a.width-4))))

	displayed := 0
	for _, sess := range a.sessions {
		if displayed >= maxLines {
			break
		}
		// Each session entry has a "last" JSON line - display it as a log entry
		if sess.Last != "" {
			ts := extractTimestamp(sess.Last)
			name := truncate(strings.TrimSuffix(sess.File, ".jsonl"), 12)

			b.WriteString(fmt.Sprintf("  %-22s %-8s %-14s %s\n",
				StyleDim.Render(ts),
				StyleValuePurple.Render("L1"),
				StyleValueCyan.Render(name),
				StyleSubtle.Render(fmt.Sprintf("%d events", sess.Events)),
			))
			displayed++
		}
	}

	if displayed == 0 {
		b.WriteString(StyleDim.Render("  Waiting for events...\n"))
	}

	return b.String()
}
