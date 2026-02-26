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

	if a.layerConfig == nil {
		b.WriteString(StyleDim.Render("  No layer configuration loaded\n"))
		return b.String()
	}

	for i, layer := range a.layerConfig.layers {
		statusStyle := StyleStatusRunning
		statusIcon := "●"
		if layer.status == "standby" {
			statusStyle = StyleDim
			statusIcon = "○"
		} else if layer.status == "active" {
			statusIcon = "●"
		}

		// Layer box
		boxWidth := a.width - 8
		if boxWidth < 40 {
			boxWidth = 40
		}

		nameStr := fmt.Sprintf("  %s %s", statusStyle.Render(statusIcon), StyleBold.Render(layer.name))
		statusStr := statusStyle.Render(fmt.Sprintf("[%s]", layer.status))

		padding := boxWidth - len(layer.name) - len(layer.status) - 8
		if padding < 1 {
			padding = 1
		}

		b.WriteString(fmt.Sprintf("  ┌%s┐\n", strings.Repeat("─", boxWidth)))
		b.WriteString(fmt.Sprintf("  │ %s%s%s │\n", nameStr, strings.Repeat(" ", padding), statusStr))
		b.WriteString(fmt.Sprintf("  │ %s%s │\n",
			StyleDim.Render(layer.detail),
			strings.Repeat(" ", maxInt(0, boxWidth-len(layer.detail)-2)),
		))
		b.WriteString(fmt.Sprintf("  └%s┘\n", strings.Repeat("─", boxWidth)))

		if i < len(a.layerConfig.layers)-1 {
			center := boxWidth / 2
			b.WriteString(fmt.Sprintf("  %s│\n", strings.Repeat(" ", center)))
			b.WriteString(fmt.Sprintf("  %s▼\n", strings.Repeat(" ", center)))
		}
	}

	return b.String()
}
