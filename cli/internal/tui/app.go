package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/DaxxSec/labyrinth/cli/internal/api"
	"github.com/DaxxSec/labyrinth/cli/internal/forensics"
	"github.com/DaxxSec/labyrinth/cli/internal/notify"
	"github.com/DaxxSec/labyrinth/cli/internal/registry"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	dashboardURL    = "http://localhost:9000"
	forensicsDir    = "/var/labyrinth/forensics"
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
	activeTab       Tab
	width           int
	height          int
	dataSource      DataSource
	apiClient       *api.Client
	fileReader      *forensics.Reader
	stats           api.Stats
	sessions        []api.SessionEntry
	environments    []registry.Environment
	selectedSession int
	logFilter       string
	err             error
	ready           bool

	// New state fields
	authEvents      []api.AuthEvent
	allEvents       []api.ForensicEvent
	containers      *api.ContainersResponse
	layerStatuses   []api.LayerStatus
	sessionDetail   *api.SessionDetail
	prompts         []api.CapturedPrompt
	logScrollOffset int
	logFilterType   string // filter by event type on Logs tab
	sessionView     int    // 0=list, 1=detail

	// Previous counts for notification delta detection
	prevSessionCount int
	prevAuthCount    int
	prevL3Count      int
	prevL4Count      int

	// Environment switching
	selectedEnv      int               // index into environments (-1 for All)
	activeEnvName    string            // current env name ("" = aggregated/All)
	dashboardURLs    map[string]string // envName → dashboard URL
	envSelectorOpen  bool              // whether env selector bar is visible
	multiClient      *api.MultiClient  // for aggregated "All" view
	targetEnvName    string            // --env flag target
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
type eventsMsg struct {
	events []api.ForensicEvent
	err    error
}
type authMsg struct {
	events []api.AuthEvent
	err    error
}
type containersMsg struct {
	containers *api.ContainersResponse
	err        error
}
type layersMsg struct {
	layers []api.LayerStatus
	err    error
}
type sessionDetailMsg struct {
	detail *api.SessionDetail
	err    error
}
type promptsMsg struct {
	prompts []api.CapturedPrompt
	err     error
}

// NewApp creates a new TUI app. If targetEnv is non-empty, the TUI starts
// focused on that environment.
func NewApp(targetEnv ...string) *App {
	a := &App{
		activeTab:     TabOverview,
		apiClient:     api.NewClient(dashboardURL),
		fileReader:    forensics.NewReader(forensicsDir),
		layerStatuses: defaultLayerStatuses(),
		selectedEnv:   -1,
		dashboardURLs: make(map[string]string),
	}
	if len(targetEnv) > 0 && targetEnv[0] != "" {
		a.targetEnvName = targetEnv[0]
	}
	return a
}

