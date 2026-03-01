package tui

import (
	"fmt"
	"strings"
)

func renderLayers(a *App, height int) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(StyleBold.Render("  LABYRINTH Layer Architecture"))
	b.WriteString("\n\n")

	if len(a.layerStatuses) == 0 {
		b.WriteString(StyleDim.Render("  No layer configuration loaded\n"))
		return b.String()
	}

	// Map standby layers to "awaiting" with prerequisite layer names
	awaitingDetail := map[int]string{
		2: "Awaiting session escalation",
		3: "Awaiting depth >= 3",
		4: "Awaiting egress interception",
	}

	for i, layer := range a.layerStatuses {
		statusStyle := StyleStatusRunning
		statusIcon := "●"
		statusLabel := "active"
		if layer.Status == "standby" {
			if i >= 2 {
				// Session-dependent layers show orange "awaiting"
				statusStyle = StyleStatusWaiting
				statusIcon = "●"
				statusLabel = "awaiting"
			} else {
				statusStyle = StyleDim
				statusIcon = "○"
				statusLabel = "standby"
			}
		}

		// Layer box
		boxWidth := a.width - 8
		if boxWidth < 40 {
			boxWidth = 40
		}

		nameStr := fmt.Sprintf("  %s %s", statusStyle.Render(statusIcon), StyleBold.Render(layer.Name))

		// Status + session badge
		statusText := fmt.Sprintf("[%s]", statusLabel)
		if layer.Sessions > 0 {
			statusText = fmt.Sprintf("[%s] %d sessions", statusLabel, layer.Sessions)
		}
		statusStr := statusStyle.Render(statusText)

		padding := boxWidth - len(layer.Name) - len(statusText) - 8
		if padding < 1 {
			padding = 1
		}

		detail := layer.Detail
		if detail == "" {
			if layer.Status == "standby" {
				if msg, ok := awaitingDetail[i]; ok {
					detail = msg
				}
			}
			// Fall back to config-style defaults
			if detail == "" {
				switch i {
				case 0:
					detail = "Encryption: AES-256-GCM | Network isolation | Retention policy"
				case 1:
					detail = "SSH portal trap (:22) | HTTP portal trap (:8080) | Session logging"
				case 2:
					detail = "Adaptive filesystem | Contradiction density: medium"
				case 3:
					detail = "Activation: on_escalation | Method: bashrc_payload | + L4 proxy on activation"
				case 4:
					detail = fmt.Sprintf("Mode: %s | [m] to cycle | Prompt logging: on", a.l4Mode)
				}
			}
		}

		detailPad := maxInt(0, boxWidth-len(detail)-2)

		b.WriteString(fmt.Sprintf("  ┌%s┐\n", strings.Repeat("─", boxWidth)))
		b.WriteString(fmt.Sprintf("  │ %s%s%s │\n", nameStr, strings.Repeat(" ", padding), statusStr))
		b.WriteString(fmt.Sprintf("  │ %s%s │\n",
			StyleDim.Render(detail),
			strings.Repeat(" ", detailPad),
		))
		b.WriteString(fmt.Sprintf("  └%s┘\n", strings.Repeat("─", boxWidth)))

		if i < len(a.layerStatuses)-1 {
			center := boxWidth / 2
			b.WriteString(fmt.Sprintf("  %s│\n", strings.Repeat(" ", center)))
			b.WriteString(fmt.Sprintf("  %s▼\n", strings.Repeat(" ", center)))
		}
	}

	return b.String()
}
