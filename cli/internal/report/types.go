package report

import "github.com/DaxxSec/labyrinth/cli/internal/api"

// Report holds the complete forensic attack report for a session.
type Report struct {
	SessionID     string
	GeneratedAt   string
	Summary       ExecutiveSummary
	Timeline      []TimelineEntry
	Credentials   CredentialReport
	Tools         ToolsAnalysis
	Services      []ServiceInteraction
	AttackGraph   string // Mermaid syntax
	Prompts       []api.CapturedPrompt
	Effectiveness EffectivenessAssessment
}

// ExecutiveSummary holds high-level metrics for the report header.
type ExecutiveSummary struct {
	Duration       string  // human-readable "4m 32s"
	DurationSecs   float64
	LayersReached  []int
	MaxDepth       int
	ConfusionScore int
	RiskLevel      string // Low/Medium/High/Critical
	TotalEvents    int
	L3Activated    bool
	L4Active       bool
	AttackerType   string // "AI Agent" / "Human" / "Automated Scanner"
	FirstSeen      string
	LastSeen       string
}

// TimelineEntry is a single event in the attack timeline with MITRE mapping.
type TimelineEntry struct {
	Timestamp   string
	Layer       int
	Event       string
	Description string
	MITRETactic string
	MITRETechID string
	Data        map[string]interface{}
}

// CredentialReport summarises planted vs captured credentials.
type CredentialReport struct {
	BaitCreds     []BaitCredStatus
	CapturedAuth  []api.AuthEvent
	MatchedBait   int
	NovelAttempts int
}

// BaitCredStatus tracks whether a planted credential was used.
type BaitCredStatus struct {
	Service  string
	Username string
	WasUsed  bool
}

// ToolsAnalysis describes the attacker's tooling fingerprint.
type ToolsAnalysis struct {
	UserAgent       string
	SDKDetected     string // parsed from user agent
	APIKeys         []string
	Models          []string
	ToolCount       int
	Domains         []string
	ToolInventory   []ToolEntry
	CommandPatterns []CommandPattern
}

// ToolEntry is a named tool with its invocation count.
type ToolEntry struct {
	Name  string
	Count int
}

// CommandPattern groups similar commands into categories.
type CommandPattern struct {
	Pattern  string
	Count    int
	Category string // enumeration, credential_access, lateral_movement
}

// ServiceInteraction summarises attacker engagement with a phantom service.
type ServiceInteraction struct {
	Protocol      string
	Port          int
	Connections   int
	AuthAttempts  int
	Queries       int
	Credentials   []string
	SampleQueries []string // first 3 queries for context
}

// EffectivenessAssessment evaluates how well the deception worked.
type EffectivenessAssessment struct {
	DeceptionWorked     []string
	DeceptionFailed     []string
	TimeWasted          string // human-readable duration
	CredentialsCaptured int
	IntelligenceGained  []string
}