func defaultLayerStatuses() []api.LayerStatus {
	return []api.LayerStatus{
		{Name: "L0: FOUNDATION", Status: "standby", Detail: "Waiting for deployment"},
		{Name: "L1: THRESHOLD", Status: "standby", Detail: "Waiting for deployment"},
		{Name: "L2: MIRAGE", Status: "standby", Detail: "Waiting for sessions"},
		{Name: "L3: BLINDFOLD", Status: "standby", Detail: "Waiting for escalation"},
		{Name: "L4: INTERCEPT", Status: "standby", Detail: "Waiting for API traffic"},
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
		// If env selector is open, handle navigation
		if a.envSelectorOpen {
			switch msg.String() {
			case "e", "escape":
				a.envSelectorOpen = false
				return a, nil
			case "left", "h":
				a.prevEnv()
				return a, tea.Batch(fetchDataCmd(a.apiClient, a.fileReader), a.tabFetchCmd())
			case "right", "l":
				a.nextEnv()
				return a, tea.Batch(fetchDataCmd(a.apiClient, a.fileReader), a.tabFetchCmd())
			case "enter":
				a.envSelectorOpen = false
				return a, nil
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "tab":
			a.activeTab = NextTab(a.activeTab)
			return a, a.tabFetchCmd()
		case "shift+tab":
			a.activeTab = PrevTab(a.activeTab)
			return a, a.tabFetchCmd()
		case "1":
			a.activeTab = GotoTab(0)
			return a, a.tabFetchCmd()
		case "2":
			a.activeTab = GotoTab(1)
			return a, a.tabFetchCmd()
		case "3":
			a.activeTab = GotoTab(2)
			return a, a.tabFetchCmd()
		case "4":
			a.activeTab = GotoTab(3)
			return a, a.tabFetchCmd()
		case "5":
			a.activeTab = GotoTab(4)
			return a, a.tabFetchCmd()
		case "r":
			return a, tea.Batch(fetchDataCmd(a.apiClient, a.fileReader), a.tabFetchCmd())
		case "e":
			if len(a.environments) > 1 {
				a.envSelectorOpen = !a.envSelectorOpen
				return a, nil
			}
		case "enter":
			if a.activeTab == TabSessions && a.sessionView == 0 && len(a.sessions) > 0 {
				a.sessionView = 1
				return a, a.fetchSessionDetailCmd()
			}
		case "escape":
			if a.activeTab == TabSessions && a.sessionView == 1 {
				a.sessionView = 0
				a.sessionDetail = nil
			}
		case "j", "down":
			switch a.activeTab {
			case TabSessions:
				if a.sessionView == 0 && len(a.sessions) > 0 {
					a.selectedSession = min(a.selectedSession+1, len(a.sessions)-1)
				}
			case TabLogs:
				a.logScrollOffset++
			}
		case "k", "up":
			switch a.activeTab {
			case TabSessions:
				if a.sessionView == 0 && a.selectedSession > 0 {
					a.selectedSession--
				}
			case TabLogs:
				if a.logScrollOffset > 0 {
					a.logScrollOffset--
				}
			}
		case "f":
			if a.activeTab == TabLogs {
				a.cycleLogFilter()
			}
		}

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true

	case tickMsg:
		cmds := []tea.Cmd{tickCmd(), fetchDataCmd(a.apiClient, a.fileReader)}
		cmds = append(cmds, a.tabFetchCmd())
		return a, tea.Batch(cmds...)

	case statsMsg:
		if msg.err == nil {
			// Notification delta detection
			if a.stats.ActiveSessions > 0 { // Skip first load
				if msg.stats.ActiveSessions > a.prevSessionCount {
					newCount := msg.stats.ActiveSessions - a.prevSessionCount
					notify.Notify("LABYRINTH", fmt.Sprintf("%d new connection(s) detected", newCount))
				}
				if msg.stats.AuthAttempts > a.prevAuthCount {
					notify.Notify("LABYRINTH", "Credentials captured!")
				}
				if msg.stats.L3Activations > a.prevL3Count {
					notify.Notify("LABYRINTH", "L3 Blindfold activated — attacker escalated")
				}
				if msg.stats.L4Interceptions > a.prevL4Count {
					notify.Notify("LABYRINTH", "L4 API interception — prompt captured!")
				}
			}

			a.prevSessionCount = msg.stats.ActiveSessions
			a.prevAuthCount = msg.stats.AuthAttempts
			a.prevL3Count = msg.stats.L3Activations
			a.prevL4Count = msg.stats.L4Interceptions

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
			a.populateDashboardURLs()

			// Handle --env flag targeting
			if a.targetEnvName != "" {
				for i, env := range a.environments {
					if env.Name == a.targetEnvName {
						a.selectedEnv = i
						a.switchEnvironment()
						break
					}
				}
				a.targetEnvName = "" // only apply once
			} else if len(a.environments) == 1 {
				// Auto-select if only one env
				a.selectedEnv = 0
				a.switchEnvironment()
			}
		}

	case eventsMsg:
		if msg.err == nil {
			a.allEvents = msg.events
		}

	case authMsg:
		if msg.err == nil {
			a.authEvents = msg.events
		}

	case containersMsg:
		if msg.err == nil {
			a.containers = msg.containers
		}

	case layersMsg:
		if msg.err == nil {
			a.layerStatuses = msg.layers
		}

	case sessionDetailMsg:
		if msg.err == nil {
			a.sessionDetail = msg.detail
		}

	case promptsMsg:
		if msg.err == nil {
			a.prompts = msg.prompts
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

	// Environment selector bar (when open or multiple envs)
	envBar := ""
	if a.envSelectorOpen {
		envBar = a.renderEnvSelector()
	}
	if envBar != "" {
		b.WriteString(envBar)
		b.WriteString("\n")
	}

	// Tab bar
	b.WriteString(a.renderTabBar())
	b.WriteString("\n")

	// Separator
	b.WriteString(StyleDim.Render(strings.Repeat("─", a.width)))
	b.WriteString("\n")

	// Tab content
	extraLines := 0
	if envBar != "" {
		extraLines = 1
	}
	contentHeight := a.height - 6 - extraLines // header + tabbar + separator + help
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
	badge := a.modeBadge()

	sourceLabel := ""
	switch a.dataSource {
	case SourceAPI:
		sourceLabel = lipgloss.NewStyle().Foreground(ColorGreen).Render("[LIVE]")
	case SourceFiles:
		sourceLabel = lipgloss.NewStyle().Foreground(ColorYellow).Render("[FILES]")
	case SourceNone:
		sourceLabel = lipgloss.NewStyle().Foreground(ColorRed).Render("[NO DATA]")
	}

	envLabel := ""
	if a.activeEnvName != "" {
		envLabel = StyleSubtle.Render(fmt.Sprintf(" %s ", a.activeEnvName))
	}

	leftPart := title
	if badge != "" {
		leftPart = leftPart + " " + badge
	}
	if envLabel != "" {
		leftPart = leftPart + " " + envLabel
	}

	padding := a.width - lipgloss.Width(leftPart) - lipgloss.Width(sourceLabel) - 4
	if padding < 1 {
		padding = 1
	}

	return fmt.Sprintf("┌─%s%s%s─┐", leftPart, strings.Repeat("─", padding), sourceLabel)
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
	base := "  [Tab] switch  [1-5] direct  [r] refresh  [q] quit"
	if len(a.environments) > 1 {
		base += "  [e] env"
	}
	if a.envSelectorOpen {
		return StyleHelp.Render(base + "  [←/→] select env  [Enter/Esc] close")
	}
	switch a.activeTab {
	case TabSessions:
		if a.sessionView == 1 {
			return StyleHelp.Render(base + "  [Esc] back to list")
		}
		return StyleHelp.Render(base + "  [j/k] select  [Enter] detail")
	case TabLogs:
		filterLabel := "all"
		if a.logFilterType != "" {
			filterLabel = a.logFilterType
		}
		return StyleHelp.Render(fmt.Sprintf("%s  [j/k] scroll  [f] filter (%s)", base, filterLabel))
	default:
		return StyleHelp.Render(base)
	}
}

// tabFetchCmd returns fetch commands for the currently active tab's data needs.
func (a *App) tabFetchCmd() tea.Cmd {
	var cmds []tea.Cmd

	// Always fetch containers and layers
	cmds = append(cmds, fetchContainersCmd(a.apiClient, a.fileReader))
	cmds = append(cmds, fetchLayersCmd(a.apiClient, a.fileReader))

	switch a.activeTab {
	case TabLogs:
		cmds = append(cmds, fetchEventsCmd(a.apiClient, a.fileReader))
	case TabSessions:
		cmds = append(cmds, fetchAuthCmd(a.apiClient, a.fileReader))
		if a.sessionView == 1 {
			cmds = append(cmds, a.fetchSessionDetailCmd())
		}
	case TabAnalysis:
		cmds = append(cmds, fetchPromptsCmd(a.apiClient, a.fileReader))
		cmds = append(cmds, fetchEventsCmd(a.apiClient, a.fileReader))
	}

	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (a *App) cycleLogFilter() {
	filters := []string{"", "connection", "auth_attempt", "container_spawned", "blindfold_activated", "api_intercepted"}
	for i, f := range filters {
		if f == a.logFilterType {
			a.logFilterType = filters[(i+1)%len(filters)]
			a.logScrollOffset = 0
			return
		}
	}
	a.logFilterType = ""
}

func (a *App) fetchSessionDetailCmd() tea.Cmd {
	if a.selectedSession >= len(a.sessions) {
		return nil
	}
	sess := a.sessions[a.selectedSession]
	sessionID := strings.TrimSuffix(sess.File, ".jsonl")
	client := a.apiClient
	reader := a.fileReader
	return func() tea.Msg {
		if client.Healthy() {
			detail, err := client.FetchSessionDetail(sessionID)
			if err == nil {
				return sessionDetailMsg{detail: detail}
			}
		}
		detail, err := reader.ReadSessionDetail(sessionID)
		return sessionDetailMsg{detail: detail, err: err}
	}
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

func fetchEventsCmd(client *api.Client, reader *forensics.Reader) tea.Cmd {
	return func() tea.Msg {
		if client.Healthy() {
			events, err := client.FetchEvents(200)
			if err == nil {
				return eventsMsg{events: events}
			}
		}
		events, err := reader.ReadAllEvents(200)
		return eventsMsg{events: events, err: err}
	}
}

func fetchAuthCmd(client *api.Client, reader *forensics.Reader) tea.Cmd {
	return func() tea.Msg {
		if client.Healthy() {
			events, err := client.FetchAuthEvents(50)
			if err == nil {
				return authMsg{events: events}
			}
		}
		events, err := reader.ReadAuthEvents()
		return authMsg{events: events, err: err}
	}
}

func fetchContainersCmd(client *api.Client, reader *forensics.Reader) tea.Cmd {
	return func() tea.Msg {
		if client.Healthy() {
			containers, err := client.FetchContainers()
			if err == nil {
				return containersMsg{containers: containers}
			}
		}
		// No file-based fallback for containers
		return containersMsg{err: fmt.Errorf("no container data")}
	}
}

func fetchLayersCmd(client *api.Client, reader *forensics.Reader) tea.Cmd {
	return func() tea.Msg {
		if client.Healthy() {
			layers, err := client.FetchLayers()
			if err == nil {
				return layersMsg{layers: layers}
			}
		}
		layers, err := reader.ComputeLayerStatus()
		return layersMsg{layers: layers, err: err}
	}
}

func fetchPromptsCmd(client *api.Client, reader *forensics.Reader) tea.Cmd {
	return func() tea.Msg {
		if client.Healthy() {
			prompts, err := client.FetchPrompts()
			if err == nil {
				return promptsMsg{prompts: prompts}
			}
		}
		prompts, err := reader.ReadPrompts()
		return promptsMsg{prompts: prompts, err: err}
	}
}

// populateDashboardURLs builds the envName → dashboardURL map from the environment list.
func (a *App) populateDashboardURLs() {
	for _, env := range a.environments {
		if env.DashboardURL != "" {
			a.dashboardURLs[env.Name] = env.DashboardURL
		} else if env.Type == "test" {
			a.dashboardURLs[env.Name] = dashboardURL // default test URL
		}
	}

	// Build multi-client for "All" aggregated view
	var urls []string
	var names []string
	for _, env := range a.environments {
		if url, ok := a.dashboardURLs[env.Name]; ok {
			urls = append(urls, url)
			names = append(names, env.Name)
		}
	}
	if len(urls) > 0 {
		a.multiClient = api.NewMultiClient(urls, names)
	}
}

// switchEnvironment updates the apiClient to point at the selected environment.
func (a *App) switchEnvironment() {
	if a.selectedEnv == -1 {
		// "All" mode — use default client, aggregation happens via multiClient
		a.activeEnvName = ""
		a.apiClient = api.NewClient(dashboardURL)
		return
	}

	if a.selectedEnv >= 0 && a.selectedEnv < len(a.environments) {
		env := a.environments[a.selectedEnv]
		a.activeEnvName = env.Name
		if url, ok := a.dashboardURLs[env.Name]; ok {
			a.apiClient = api.NewClient(url)
		}
	}
}

// nextEnv moves to the next environment in the selector.
func (a *App) nextEnv() {
	if len(a.environments) == 0 {
		return
	}
	// -1 → 0 → 1 → ... → len-1 → -1
	a.selectedEnv++
	if a.selectedEnv >= len(a.environments) {
		a.selectedEnv = -1
	}
	a.switchEnvironment()
}

// prevEnv moves to the previous environment in the selector.
func (a *App) prevEnv() {
	if len(a.environments) == 0 {
		return
	}
	// -1 → len-1 → len-2 → ... → 0 → -1
	a.selectedEnv--
	if a.selectedEnv < -1 {
		a.selectedEnv = len(a.environments) - 1
	}
	a.switchEnvironment()
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
