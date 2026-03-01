package tui

import (
	"fmt"
	"strings"
)

func renderEnvironment(a *App, height int) string {
	leftWidth := a.width*3/5 - 1
	rightWidth := a.width - leftWidth - 3
	if leftWidth < 40 {
		leftWidth = 40
	}
	if rightWidth < 20 {
		rightWidth = 20
	}

	left := renderBaitIdentity(a, leftWidth, height)
	right := renderContainerLogsPanel(a, rightWidth, height)

	return lipglossJoinHorizontal(left, " │ ", right)
}

func renderBaitIdentity(a *App, width, height int) string {
	var b strings.Builder

	b.WriteString("\n")

	if a.baitIdentity == nil {
		b.WriteString(StyleDim.Render("  No bait identity loaded\n\n"))
		b.WriteString(StyleDim.Render("  Bait identity is generated when the HTTP honeypot starts.\n"))
		b.WriteString(StyleDim.Render("  Deploy an environment to populate this view.\n"))
		return b.String()
	}

	id := a.baitIdentity

	// Company Identity
	b.WriteString(StyleBold.Render("  ── Company Identity "))
	b.WriteString(StyleDim.Render(strings.Repeat("─", maxInt(1, width-24))))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s  %s\n", StyleSubtle.Render("Company:"), StyleValueCyan.Render(id.Company)))
	b.WriteString(fmt.Sprintf("  %s   %s\n", StyleSubtle.Render("Domain:"), StyleValueCyan.Render(id.Domain)))
	b.WriteString("\n")

	// Bait Users
	b.WriteString(StyleBold.Render("  ── Bait Users "))
	b.WriteString(StyleDim.Render(strings.Repeat("─", maxInt(1, width-18))))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s\n",
		StyleSubtle.Render(fmt.Sprintf("%-14s %-28s %s", "USERNAME", "EMAIL", "ROLE"))))
	b.WriteString(fmt.Sprintf("  %s\n", StyleDim.Render(strings.Repeat("─", maxInt(1, width-4)))))

	for _, u := range id.Users {
		uname := truncate(u.Uname, 12)
		email := truncate(u.Email, 26)
		role := truncate(u.Role, 14)
		b.WriteString(fmt.Sprintf("  %-14s %-28s %s\n",
			StyleValueGreen.Render(uname),
			StyleValueCyan.Render(email),
			StyleDim.Render(role),
		))
	}
	b.WriteString("\n")

	// Planted Credentials
	b.WriteString(StyleBold.Render("  ── Planted Credentials "))
	b.WriteString(StyleDim.Render(strings.Repeat("─", maxInt(1, width-27))))
	b.WriteString("\n")

	creds := []struct {
		label string
		value string
	}{
		{"DB_PASS", id.DBPass},
		{"AWS_KEY_ID", id.AWSKeyID},
		{"AWS_SECRET", id.AWSSecret},
		{"JWT_SECRET", id.JWTSecret},
		{"API_KEY", id.APIKey},
		{"STRIPE_KEY", id.StripeKey},
		{"DEPLOY_KEY", id.DeployKey},
		{"REDIS_TOKEN", id.RedisToken},
	}

	for _, c := range creds {
		if c.value == "" {
			continue
		}
		val := c.value
		if len(val) > 28 {
			val = val[:25] + "..."
		}
		b.WriteString(fmt.Sprintf("  %-14s %s\n",
			StyleSubtle.Render(c.label),
			StyleValueRed.Render(val),
		))
	}
	b.WriteString("\n")

	// Bait Paths
	b.WriteString(StyleBold.Render("  ── Bait Paths "))
	b.WriteString(StyleDim.Render(strings.Repeat("─", maxInt(1, width-18))))
	b.WriteString("\n")

	if len(id.BaitPaths) > 0 {
		for _, p := range id.BaitPaths {
			b.WriteString(fmt.Sprintf("  %s\n", StyleValueGreen.Render(p)))
		}
	} else {
		b.WriteString(StyleDim.Render("  No bait paths configured\n"))
	}

	return b.String()
}

func renderContainerLogsPanel(a *App, width, height int) string {
	var b strings.Builder

	svc := envLogServices[a.selectedLogSvc]

	b.WriteString("\n")
	b.WriteString(StyleBold.Render(fmt.Sprintf("  ── Container Logs: %s ", svc)))
	b.WriteString(StyleDim.Render(strings.Repeat("─", maxInt(1, width-24-len(svc)))))
	b.WriteString("\n\n")

	logs, ok := a.containerLogs[svc]
	if !ok || logs == nil || len(logs.Lines) == 0 {
		if ok && logs != nil && logs.Error != "" {
			b.WriteString(StyleValueRed.Render(fmt.Sprintf("  Error: %s\n", truncate(logs.Error, width-10))))
		} else {
			b.WriteString(StyleDim.Render("  No logs available\n"))
			b.WriteString(StyleDim.Render("  Press [s] to switch service\n"))
		}
		return b.String()
	}

	maxLines := height - 6
	if maxLines < 5 {
		maxLines = 5
	}

	total := len(logs.Lines)

	// Clamp scroll
	if a.envLogScroll > total-maxLines {
		a.envLogScroll = maxInt(0, total-maxLines)
	}

	endIdx := a.envLogScroll + maxLines
	if endIdx > total {
		endIdx = total
	}

	b.WriteString(StyleDim.Render(fmt.Sprintf("  %d-%d of %d lines", a.envLogScroll+1, endIdx, total)))
	b.WriteString("\n\n")

	lineWidth := maxInt(1, width-4)
	for i := a.envLogScroll; i < endIdx && i < total; i++ {
		line := logs.Lines[i]
		if len(line) > lineWidth {
			line = line[:lineWidth-3] + "..."
		}
		b.WriteString(fmt.Sprintf("  %s\n", StyleDim.Render(line)))
	}

	return b.String()
}

func lipglossJoinHorizontal(left, sep, right string) string {
	leftLines := strings.Split(left, "\n")
	rightLines := strings.Split(right, "\n")

	maxLen := len(leftLines)
	if len(rightLines) > maxLen {
		maxLen = len(rightLines)
	}

	// Find widest left line
	leftWidth := 0
	for _, l := range leftLines {
		w := len(l)
		if w > leftWidth {
			leftWidth = w
		}
	}

	var b strings.Builder
	for i := 0; i < maxLen; i++ {
		l := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		r := ""
		if i < len(rightLines) {
			r = rightLines[i]
		}
		// Pad left side
		padding := leftWidth - len(l)
		if padding < 0 {
			padding = 0
		}
		b.WriteString(l)
		b.WriteString(strings.Repeat(" ", padding))
		b.WriteString(sep)
		b.WriteString(r)
		if i < maxLen-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}
