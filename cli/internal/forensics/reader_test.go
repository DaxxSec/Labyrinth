package forensics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseJSONLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-session.jsonl")

	content := `{"timestamp":"2026-02-26T14:32:00Z","session_id":"LAB-001","layer":1,"event":"connection","data":{"source_ip":"10.0.1.5"}}
{"timestamp":"2026-02-26T14:33:00Z","session_id":"LAB-001","layer":2,"event":"enumerate","data":{"path":"/etc"}}
{"timestamp":"2026-02-26T14:35:00Z","session_id":"LAB-001","layer":2,"event":"escalation","data":{}}
`
	os.WriteFile(path, []byte(content), 0644)

	events, err := ParseJSONLFile(path)
	if err != nil {
		t.Fatalf("ParseJSONLFile failed: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	if events[0].SessionID != "LAB-001" {
		t.Errorf("events[0].SessionID = %q, want %q", events[0].SessionID, "LAB-001")
	}
	if events[0].Layer != 1 {
		t.Errorf("events[0].Layer = %d, want 1", events[0].Layer)
	}
	if events[1].Event != "enumerate" {
		t.Errorf("events[1].Event = %q, want %q", events[1].Event, "enumerate")
	}
	if events[2].Event != "escalation" {
		t.Errorf("events[2].Event = %q, want %q", events[2].Event, "escalation")
	}
}

func TestStatsAggregation(t *testing.T) {
	dir := t.TempDir()
	sessionsDir := filepath.Join(dir, "sessions")
	promptsDir := filepath.Join(dir, "prompts")
	os.MkdirAll(sessionsDir, 0755)
	os.MkdirAll(promptsDir, 0755)

	// Create 2 session files with 3 and 2 events
	os.WriteFile(filepath.Join(sessionsDir, "s1.jsonl"), []byte(
		`{"session_id":"s1","event":"a"}
{"session_id":"s1","event":"b"}
{"session_id":"s1","event":"c"}
`), 0644)
	os.WriteFile(filepath.Join(sessionsDir, "s2.jsonl"), []byte(
		`{"session_id":"s2","event":"a"}
{"session_id":"s2","event":"b"}
`), 0644)

	// Create 1 prompt file
	os.WriteFile(filepath.Join(promptsDir, "p1.txt"), []byte("captured prompt"), 0644)

	reader := NewReader(dir)
	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats failed: %v", err)
	}

	if stats.ActiveSessions != 2 {
		t.Errorf("ActiveSessions = %d, want 2", stats.ActiveSessions)
	}
	if stats.CapturedPrompts != 1 {
		t.Errorf("CapturedPrompts = %d, want 1", stats.CapturedPrompts)
	}
	if stats.TotalEvents != 5 {
		t.Errorf("TotalEvents = %d, want 5", stats.TotalEvents)
	}
}

func TestEmptyForensicsDir(t *testing.T) {
	dir := t.TempDir()
	reader := NewReader(dir)

	stats, err := reader.ReadStats()
	if err != nil {
		t.Fatalf("ReadStats failed: %v", err)
	}

	if stats.ActiveSessions != 0 || stats.CapturedPrompts != 0 || stats.TotalEvents != 0 {
		t.Errorf("Expected zeroed stats, got %+v", stats)
	}

	sessions, err := reader.ReadSessions()
	if err != nil {
		t.Fatalf("ReadSessions failed: %v", err)
	}
	if len(sessions) != 0 {
		t.Errorf("Expected 0 sessions, got %d", len(sessions))
	}
}

func TestMalformedJSONLLineSkipped(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "malformed.jsonl")

	content := `{"session_id":"good1","event":"ok"}
this is not valid json
{"session_id":"good2","event":"ok"}
`
	os.WriteFile(path, []byte(content), 0644)

	events, err := ParseJSONLFile(path)
	if err != nil {
		t.Fatalf("ParseJSONLFile failed: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 valid events (skipping malformed), got %d", len(events))
	}
}
