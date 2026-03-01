package api

// Stats matches the /api/stats response from the dashboard.
type Stats struct {
	ActiveSessions  int `json:"active_sessions"`
	CapturedPrompts int `json:"captured_prompts"`
	TotalEvents     int `json:"total_events"`
	AuthAttempts    int `json:"auth_attempts"`
	HTTPRequests    int `json:"http_requests"`
	L3Activations   int `json:"l3_activations"`
	L4Interceptions int `json:"l4_interceptions"`
	MaxDepthReached int `json:"max_depth_reached"`
	ActiveContainers int `json:"active_containers"`
}

// SessionEntry matches the /api/sessions response entries.
type SessionEntry struct {
	File        string `json:"file"`
	Events      int    `json:"events"`
	Last        string `json:"last"`                   // Last JSONL line as raw string
	Environment string `json:"environment,omitempty"`   // Set by MultiClient for aggregated views
}

// ForensicEvent matches the JSONL schema from session_logger.py.
type ForensicEvent struct {
	Timestamp string                 `json:"timestamp"`
	SessionID string                 `json:"session_id"`
	Layer     int                    `json:"layer"`
	Event     string                 `json:"event"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Source    string                 `json:"source,omitempty"`
}

// AuthEvent represents a credential capture from auth_events.jsonl.
type AuthEvent struct {
	Timestamp string `json:"timestamp"`
	Service   string `json:"service"`
	SrcIP     string `json:"src_ip"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Event     string `json:"event,omitempty"`
}

// ContainerStatus represents a Docker container's status.
type ContainerStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	State  string `json:"state"`
	Ports  string `json:"ports"`
	Layer  string `json:"layer"`
}

// ContainersResponse holds infrastructure and session containers.
type ContainersResponse struct {
	Infrastructure []ContainerStatus `json:"infrastructure"`
	Sessions       []ContainerStatus `json:"sessions"`
}

// LayerStatus represents the live status of a LABYRINTH layer.
type LayerStatus struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Detail   string `json:"detail"`
	Sessions int    `json:"sessions"`
}

// SessionDetail holds the full event timeline for a single session.
type SessionDetail struct {
	SessionID       string         `json:"session_id"`
	Events          []ForensicEvent `json:"events"`
	MaxDepth        int            `json:"max_depth"`
	L3Activated     bool           `json:"l3_activated"`
	LayersTriggered []int          `json:"layers_triggered"`
	FirstSeen       string         `json:"first_seen"`
	LastSeen        string         `json:"last_seen"`
	HasPrompts      bool           `json:"has_prompts"`
	PromptText      string         `json:"prompt_text"`
}

// CapturedPrompt represents a single captured AI system prompt.
type CapturedPrompt struct {
	SessionID string `json:"session_id"`
	Timestamp string `json:"timestamp"`
	Domain    string `json:"domain"`
	Text      string `json:"text"`
}

// EventsResponse holds a paginated event list.
type EventsResponse struct {
	Events []ForensicEvent `json:"events"`
	Total  int             `json:"total"`
}

// AuthResponse holds auth event results.
type AuthResponse struct {
	AuthEvents []AuthEvent `json:"auth_events"`
}

// LayersResponse holds layer status results.
type LayersResponse struct {
	Layers []LayerStatus `json:"layers"`
}

// PromptsResponse holds captured prompt results.
type PromptsResponse struct {
	Prompts []CapturedPrompt `json:"prompts"`
}

// ResetResponse holds the result of a reset operation.
type ResetResponse struct {
	ContainersRemoved int      `json:"containers_removed"`
	FilesCleared      int      `json:"files_cleared"`
	Errors            []string `json:"errors"`
}

// SessionAnalysis holds post-mortem analysis data for a session.
type SessionAnalysis struct {
	SessionID       string            `json:"session_id"`
	TotalEvents     int               `json:"total_events"`
	DurationSeconds float64           `json:"duration_seconds"`
	LayersReached   []int             `json:"layers_reached"`
	MaxDepth        int               `json:"max_depth"`
	ConfusionScore  int               `json:"confusion_score"`
	Phases          []SessionPhase    `json:"phases"`
	EventBreakdown  map[string]int    `json:"event_breakdown"`
	KeyMoments      []KeyMoment       `json:"key_moments"`
	L3Activated     bool              `json:"l3_activated"`
	L4Active        bool              `json:"l4_active"`
}

// SessionPhase represents a behavioral phase within a session.
type SessionPhase struct {
	Phase  string `json:"phase"`
	Start  string `json:"start"`
	End    string `json:"end"`
	Events int    `json:"events"`
}

// KeyMoment represents a notable event in a session timeline.
type KeyMoment struct {
	Timestamp   string `json:"timestamp"`
	Event       string `json:"event"`
	Description string `json:"description"`
	Layer       int    `json:"layer"`
}

// L4ModeResponse holds the current L4 interceptor mode.
type L4ModeResponse struct {
	Mode       string   `json:"mode"`
	ValidModes []string `json:"valid_modes"`
}

// L4IntelSummary holds intelligence summary for a session.
type L4IntelSummary struct {
	SessionID      string   `json:"session_id,omitempty"`
	InterceptCount int      `json:"intercept_count"`
	FirstSeen      string   `json:"first_seen"`
	LastSeen       string   `json:"last_seen"`
	APIKeys        []string `json:"api_keys"`
	KeyType        string   `json:"key_type"`
	Models         []string `json:"models"`
	UserAgent      string   `json:"user_agent"`
	OpenAIOrg      string   `json:"openai_org,omitempty"`
	OpenAIProject  string   `json:"openai_project,omitempty"`
	ToolCount      int      `json:"tool_count"`
	Domains        []string `json:"domains"`
}

// L4IntelResponse holds the list of intelligence reports.
type L4IntelResponse struct {
	Intel []L4IntelSummary `json:"intel"`
}
