package api

// Stats matches the /api/stats response from the dashboard.
type Stats struct {
	ActiveSessions int `json:"active_sessions"`
	CapturedPrompts int `json:"captured_prompts"`
	TotalEvents    int `json:"total_events"`
}

// SessionEntry matches the /api/sessions response entries.
type SessionEntry struct {
	File   string `json:"file"`
	Events int    `json:"events"`
	Last   string `json:"last"` // Last JSONL line as raw string
}

// ForensicEvent matches the JSONL schema from session_logger.py.
type ForensicEvent struct {
	Timestamp string                 `json:"timestamp"`
	SessionID string                 `json:"session_id"`
	Layer     int                    `json:"layer"`
	Event     string                 `json:"event"`
	Data      map[string]interface{} `json:"data,omitempty"`
}
