package collectors

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAgentName(t *testing.T) {
	t.Parallel()
	var a Agent
	if a.Name() != "agent" {
		t.Fatalf("Name() = %q, want agent", a.Name())
	}
}

func TestAgentReadSessionsJSONL(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "sessions.jsonl")
	content := "" +
		`{"id":"s1","started_at":"2026-08-31T10:00:00Z","ended_at":"2026-08-31T10:05:00Z","summary":"first session","project_path":"C:\\dev\\demo"}` + "\n" +
		`{"id":"s2","startedAt":"2026-08-31T11:00:00Z","endedAt":"2026-08-31T11:02:00Z","title":"second session"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	evts, err := (Agent{}).ReadSessions(ctx, AgentSource{
		Name:        "fixture",
		PathPattern: filepath.Join(dir, "*.jsonl"),
		Format:      "jsonl",
	})
	if err != nil {
		t.Fatalf("ReadSessions() error = %v", err)
	}
	if len(evts) != 2 {
		t.Fatalf("len(evts) = %d, want 2", len(evts))
	}

	for _, evt := range evts {
		if evt.Type != "agent.session" {
			t.Fatalf("Type = %q, want agent.session", evt.Type)
		}
		if evt.Source != "agent" {
			t.Fatalf("Source = %q, want agent", evt.Source)
		}
		if len(evt.Payload) == 0 {
			t.Fatal("expected non-empty payload")
		}
		var payload map[string]any
		if err := json.Unmarshal(evt.Payload, &payload); err != nil {
			t.Fatalf("payload: %v", err)
		}
		if payload["id"] == nil {
			t.Fatalf("payload missing id: %#v", payload)
		}
	}

	if evts[0].Summary != "first session" {
		t.Fatalf("evts[0].Summary = %q, want first session", evts[0].Summary)
	}
	if evts[0].ProjectPath == "" {
		t.Fatal("evts[0].ProjectPath empty, want project_path from session")
	}
	if evts[1].Summary != "second session" {
		t.Fatalf("evts[1].Summary = %q, want second session", evts[1].Summary)
	}
	wantStart := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	if !evts[0].StartedAt.Equal(wantStart) {
		t.Fatalf("StartedAt = %v, want %v", evts[0].StartedAt, wantStart)
	}
}

func TestAgentReadSessionsSkipsMalformedLines(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "mixed.jsonl")
	content := "" +
		`{"id":"ok1","started_at":"2026-08-31T10:00:00Z","summary":"good"}` + "\n" +
		`{not valid json` + "\n" +
		`{"id":"ok2","title":"also good"}` + "\n" +
		`` + "\n" +
		`42` + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	evts, err := (Agent{}).ReadSessions(ctx, AgentSource{
		Name:        "fixture",
		PathPattern: path,
		Format:      "jsonl",
	})
	if err != nil {
		t.Fatalf("ReadSessions() error = %v", err)
	}
	if len(evts) != 2 {
		t.Fatalf("len(evts) = %d, want 2 (malformed isolated)", len(evts))
	}
	ids := []string{}
	for _, evt := range evts {
		var payload map[string]any
		_ = json.Unmarshal(evt.Payload, &payload)
		if id, _ := payload["id"].(string); id != "" {
			ids = append(ids, id)
		}
	}
	if len(ids) != 2 || ids[0] != "ok1" || ids[1] != "ok2" {
		t.Fatalf("ids = %v, want [ok1 ok2]", ids)
	}
}

func TestAgentReadSessionsMultipleFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.jsonl"), []byte(`{"id":"a1","summary":"from a"}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.jsonl"), []byte(`{"id":"b1","summary":"from b"}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	evts, err := (Agent{}).ReadSessions(ctx, AgentSource{
		Name:        "multi",
		PathPattern: filepath.Join(dir, "*.jsonl"),
		Format:      "jsonl",
	})
	if err != nil {
		t.Fatalf("ReadSessions() error = %v", err)
	}
	if len(evts) != 2 {
		t.Fatalf("len(evts) = %d, want 2", len(evts))
	}
}
