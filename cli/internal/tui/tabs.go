package tui

// Tab represents a dashboard tab.
type Tab int

const (
	TabOverview Tab = iota
	TabSessions
	TabLayers
	TabAnalysis
	TabLogs
	TabEnvironment
	tabCount // sentinel for wrap-around
)

var tabNames = []string{
	"Overview",
	"Sessions",
	"Layers",
	"Intel",
	"Logs",
	"Environment",
}

// Name returns the display name for a tab.
func (t Tab) Name() string {
	if int(t) < len(tabNames) {
		return tabNames[t]
	}
	return "Unknown"
}

// NextTab returns the next tab, wrapping around.
func NextTab(current Tab) Tab {
	return (current + 1) % tabCount
}

// PrevTab returns the previous tab, wrapping around.
func PrevTab(current Tab) Tab {
	if current == 0 {
		return tabCount - 1
	}
	return current - 1
}

// GotoTab returns a specific tab by index (clamped).
func GotoTab(index int) Tab {
	if index < 0 {
		return 0
	}
	if index >= int(tabCount) {
		return tabCount - 1
	}
	return Tab(index)
}
