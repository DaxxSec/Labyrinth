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

	// Analysis view — post-mortem
	if a.sessionView == 2 && a.sessionAnalysis != nil {
		return renderSessionAnalysis(a, height)
	}

	// Detail view — full event timeline
	if a.sessionView == 1 && a.sessionDetail != nil {
		return renderSessionTimeline(a, height)
	}

	listWidth := a.width*2/3 - 1
	detailWidth := a.width - listWidth - 3
	if listWidth < 40 {
		listWidth = 40
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
	b.WriteString(StyleDim.Render("  Point an offensive agent at the portal trap: localhost:22 or localhost:8080\n"))
	return b.String()
}

func renderSessionList(a *App, width, height int) string {
	var b strings.Builder

	header := fmt.Sprintf(" Sessions (%d)", len(a.sessions))
	b.WriteString(StyleBold.Render(header))
	b.WriteString("\n")

	// Dynamic column widths based on available panel width
	colLayer := 4  // "L3"
	colEvents := 7 // "12345"
	colDepth := 6  // "5"
	colTime := 9   // "02:26:32"
	fixedCols := colLayer + colEvents + colDepth + colTime + 8 // spacing
	nameWidth := width - fixedCols - 2                         // 2 for prefix
	if nameWidth < 20 {
		nameWidth = 20
	}

	headerLine := fmt.Sprintf("  %s %s %s %s %s",
		padRight("ID", nameWidth),
		padRight("L", colLayer),
		padRight("EVENTS", colEvents),
		padRight("DEPTH", colDepth),
		"LAST",
	)
	b.WriteString(StyleSubtle.Render(headerLine))
	b.WriteString("\n")

	maxRows := height - 4
	// Reserve space for auth captures
	if len(a.authEvents) > 0 {
		maxRows -= min(len(a.authEvents)+3, 10)
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

		name := truncate(strings.TrimSuffix(sess.File, ".jsonl"), nameWidth)

		// Extract layer and depth from last event
		layer := extractLayer(sess.Last)
		lStyle := getLayerStyle(layer)
		layerStr := lStyle.Render(fmt.Sprintf("L%d", layer))

		depth := extractDepth(sess.Last)
		depthStr := "-"
		if depth > 0 {
			depthStr = fmt.Sprintf("%d", depth)
		}

		ts := extractTimestamp(sess.Last)
		if len(ts) > colTime {
			ts = ts[:colTime]
		}

		b.WriteString(fmt.Sprintf("%s%s %s %s %s %s\n",
			prefix,
			padRight(nameStyle.Render(name), nameWidth),
			padRight(layerStr, colLayer),
			padRight(fmt.Sprintf("%d", sess.Events), colEvents),
			padRight(StyleValuePurple.Render(depthStr), colDepth),
			StyleDim.Render(ts),
		))
	}

	// Auth captures section
	if len(a.authEvents) > 0 {
		b.WriteString("\n")
		b.WriteString(StyleBold.Render(fmt.Sprintf(" Credentials (%d)", len(a.authEvents))))
		b.WriteString("\n")

		// Dynamic truncation based on available width
		userWidth := (width - 24) / 3
		if userWidth < 14 {
			userWidth = 14
		}
		passWidth := (width - 24) / 4
		if passWidth < 10 {
			passWidth = 10
		}

		maxAuth := min(len(a.authEvents), 7)
		for i := 0; i < maxAuth; i++ {
			auth := a.authEvents[i]

			lBadge := getLayerStyle(1).Render("L1")
			svc := truncate(auth.Service, 6)
			user := truncate(auth.Username, userWidth)
			pass := "****"
			if auth.Password != "" {
				pass = truncate(auth.Password, passWidth)
			}

			ts := ""
			if len(auth.Timestamp) > 11 {
				ts = auth.Timestamp[11:]
				if len(ts) > 8 {
					ts = ts[:8]
				}
			}

			b.WriteString(fmt.Sprintf("  %s %s %s:%s %s %s\n",
				lBadge,
				StyleDim.Render(svc),
				StyleValueCyan.Render(user),
				StyleValueRed.Render(pass),
				StyleDim.Render(auth.SrcIP),
				StyleDim.Render(ts),
			))
		}
		if len(a.authEvents) > maxAuth {
			b.WriteString(StyleDim.Render(fmt.Sprintf("  ... %d more\n", len(a.authEvents)-maxAuth)))
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
	sessName := strings.TrimSuffix(sess.File, ".jsonl")
	b.WriteString(StyleBold.Render(fmt.Sprintf(" %s", sessName)))
	b.WriteString("\n\n")

	layer := extractLayer(sess.Last)
	lStyle := getLayerStyle(layer)
	b.WriteString(StyleCardLabel.Render("  Layer: "))
	b.WriteString(lStyle.Render(fmt.Sprintf("L%d", layer)))
	b.WriteString("  ")
	b.WriteString(StyleCardLabel.Render("Events: "))
	b.WriteString(StyleValueCyan.Render(fmt.Sprintf("%d", sess.Events)))
	b.WriteString("\n")

	depth := extractDepth(sess.Last)
	if depth > 0 {
		b.WriteString(StyleCardLabel.Render("  Max Depth: "))
		b.WriteString(StyleValuePurple.Render(fmt.Sprintf("%d", depth)))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	b.WriteString(StyleSubtle.Render("  [Enter] timeline  [a] analysis"))
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
		details := formatEventData(ev.Event, ev.Data)

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

func renderSessionAnalysis(a *App, height int) string {
	var b strings.Builder
	an := a.sessionAnalysis

	b.WriteString("\n")
	b.WriteString(StyleBold.Render(fmt.Sprintf("  Session Analysis: %s", an.SessionID)))
	b.WriteString("\n")
	b.WriteString(StyleDim.Render(fmt.Sprintf("  %s", strings.Repeat("─", a.width-4))))
	b.WriteString("\n\n")

	// Duration
	durStr := formatDuration(an.DurationSeconds)
	b.WriteString(fmt.Sprintf("  Duration: %s    Events: %s    Max Depth: %s\n",
		StyleValueCyan.Render(durStr),
		StyleValueCyan.Render(fmt.Sprintf("%d", an.TotalEvents)),
		StyleValuePurple.Render(fmt.Sprintf("%d", an.MaxDepth)),
	))

	// Confusion score bar
	filled := an.ConfusionScore / 5 // 20 chars total
	empty := 20 - filled
	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	scoreStyle := StyleValueGreen
	if an.ConfusionScore >= 60 {
		scoreStyle = StyleValueRed
	} else if an.ConfusionScore >= 30 {
		scoreStyle = lipgloss.NewStyle().Foreground(ColorYellow)
	}
	b.WriteString(fmt.Sprintf("  Confusion Score: %s %s\n",
		scoreStyle.Render(bar),
		scoreStyle.Render(fmt.Sprintf("%d/100", an.ConfusionScore)),
	))
	b.WriteString("\n")

	// Layers reached
	if len(an.LayersReached) > 0 {
		var badges []string
		for _, l := range an.LayersReached {
			badges = append(badges, getLayerStyle(l).Render(fmt.Sprintf("L%d", l)))
		}
		b.WriteString(fmt.Sprintf("  Layers Reached: %s", strings.Join(badges, " → ")))
	}

	l3Str := StyleDim.Render("inactive")
	if an.L3Activated {
		l3Str = StyleValueRed.Render("ACTIVATED")
	}
	l4Str := StyleDim.Render("inactive")
	if an.L4Active {
		l4Str = lipgloss.NewStyle().Foreground(ColorYellow).Bold(true).Render("ACTIVE")
	}
	b.WriteString(fmt.Sprintf("    L3 Blindfold: %s    L4 Intercept: %s\n", l3Str, l4Str))
	b.WriteString("\n")

	// Behavioral phases
	if len(an.Phases) > 0 {
		b.WriteString(StyleBold.Render("  ── Behavioral Phases "))
		b.WriteString(StyleDim.Render(strings.Repeat("─", maxInt(1, a.width-26))))
		b.WriteString("\n")

		phaseLabels := map[string]string{
			"reconnaissance":       "Reconnaissance",
			"credential_discovery": "Credential Discovery",
			"initial_access":       "Initial Access",
			"escalation":           "Escalation",
			"confusion":            "Confusion",
			"blindfold":            "Blindfold",
			"interception":         "Interception",
		}

		for _, p := range an.Phases {
			label := phaseLabels[p.Phase]
			if label == "" {
				label = p.Phase
			}
			startTs := p.Start
			if len(startTs) > 11 {
				startTs = startTs[11:]
			}
			if len(startTs) > 8 {
				startTs = startTs[:8]
			}
			endTs := p.End
			if len(endTs) > 11 {
				endTs = endTs[11:]
			}
			if len(endTs) > 8 {
				endTs = endTs[:8]
			}

			b.WriteString(fmt.Sprintf("  ● %-22s %s - %s   %s\n",
				StyleValueCyan.Render(label),
				StyleDim.Render(startTs),
				StyleDim.Render(endTs),
				StyleSubtle.Render(fmt.Sprintf("%d events", p.Events)),
			))
		}
		b.WriteString("\n")
	}

	// Key moments
	if len(an.KeyMoments) > 0 {
		b.WriteString(StyleBold.Render("  ── Key Moments "))
		b.WriteString(StyleDim.Render(strings.Repeat("─", maxInt(1, a.width-20))))
		b.WriteString("\n")

		for _, m := range an.KeyMoments {
			ts := m.Timestamp
			if len(ts) > 11 {
				ts = ts[11:]
			}
			if len(ts) > 8 {
				ts = ts[:8]
			}
			lStyle := getLayerStyle(m.Layer)
			b.WriteString(fmt.Sprintf("  %s  %s  %s\n",
				StyleDim.Render(ts),
				lStyle.Render(fmt.Sprintf("L%d", m.Layer)),
				StyleDim.Render(m.Description),
			))
		}
		b.WriteString("\n")
	}

	// Event breakdown
	if len(an.EventBreakdown) > 0 {
		b.WriteString(StyleBold.Render("  ── Event Breakdown "))
		b.WriteString(StyleDim.Render(strings.Repeat("─", maxInt(1, a.width-24))))
		b.WriteString("\n")

		// Find max count for bar scaling
		maxCount := 0
		for _, c := range an.EventBreakdown {
			if c > maxCount {
				maxCount = c
			}
		}

		// Sort keys for stable display
		var types []string
		for t := range an.EventBreakdown {
			types = append(types, t)
		}
		// Simple sort
		for i := 0; i < len(types); i++ {
			for j := i + 1; j < len(types); j++ {
				if an.EventBreakdown[types[i]] < an.EventBreakdown[types[j]] {
					types[i], types[j] = types[j], types[i]
				}
			}
		}

		barWidth := 20
		for _, t := range types {
			count := an.EventBreakdown[t]
			filled := barWidth
			if maxCount > 0 {
				filled = count * barWidth / maxCount
			}
			if filled < 1 && count > 0 {
				filled = 1
			}
			bar := strings.Repeat("█", filled) + strings.Repeat(" ", barWidth-filled)
			b.WriteString(fmt.Sprintf("  %-22s %s  %s\n",
				StyleDim.Render(t),
				StyleValueCyan.Render(bar),
				StyleSubtle.Render(fmt.Sprintf("%d", count)),
			))
		}
	}

	return b.String()
}

func formatDuration(seconds float64) string {
	if seconds < 60 {
		return fmt.Sprintf("%.0fs", seconds)
	}
	mins := int(seconds) / 60
	secs := int(seconds) % 60
	if mins >= 60 {
		hours := mins / 60
		mins = mins % 60
		return fmt.Sprintf("%dh %dm %ds", hours, mins, secs)
	}
	return fmt.Sprintf("%dm %ds", mins, secs)
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

func padRight(styled string, totalWidth int) string {
	w := lipgloss.Width(styled)
	if w >= totalWidth {
		return styled
	}
	return styled + strings.Repeat(" ", totalWidth-w)
}

func extractLayer(jsonLine string) int {
	var ev struct {
		Layer int `json:"layer"`
	}
	if err := json.Unmarshal([]byte(jsonLine), &ev); err == nil {
		return ev.Layer
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
