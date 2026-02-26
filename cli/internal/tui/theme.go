package tui

import "charm.land/lipgloss/v2"

// Colors matching the dashboard CSS palette
var (
	ColorBg        = lipgloss.Color("#0a0a0f")
	ColorCardBg    = lipgloss.Color("#0d1117")
	ColorBorder    = lipgloss.Color("#1a1a2e")
	ColorText      = lipgloss.Color("#c9d1d9")
	ColorDim       = lipgloss.Color("#484f58")
	ColorSubtle    = lipgloss.Color("#8b949e")
	ColorGreen     = lipgloss.Color("#00ff88")
	ColorCyan      = lipgloss.Color("#00ccff")
	ColorRed       = lipgloss.Color("#ff3366")
	ColorPurple    = lipgloss.Color("#cc33ff")
	ColorYellow    = lipgloss.Color("#ffaa00")
	ColorMagenta   = lipgloss.Color("#ff66ff")
)

// Styles
var (
	StyleTitle = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Bold(true)

	StyleLive = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Background(lipgloss.Color("#00ff8820")).
			Padding(0, 1)

	StyleActiveTab = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Bold(true).
			Underline(true)

	StyleInactiveTab = lipgloss.NewStyle().
				Foreground(ColorDim)

	StyleCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	StyleCardLabel = lipgloss.NewStyle().
			Foreground(ColorSubtle).
			Bold(true)

	StyleValueGreen = lipgloss.NewStyle().
			Foreground(ColorGreen).
			Bold(true)

	StyleValueCyan = lipgloss.NewStyle().
			Foreground(ColorCyan).
			Bold(true)

	StyleValueRed = lipgloss.NewStyle().
			Foreground(ColorRed).
			Bold(true)

	StyleValuePurple = lipgloss.NewStyle().
				Foreground(ColorPurple).
				Bold(true)

	StyleDim = lipgloss.NewStyle().
			Foreground(ColorDim)

	StyleSubtle = lipgloss.NewStyle().
			Foreground(ColorSubtle)

	StyleBold = lipgloss.NewStyle().
			Foreground(ColorText).
			Bold(true)

	StyleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder)

	StyleStatusRunning = lipgloss.NewStyle().
				Foreground(ColorGreen)

	StyleStatusStopped = lipgloss.NewStyle().
				Foreground(ColorRed)

	StyleHelp = lipgloss.NewStyle().
			Foreground(ColorDim).
			Padding(1, 0)
)
