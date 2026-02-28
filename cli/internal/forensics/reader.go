package forensics

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/DaxxSec/labyrinth/cli/internal/api"
)

// Reader provides direct access to forensic JSONL files.
type Reader struct {
	dir string // e.g. /var/labyrinth/forensics
}

// NewReader creates a forensics reader for the given directory.
func NewReader(dir string) *Reader {
	return &Reader{dir: dir}
}

// ReadStats aggregates stats from forensic files on disk.
func (r *Reader) ReadStats() (api.Stats, error) {
	var stats api.Stats

	sessionsDir := filepath.Join(r.dir, "sessions")
	sessionFiles, err := filepath.Glob(filepath.Join(sessionsDir, "*.jsonl"))
	if err != nil {
		return stats, err
	}
	stats.ActiveSessions = len(sessionFiles)

	promptsDir := filepath.Join(r.dir, "prompts")
	promptFiles, err := filepath.Glob(filepath.Join(promptsDir, "*.txt"))
	if err != nil {
		return stats, err
	}
	stats.CapturedPrompts = len(promptFiles)

	for _, f := range sessionFiles {
		events, _ := ParseJSONLFile(f)
		stats.TotalEvents += len(events)
		for _, ev := range events {
			switch ev.Event {
			case "blindfold_activated":
				stats.L3Activations++
			case "api_intercepted":
				stats.L4Interceptions++
			}
			if ev.Data != nil {
				if d, ok := ev.Data["depth"]; ok {
					if depth, ok := d.(float64); ok && int(depth) > stats.MaxDepthReached {
						stats.MaxDepthReached = int(depth)
					}
				}
			}
		}
	}

	// Auth events
	authFile := filepath.Join(r.dir, "auth_events.jsonl")
	authEvents, _ := ParseJSONLFile(authFile)
	stats.AuthAttempts = len(authEvents)
	stats.TotalEvents += stats.AuthAttempts

	// HTTP events
	httpFile := filepath.Join(r.dir, "http.jsonl")
	httpEvents, _ := ParseJSONLFile(httpFile)
	stats.HTTPRequests = len(httpEvents)
	stats.TotalEvents += stats.HTTPRequests

	return stats, nil
}

// ReadSessions returns session entries from the forensics directory.
func (r *Reader) ReadSessions() ([]api.SessionEntry, error) {
	sessionsDir := filepath.Join(r.dir, "sessions")
	files, err := filepath.Glob(filepath.Join(sessionsDir, "*.jsonl"))
	if err != nil {
		return nil, err
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))
	if len(files) > 50 {
		files = files[:50]
	}

	var sessions []api.SessionEntry
	for _, f := range files {
		lines, err := readAllLines(f)
		if err != nil {
			continue
		}
		last := ""
		if len(lines) > 0 {
			last = lines[len(lines)-1]
		}
		sessions = append(sessions, api.SessionEntry{
			File:   filepath.Base(f),
			Events: len(lines),
			Last:   last,
		})
	}

	return sessions, nil
}

// ReadEvents parses all events from a single JSONL session file.
func (r *Reader) ReadEvents(sessionFile string) ([]api.ForensicEvent, error) {
	path := filepath.Join(r.dir, "sessions", sessionFile)
	return ParseJSONLFile(path)
}

// ReadAuthEvents reads auth credential captures from auth_events.jsonl.
func (r *Reader) ReadAuthEvents() ([]api.AuthEvent, error) {
	path := filepath.Join(r.dir, "auth_events.jsonl")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var events []api.AuthEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var ev api.AuthEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		events = append(events, ev)
	}

	return events, scanner.Err()
}

// ReadAllEvents merges all JSONL sources into a unified event stream, sorted by timestamp desc.
func (r *Reader) ReadAllEvents(limit int) ([]api.ForensicEvent, error) {
	var all []api.ForensicEvent

	// Session events
	sessionsDir := filepath.Join(r.dir, "sessions")
	files, _ := filepath.Glob(filepath.Join(sessionsDir, "*.jsonl"))
	for _, f := range files {
		events, _ := ParseJSONLFile(f)
		for i := range events {
			events[i].Source = "session"
		}
		all = append(all, events...)
	}

	// Auth events
	authFile := filepath.Join(r.dir, "auth_events.jsonl")
	authRaw, _ := ParseJSONLFile(authFile)
	for _, ev := range authRaw {
		all = append(all, api.ForensicEvent{
			Timestamp: ev.Timestamp,
			SessionID: "",
			Layer:     1,
			Event:     ev.Event,
			Data:      ev.Data,
			Source:    "auth",
		})
	}

	// HTTP events
	httpFile := filepath.Join(r.dir, "http.jsonl")
	httpRaw, _ := ParseJSONLFile(httpFile)
	for _, ev := range httpRaw {
		all = append(all, api.ForensicEvent{
			Timestamp: ev.Timestamp,
			SessionID: "",
			Layer:     1,
			Event:     ev.Event,
			Data:      ev.Data,
			Source:    "http",
		})
	}

	// Sort by timestamp desc
	sort.Slice(all, func(i, j int) bool {
		return all[i].Timestamp > all[j].Timestamp
	})

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}

	return all, nil
}

