package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/DaxxSec/labyrinth/cli/internal/api"
	"github.com/DaxxSec/labyrinth/cli/internal/forensics"
	"github.com/DaxxSec/labyrinth/cli/internal/registry"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	dashboardURL   = "http://localhost:9000"
	forensicsDir   = "/var/labyrinth/forensics"
	refreshInterval = 2 * time.Second
)

// DataSource indicates how data is being fetched.
type DataSource int

const (
	SourceNone DataSource = iota
	SourceAPI
	SourceFiles
)

// App is the main bubbletea model.
type App struct {
	activeTab    Tab
	width        int
	height       int
	dataSource   DataSource
	apiClient    *api.Client
	fileReader   *forensics.Reader
	stats        api.Stats
	sessions     []api.SessionEntry
	environments []registry.Environment
	selectedSession int
	logLines     []string
	logFilter    string
	layerConfig  *layerInfo
	err          error
	ready        bool
}

type layerInfo struct {
	layers [5]layerDetail
}

type layerDetail struct {
	name   string
	status string
	detail string
}

// Messages
type tickMsg time.Time
type statsMsg struct {
	stats    api.Stats
	sessions []api.SessionEntry
	source   DataSource
	err      error
}
type envsMsg struct {
	envs []registry.Environment
	err  error
}

// NewApp creates a new TUI app.
func NewApp() *App {
	return &App{
		activeTab:   TabOverview,
		apiClient:   api.NewClient(dashboardURL),
		fileReader:  forensics.NewReader(forensicsDir),
		layerConfig: defaultLayerInfo(),
	}
}

func defaultLayerInfo() *layerInfo {
	return &layerInfo{
		layers: [5]layerDetail{
			{name: "L0: FOUNDATION", status: "active", detail: "Encryption: AES-256-GCM | Network isolation | Retention policy"},
			{name: "L1: THRESHOLD", status: "active", detail: "SSH portal trap (:2222) | HTTP portal trap (:8080) | Session logging"},
			{name: "L2: MIRAGE", status: "standby", detail: "Adaptive filesystem | Contradiction density: medium"},
			{name: "L3: BLINDFOLD", status: "standby", detail: "Activation: on_escalation | Method: bashrc_payload"},
			{name: "L4: INTERCEPT", status: "standby", detail: "Mode: auto | Default swap: passive | Prompt logging: on"},
		},
	}
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		fetchDataCmd(a.apiClient, a.fileReader),
		fetchEnvsCmd(),
	)
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "tab":
			a.activeTab = NextTab(a.activeTab)
		case "shift+tab":
			a.activeTab = PrevTab(a.activeTab)
		case "1":
			a.activeTab = GotoTab(0)
		case "2":
			a.activeTab = GotoTab(1)
		case "3":
			a.activeTab = GotoTab(2)
		case "4":
			a.activeTab = GotoTab(3)
		case "5":
			a.activeTab = GotoTab(4)
		case "r":
			return a, fetchDataCmd(a.apiClient, a.fileReader)
		case "j", "down":
			if a.activeTab == TabSessions && len(a.sessions) > 0 {
				a.selectedSession = min(a.selectedSession+1, len(a.sessions)-1)
			}
		case "k", "up":
			if a.activeTab == TabSessions && a.selectedSession > 0 {
				a.selectedSession--
			}
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true

	case tickMsg:
		return a, tea.Batch(tickCmd(), fetchDataCmd(a.apiClient, a.fileReader))

	case statsMsg:
		if msg.err == nil {
			a.stats = msg.stats
			a.sessions = msg.sessions
			a.dataSource = msg.source
			a.err = nil
		} else {
			a.err = msg.err
		}

	case envsMsg:
		if msg.err == nil {
			a.environments = msg.envs
		}
	}

	return a, nil
}

func (a *App) View() tea.View {
	if !a.ready {
		v := tea.NewView("  Initializing LABYRINTH TUI...")
		v.AltScreen = true
		return v
	}

	var b strings.Builder

	// Header
	b.WriteString(a.renderHeader())
	b.WriteString("\n")

	// Tab bar
	b.WriteString(a.renderTabBar())
	b.WriteString("\n")

	// Separator
	b.WriteString(StyleDim.Render(strings.Repeat("─", a.width)))
	b.WriteString("\n")

	// Tab content
	contentHeight := a.height - 6 // header + tabbar + separator + help
	switch a.activeTab {
	case TabOverview:
		b.WriteString(renderOverview(a, contentHeight))
	case TabSessions:
		b.WriteString(renderSessions(a, contentHeight))
	case TabLayers:
		b.WriteString(renderLayers(a, contentHeight))
	case TabAnalysis:
		b.WriteString(renderAnalysis(a, contentHeight))
	case TabLogs:
		b.WriteString(renderLogs(a, contentHeight))
	}

	// Help bar
	b.WriteString("\n")
	b.WriteString(a.renderHelp())

	v := tea.NewView(b.String())
	v.AltScreen = true
	return v
}

func (a *App) renderHeader() string {
	title := StyleTitle.Render(" LABYRINTH ")

	sourceLabel := ""
	switch a.dataSource {
	case SourceAPI:
		sourceLabel = lipgloss.NewStyle().Foreground(ColorGreen).Render("[LIVE]")
	case SourceFiles:
		sourceLabel = lipgloss.NewStyle().Foreground(ColorYellow).Render("[FILES]")
	case SourceNone:
		sourceLabel = lipgloss.NewStyle().Foreground(ColorRed).Render("[NO DATA]")
	}

	padding := a.width - lipgloss.Width(title) - lipgloss.Width(sourceLabel) - 4
	if padding < 1 {
		padding = 1
	}

	return fmt.Sprintf("┌─%s%s%s─┐", title, strings.Repeat("─", padding), sourceLabel)
}

func (a *App) renderTabBar() string {
	var tabs []string
	for i := 0; i < int(tabCount); i++ {
		tab := Tab(i)
		name := fmt.Sprintf(" %s ", tab.Name())
		if tab == a.activeTab {
			tabs = append(tabs, StyleActiveTab.Render(name))
		} else {
			tabs = append(tabs, StyleInactiveTab.Render(name))
		}
	}
	return "│ " + strings.Join(tabs, " │ ") + " │"
}

func (a *App) renderHelp() string {
	return StyleHelp.Render("  [Tab] switch tabs  [1-5] direct tab  [r] refresh  [q] quit")
}

// Commands
func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchDataCmd(client *api.Client, reader *forensics.Reader) tea.Cmd {
	return func() tea.Msg {
		// Try API first
		if client.Healthy() {
			stats, err := client.FetchStats()
			if err == nil {
				sessions, _ := client.FetchSessions()
				return statsMsg{stats: stats, sessions: sessions, source: SourceAPI}
			}
		}

		// Fallback to files
		stats, err := reader.ReadStats()
		if err == nil && (stats.ActiveSessions > 0 || stats.TotalEvents > 0) {
			sessions, _ := reader.ReadSessions()
			return statsMsg{stats: stats, sessions: sessions, source: SourceFiles}
		}

		return statsMsg{source: SourceNone, err: fmt.Errorf("no data source available")}
	}
}

func fetchEnvsCmd() tea.Cmd {
	return func() tea.Msg {
		reg := registry.New("")
		envs, err := reg.ListAll()
		return envsMsg{envs: envs, err: err}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
