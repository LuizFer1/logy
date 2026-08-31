package collectors

import (
	"context"
	"encoding/json"
	"runtime"
	"testing"
	"time"
)

func TestExecRecordsDurationExitAndDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	evt, err := Exec(ctx, Project{Path: dir, Name: "demo"}, "go", []string{"env", "GOVERSION"})
	if err != nil {
		t.Fatalf("Exec() error = %v", err)
	}
	if evt.Type != "terminal.command" {
		t.Fatalf("Type = %q, want terminal.command", evt.Type)
	}
	if evt.Directory != dir {
		t.Fatalf("Directory = %q, want %q", evt.Directory, dir)
	}
	if evt.ProjectPath != dir {
		t.Fatalf("ProjectPath = %q, want %q", evt.ProjectPath, dir)
	}
	if !evt.EndedAt.After(evt.StartedAt) && !evt.EndedAt.Equal(evt.StartedAt) {
		t.Fatalf("timestamps started=%v ended=%v", evt.StartedAt, evt.EndedAt)
	}
	if evt.EndedAt.Sub(evt.StartedAt) < 0 {
		t.Fatal("negative duration")
	}

	var payload map[string]any
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		t.Fatalf("payload: %v", err)
	}
	if payload["command"] != "go" {
		t.Fatalf("command = %#v, want go", payload["command"])
	}
	code, ok := payload["exit_code"].(float64)
	if !ok {
		t.Fatalf("exit_code missing: %#v", payload)
	}
	if code != 0 {
		t.Fatalf("exit_code = %v, want 0", code)
	}
	if _, ok := payload["duration_ms"]; !ok {
		t.Fatalf("duration_ms missing: %#v", payload)
	}
}

func TestExecRecordsNonZeroExit(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	evt, err := Exec(ctx, Project{Path: t.TempDir(), Name: "demo"}, "go", []string{"not-a-real-go-command"})
	if err != nil {
		t.Fatalf("Exec() error = %v, want event with non-zero exit", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		t.Fatalf("payload: %v", err)
	}
	code, _ := payload["exit_code"].(float64)
	if code == 0 {
		t.Fatalf("exit_code = 0, want non-zero: %#v", payload)
	}
}

func TestExecDoesNotWaitOnStdin(t *testing.T) {
	t.Parallel()
	name, args := stdinReaderCommand()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := Exec(ctx, Project{Path: t.TempDir(), Name: "demo"}, name, args)
	if ctx.Err() != nil {
		t.Fatalf("Exec hung waiting for stdin: %v", err)
	}
}

func stdinReaderCommand() (string, []string) {
	if runtime.GOOS == "windows" {
		return "findstr", []string{"nomatch-xyz"}
	}
	return "cat", nil
}
