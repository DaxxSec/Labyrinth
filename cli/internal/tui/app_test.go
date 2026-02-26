package tui

import "testing"

func TestTabNavigation(t *testing.T) {
	current := TabOverview
	expected := []Tab{TabSessions, TabLayers, TabAnalysis, TabLogs, TabOverview}

	for i, want := range expected {
		current = NextTab(current)
		if current != want {
			t.Errorf("Step %d: NextTab got %d (%s), want %d (%s)",
				i, current, current.Name(), want, want.Name())
		}
	}
}

func TestPrevTab(t *testing.T) {
	current := TabOverview
	prev := PrevTab(current)
	if prev != TabLogs {
		t.Errorf("PrevTab(Overview) = %s, want Logs", prev.Name())
	}

	prev = PrevTab(TabSessions)
	if prev != TabOverview {
		t.Errorf("PrevTab(Sessions) = %s, want Overview", prev.Name())
	}
}

func TestTabDirectSelect(t *testing.T) {
	tab := GotoTab(3)
	if tab != TabAnalysis {
		t.Errorf("GotoTab(3) = %s, want Analysis", tab.Name())
	}
}

func TestTabDirectSelectClamped(t *testing.T) {
	tab := GotoTab(100)
	if tab != TabLogs {
		t.Errorf("GotoTab(100) = %s, want Logs (clamped)", tab.Name())
	}

	tab = GotoTab(-1)
	if tab != TabOverview {
		t.Errorf("GotoTab(-1) = %s, want Overview (clamped)", tab.Name())
	}
}

func TestInitialState(t *testing.T) {
	app := NewApp()
	if app.activeTab != TabOverview {
		t.Errorf("Initial tab = %s, want Overview", app.activeTab.Name())
	}
}

func TestTabNames(t *testing.T) {
	names := []struct {
		tab  Tab
		name string
	}{
		{TabOverview, "Overview"},
		{TabSessions, "Sessions"},
		{TabLayers, "Layers"},
		{TabAnalysis, "Analysis"},
		{TabLogs, "Logs"},
	}

	for _, tt := range names {
		if tt.tab.Name() != tt.name {
			t.Errorf("Tab %d name = %q, want %q", tt.tab, tt.tab.Name(), tt.name)
		}
	}
}
