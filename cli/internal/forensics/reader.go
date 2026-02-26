package forensics

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

	"github.com/ItzDaxxy/labyrinth/cli/internal/api"
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
	sessionFiles, _ := filepath.Glob(filepath.Join(sessionsDir, "*.jsonl"))
	stats.ActiveSessions = len(sessionFiles)

	promptsDir := filepath.Join(r.dir, "prompts")
	promptFiles, _ := filepath.Glob(filepath.Join(promptsDir, "*.txt"))
	stats.CapturedPrompts = len(promptFiles)

	for _, f := range sessionFiles {
		count, _ := countLines(f)
		stats.TotalEvents += count
	}

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
