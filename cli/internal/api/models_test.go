package api

import (
	"encoding/json"
	"testing"
)

func TestDeserializeStats(t *testing.T) {
	raw := `{"active_sessions":3,"captured_prompts":7,"total_events":142}`

	var stats Stats
	if err := json.Unmarshal([]byte(raw), &stats); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if stats.ActiveSessions != 3 {
		t.Errorf("ActiveSessions = %d, want 3", stats.ActiveSessions)
	}
	if stats.CapturedPrompts != 7 {
		t.Errorf("CapturedPrompts = %d, want 7", stats.CapturedPrompts)
	}
	if stats.TotalEvents != 142 {
		t.Errorf("TotalEvents = %d, want 142", stats.TotalEvents)
	}
}

func TestDeserializeSessionEntry(t *testing.T) {
	raw := `[{"file":"LAB-001.jsonl","events":42,"last":"{\"timestamp\":\"2026-02-26T14:32:00Z\"}"}]`

	var sessions []SessionEntry
	if err := json.Unmarshal([]byte(raw), &sessions); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session, got %d", len(sessions))
	}

	s := sessions[0]
	if s.File != "LAB-001.jsonl" {
		t.Errorf("File = %q, want %q", s.File, "LAB-001.jsonl")
	}
	if s.Events != 42 {
		t.Errorf("Events = %d, want 42", s.Events)
	}
	if s.Last == "" {
		t.Error("Last should not be empty")
	}
}

func TestDeserializeForensicEvent(t *testing.T) {
	raw := `{"timestamp":"2026-02-26T14:32:00Z","session_id":"LAB-TEST-001","layer":1,"event":"connection","data":{"source_ip":"127.0.0.1","service":"ssh"}}`

	var ev ForensicEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if ev.Timestamp != "2026-02-26T14:32:00Z" {
		t.Errorf("Timestamp = %q, want %q", ev.Timestamp, "2026-02-26T14:32:00Z")
	}
	if ev.SessionID != "LAB-TEST-001" {
		t.Errorf("SessionID = %q, want %q", ev.SessionID, "LAB-TEST-001")
	}
	if ev.Layer != 1 {
		t.Errorf("Layer = %d, want 1", ev.Layer)
	}
	if ev.Event != "connection" {
		t.Errorf("Event = %q, want %q", ev.Event, "connection")
	}
	if ev.Data == nil {
		t.Fatal("Data should not be nil")
	}
	if ev.Data["source_ip"] != "127.0.0.1" {
		t.Errorf("Data[source_ip] = %v, want %q", ev.Data["source_ip"], "127.0.0.1")
	}
}

func TestDeserializeMissingDataField(t *testing.T) {
	raw := `{"timestamp":"2026-02-26T14:32:00Z","session_id":"LAB-TEST-001","layer":1,"event":"disconnect"}`

	var ev ForensicEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if ev.Data != nil {
		t.Errorf("Expected nil Data, got %v", ev.Data)
	}
	if ev.Event != "disconnect" {
		t.Errorf("Event = %q, want %q", ev.Event, "disconnect")
	}
}
