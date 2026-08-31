package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"logy/internal/events"
	"logy/internal/storage"
)

func TestNotePersistsAndListsInProjectShow(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	project := filepath.Join(home, "app")

	var stdout, stderr bytes.Buffer
	code := runWith(cliOptions{Home: home}, []string{"note", "--project", project, "Decidi usar SQLite"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("note exit = %d, stderr = %q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runWith(cliOptions{Home: home}, []string{"project", "show", project}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project show exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Decidi usar SQLite") {
		t.Fatalf("project show = %q, want note text", stdout.String())
	}
}

func TestPurgeDryRunDoesNotDelete(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "data"), 0755); err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(filepath.Join(home, "data", "logy.db"))
	if err != nil {
		t.Fatal(err)
	}
	old := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := db.AppendEvents(context.Background(), []events.Event{
		{ID: "old-1", StartedAt: old, EndedAt: old, ProjectPath: `C:\dev\a`, Type: "git.commit", Summary: "ancient", Source: "git"},
	}); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	var stdout, stderr bytes.Buffer
	code := runWith(cliOptions{Home: home}, []string{"purge", "--older-than", "2026-01-01", "--dry-run"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("purge dry-run exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "would delete") {
		t.Fatalf("stdout = %q, want would delete", stdout.String())
	}

	db, err = storage.Open(filepath.Join(home, "data", "logy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	evts, err := db.Search(context.Background(), events.EventFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(evts) != 1 {
		t.Fatalf("events after dry-run = %d, want 1", len(evts))
	}
}

func TestSummarizeAndAskOffline(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "data"), 0755); err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(filepath.Join(home, "data", "logy.db"))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 31, 15, 0, 0, 0, time.UTC)
	if err := db.AppendEvents(context.Background(), []events.Event{
		{ID: "c1", StartedAt: now, EndedAt: now, ProjectPath: `C:\dev\Logy`, Type: "git.commit", Summary: "wire ask", Source: "git"},
	}); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	opts := cliOptions{Home: home, Now: now}
	var stdout, stderr bytes.Buffer
	if code := runWith(opts, []string{"today"}, &stdout, &stderr); code != 0 {
		t.Fatalf("today exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Logy") && !strings.Contains(stdout.String(), "commit") {
		t.Fatalf("today = %q, want project/commit summary", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runWith(opts, []string{"ask", "o que fiz no projeto Logy hoje?"}, &stdout, &stderr); code != 0 {
		t.Fatalf("ask exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "wire ask") {
		t.Fatalf("ask = %q, want commit summary", stdout.String())
	}
}

func TestEventsListsSinceFilter(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "data"), 0755); err != nil {
		t.Fatal(err)
	}
	db, err := storage.Open(filepath.Join(home, "data", "logy.db"))
	if err != nil {
		t.Fatal(err)
	}
	old := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	recent := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	if err := db.AppendEvents(context.Background(), []events.Event{
		{ID: "old", StartedAt: old, EndedAt: old, ProjectPath: `C:\dev\a`, Type: "git.commit", Summary: "old commit", Source: "git"},
		{ID: "new", StartedAt: recent, EndedAt: recent, ProjectPath: `C:\dev\a`, Type: "git.commit", Summary: "new commit", Source: "git"},
	}); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	var stdout, stderr bytes.Buffer
	code := runWith(cliOptions{Home: home}, []string{"events", "--since", "2026-08-01"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("events exit = %d, stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "new commit") {
		t.Fatalf("events = %q, want new commit", out)
	}
	if strings.Contains(out, "old commit") {
		t.Fatalf("events = %q, must not include old commit", out)
	}
}