// ReadSessionDetail reads and analyzes a single session's complete timeline.
func (r *Reader) ReadSessionDetail(sessionID string) (*api.SessionDetail, error) {
	path := filepath.Join(r.dir, "sessions", sessionID+".jsonl")
	events, err := ParseJSONLFile(path)
	if err != nil {
		return nil, err
	}

	detail := &api.SessionDetail{
		SessionID: sessionID,
		Events:    events,
	}

	layerSet := make(map[int]bool)
	for _, ev := range events {
		layerSet[ev.Layer] = true
		if ev.Event == "blindfold_activated" {
			detail.L3Activated = true
		}
		if ev.Data != nil {
			if d, ok := ev.Data["depth"]; ok {
				if depth, ok := d.(float64); ok && int(depth) > detail.MaxDepth {
					detail.MaxDepth = int(depth)
				}
			}
		}
	}

	for l := range layerSet {
		detail.LayersTriggered = append(detail.LayersTriggered, l)
	}
	sort.Ints(detail.LayersTriggered)

	if len(events) > 0 {
		detail.FirstSeen = events[0].Timestamp
		detail.LastSeen = events[len(events)-1].Timestamp
	}

	// Check for prompt file
	promptPath := filepath.Join(r.dir, "prompts", sessionID+".txt")
	if data, err := os.ReadFile(promptPath); err == nil {
		detail.HasPrompts = true
		detail.PromptText = string(data)
	}

	return detail, nil
}

// ComputeLayerStatus derives layer status from all forensic data on disk.
func (r *Reader) ComputeLayerStatus() ([]api.LayerStatus, error) {
	layers := []api.LayerStatus{
		{Name: "L0: FOUNDATION", Status: "standby"},
		{Name: "L1: THRESHOLD", Status: "standby"},
		{Name: "L2: MIRAGE", Status: "standby"},
		{Name: "L3: BLINDFOLD", Status: "standby"},
		{Name: "L4: INTERCEPT", Status: "standby"},
	}

	sessionsDir := filepath.Join(r.dir, "sessions")
	files, _ := filepath.Glob(filepath.Join(sessionsDir, "*.jsonl"))

	if len(files) > 0 {
		// If we have any session data, L0 and L1 were active
		layers[0].Status = "active"
		layers[0].Detail = "Forensic data present"
		layers[1].Status = "active"
		layers[1].Detail = "Portal trap sessions recorded"
	}

	l2Sessions := make(map[string]bool)
	l3Sessions := make(map[string]bool)
	l4Sessions := make(map[string]bool)

	for _, f := range files {
		events, _ := ParseJSONLFile(f)
		for _, ev := range events {
			switch ev.Event {
			case "container_spawned", "depth_increase":
				l2Sessions[ev.SessionID] = true
			case "blindfold_activated":
				l3Sessions[ev.SessionID] = true
			case "api_intercepted":
				l4Sessions[ev.SessionID] = true
			}
		}
	}

	if len(l2Sessions) > 0 {
		layers[2].Status = "active"
		layers[2].Detail = "Adaptive filesystem engaged"
		layers[2].Sessions = len(l2Sessions)
	}
	if len(l3Sessions) > 0 {
		layers[3].Status = "active"
		layers[3].Detail = "Blindfold activated"
		layers[3].Sessions = len(l3Sessions)
	}
	if len(l4Sessions) > 0 {
		layers[4].Status = "active"
		layers[4].Detail = "API interception active"
		layers[4].Sessions = len(l4Sessions)
	}

	return layers, nil
}

// ReadPrompts parses captured AI prompts from prompts/*.txt files.
func (r *Reader) ReadPrompts() ([]api.CapturedPrompt, error) {
	promptsDir := filepath.Join(r.dir, "prompts")
	files, err := filepath.Glob(filepath.Join(promptsDir, "*.txt"))
	if err != nil {
		return nil, err
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	var prompts []api.CapturedPrompt
	for _, f := range files {
		sessionID := strings.TrimSuffix(filepath.Base(f), ".txt")
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		content := string(data)

		// Parse sections separated by "--- TIMESTAMP | DOMAIN ---"
		sections := strings.Split(content, "---")
		for i := 0; i < len(sections); i++ {
			section := strings.TrimSpace(sections[i])
			if section == "" {
				continue
			}
			if strings.Contains(section, "|") && i+1 < len(sections) {
				parts := strings.SplitN(section, "|", 2)
				timestamp := strings.TrimSpace(parts[0])
				domain := ""
				if len(parts) > 1 {
					domain = strings.TrimSpace(parts[1])
				}
				i++
				text := ""
				if i < len(sections) {
					text = strings.TrimSpace(sections[i])
				}
				prompts = append(prompts, api.CapturedPrompt{
					SessionID: sessionID,
					Timestamp: timestamp,
					Domain:    domain,
					Text:      text,
				})
			} else {
				prompts = append(prompts, api.CapturedPrompt{
					SessionID: sessionID,
					Text:      section,
				})
			}
		}
	}

	return prompts, nil
}

// ParseJSONLFile reads a JSONL file and returns parsed forensic events.
// Malformed lines are silently skipped.
func ParseJSONLFile(path string) ([]api.ForensicEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var events []api.ForensicEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var ev api.ForensicEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue // Skip malformed lines
		}
		events = append(events, ev)
	}

	return events, scanner.Err()
}

func countLines(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

func readAllLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
