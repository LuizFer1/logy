package storage

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"logy/internal/events"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(filepath.Join(t.TempDir(), "logy.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return db
}

func sampleEvent(id, project string, started time.Time, eventType events.EventType) events.Event {
	return events.Normalize(events.Event{
		ID:          id,
		StartedAt:   started,
		EndedAt:     started.Add(time.Minute),
		ProjectPath: project,
		Directory:   project,
		Type:        eventType,
		Summary:     "did work",
		Payload:     json.RawMessage(`{"files":1}`),
		Source:      "git",
		Sensitivity: events.SensitivityNormal,
	})
}

func TestOpenCreatesSchema(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()

	tables := []string{
		"roots",
		"projects",
		"ignored_projects",
		"events",
		"notes",
		"summaries",
		"agent_sources",
		"collector_state",
	}
	for _, table := range tables {
		var name string
		err := db.db.QueryRowContext(ctx, "SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Fatalf("missing table %s: %v", table, err)
		}
	}

	var mode string
	if err := db.db.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
}

func TestInsertAndSearchEvents(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	started := time.Date(2026, time.August, 31, 10, 0, 0, 0, time.UTC)
	event := sampleEvent("evt-1", `C:\dev\app`, started, events.EventType("git.commit"))

	if err := db.InsertEvent(ctx, event); err != nil {
		t.Fatalf("InsertEvent() error = %v", err)
	}

	got, err := db.SearchEvents(ctx, events.EventFilter{})
	if err != nil {
		t.Fatalf("SearchEvents() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("SearchEvents() len = %d, want 1", len(got))
	}
	if got[0].ID != event.ID || got[0].Summary != event.Summary || string(got[0].Type) != string(event.Type) {
		t.Fatalf("SearchEvents() = %#v, want %#v", got[0], event)
	}
	if string(got[0].Payload) != string(event.Payload) {
		t.Fatalf("payload = %s, want %s", got[0].Payload, event.Payload)
	}
}

func TestInsertEventsBatchAndFilters(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	t1 := time.Date(2026, time.August, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, time.August, 15, 10, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, time.August, 31, 10, 0, 0, 0, time.UTC)

	evts := []events.Event{
		sampleEvent("a", `C:\dev\app`, t1, events.EventType("git.commit")),
		sampleEvent("b", `C:\dev\app`, t2, events.EventType("git.push")),
		sampleEvent("c", `C:\dev\other`, t3, events.EventType("git.commit")),
	}
	if err := db.InsertEvents(ctx, evts); err != nil {
		t.Fatalf("InsertEvents() error = %v", err)
	}

	t.Run("date filter", func(t *testing.T) {
		got, err := db.SearchEvents(ctx, events.EventFilter{From: t2, To: t2.Add(time.Hour)})
		if err != nil {
			t.Fatalf("SearchEvents() error = %v", err)
		}
		if len(got) != 1 || got[0].ID != "b" {
			t.Fatalf("date filter = %#v, want event b", got)
		}
	})

	t.Run("project filter", func(t *testing.T) {
		got, err := db.SearchEvents(ctx, events.EventFilter{ProjectPath: `C:\dev\other`})
		if err != nil {
			t.Fatalf("SearchEvents() error = %v", err)
		}
		if len(got) != 1 || got[0].ID != "c" {
			t.Fatalf("project filter = %#v, want event c", got)
		}
	})

	t.Run("type filter", func(t *testing.T) {
		got, err := db.SearchEvents(ctx, events.EventFilter{Types: []events.EventType{"git.push"}})
		if err != nil {
			t.Fatalf("SearchEvents() error = %v", err)
		}
		if len(got) != 1 || got[0].ID != "b" {
			t.Fatalf("type filter = %#v, want event b", got)
		}
	})
}

func TestDeleteEventsByFilter(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	started := time.Date(2026, time.August, 31, 10, 0, 0, 0, time.UTC)
	if err := db.InsertEvents(ctx, []events.Event{
		sampleEvent("keep", `C:\dev\app`, started, events.EventType("git.commit")),
		sampleEvent("drop", `C:\dev\other`, started, events.EventType("git.commit")),
	}); err != nil {
		t.Fatalf("InsertEvents() error = %v", err)
	}

	n, err := db.DeleteEvents(ctx, events.EventFilter{ProjectPath: `C:\dev\other`})
	if err != nil {
		t.Fatalf("DeleteEvents() error = %v", err)
	}
	if n != 1 {
		t.Fatalf("DeleteEvents() = %d, want 1", n)
	}

	got, err := db.SearchEvents(ctx, events.EventFilter{})
	if err != nil {
		t.Fatalf("SearchEvents() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != "keep" {
		t.Fatalf("remaining = %#v, want keep", got)
	}
}

func TestRestartPersistence(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "logy.db")
	ctx := context.Background()
	started := time.Date(2026, time.August, 31, 10, 0, 0, 0, time.UTC)

	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := db.InsertEvent(ctx, sampleEvent("persisted", `C:\dev\app`, started, events.EventType("git.commit"))); err != nil {
		db.Close()
		t.Fatalf("InsertEvent() error = %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopen Open() error = %v", err)
	}
	t.Cleanup(func() { reopened.Close() })

	got, err := reopened.SearchEvents(ctx, events.EventFilter{})
	if err != nil {
		t.Fatalf("SearchEvents() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != "persisted" {
		t.Fatalf("after restart = %#v, want persisted event", got)
	}
}

func TestEmptyDatabaseReturnsEmptyResults(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()

	eventsGot, err := db.SearchEvents(ctx, events.EventFilter{})
	if err != nil {
		t.Fatalf("SearchEvents() error = %v", err)
	}
	if len(eventsGot) != 0 {
		t.Fatalf("SearchEvents() = %#v, want empty", eventsGot)
	}

	roots, err := db.ListRoots(ctx)
	if err != nil {
		t.Fatalf("ListRoots() error = %v", err)
	}
	if len(roots) != 0 {
		t.Fatalf("ListRoots() = %#v, want empty", roots)
	}

	projects, err := db.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("ListProjects() = %#v, want empty", projects)
	}

	notes, err := db.ListNotes(ctx, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("ListNotes() error = %v", err)
	}
	if len(notes) != 0 {
		t.Fatalf("ListNotes() = %#v, want empty", notes)
	}
}

func TestRootsCRUD(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()

	if err := db.AddRoot(ctx, `C:\dev`); err != nil {
		t.Fatalf("AddRoot() error = %v", err)
	}
	if err := db.AddRoot(ctx, `C:\dev`); err != nil {
		t.Fatalf("duplicate AddRoot() error = %v", err)
	}

	roots, err := db.ListRoots(ctx)
	if err != nil {
		t.Fatalf("ListRoots() error = %v", err)
	}
	if len(roots) != 1 || roots[0].Path != `C:\dev` {
		t.Fatalf("ListRoots() = %#v", roots)
	}

	if err := db.RemoveRoot(ctx, `C:\dev`); err != nil {
		t.Fatalf("RemoveRoot() error = %v", err)
	}
	roots, err = db.ListRoots(ctx)
	if err != nil {
		t.Fatalf("ListRoots() after remove error = %v", err)
	}
	if len(roots) != 0 {
		t.Fatalf("ListRoots() after remove = %#v", roots)
	}
}

func TestProjectsCRUDAndIgnore(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()

	if err := db.AddRoot(ctx, `C:\dev`); err != nil {
		t.Fatalf("AddRoot() error = %v", err)
	}
	roots, err := db.ListRoots(ctx)
	if err != nil {
		t.Fatalf("ListRoots() error = %v", err)
	}

	proj := Project{
		Path:   `C:\dev\app`,
		Name:   "app",
		RootID: roots[0].ID,
	}
	if err := db.UpsertProject(ctx, proj); err != nil {
		t.Fatalf("UpsertProject() error = %v", err)
	}
	if err := db.UpsertProject(ctx, proj); err != nil {
		t.Fatalf("second UpsertProject() error = %v", err)
	}

	projects, err := db.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects() error = %v", err)
	}
	if len(projects) != 1 || projects[0].Path != proj.Path || projects[0].Name != proj.Name {
		t.Fatalf("ListProjects() = %#v", projects)
	}

	ignored, err := db.IsIgnored(ctx, proj.Path)
	if err != nil {
		t.Fatalf("IsIgnored() error = %v", err)
	}
	if ignored {
		t.Fatal("expected project not ignored")
	}

	if err := db.IgnoreProject(ctx, proj.Path); err != nil {
		t.Fatalf("IgnoreProject() error = %v", err)
	}
	ignored, err = db.IsIgnored(ctx, proj.Path)
	if err != nil {
		t.Fatalf("IsIgnored() after ignore error = %v", err)
	}
	if !ignored {
		t.Fatal("expected project ignored")
	}

	if err := db.UnignoreProject(ctx, proj.Path); err != nil {
		t.Fatalf("UnignoreProject() error = %v", err)
	}
	ignored, err = db.IsIgnored(ctx, proj.Path)
	if err != nil {
		t.Fatalf("IsIgnored() after unignore error = %v", err)
	}
	if ignored {
		t.Fatal("expected project unignored")
	}
}

func TestIgnoreUnknownProjectCreatesRow(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()

	path := `C:\dev\secret`
	if err := db.IgnoreProject(ctx, path); err != nil {
		t.Fatalf("IgnoreProject() error = %v", err)
	}
	ignored, err := db.IsIgnored(ctx, path)
	if err != nil {
		t.Fatalf("IsIgnored() error = %v", err)
	}
	if !ignored {
		t.Fatal("expected unknown project to be ignored after IgnoreProject")
	}
}

func TestNotesCRUD(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()

	if err := db.AddNote(ctx, `C:\dev\app`, "Decidi migrar para PostgreSQL"); err != nil {
		t.Fatalf("AddNote() error = %v", err)
	}
	if err := db.AddNote(ctx, `C:\dev\other`, "outra nota"); err != nil {
		t.Fatalf("AddNote() other error = %v", err)
	}

	notes, err := db.ListNotes(ctx, `C:\dev\app`, time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("ListNotes() error = %v", err)
	}
	if len(notes) != 1 || notes[0].Content != "Decidi migrar para PostgreSQL" {
		t.Fatalf("ListNotes() = %#v", notes)
	}
}

func TestCollectorStateRoundTrip(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	runAt := time.Date(2026, time.August, 31, 12, 0, 0, 0, time.UTC)

	state := CollectorState{
		Collector:   "git",
		ProjectPath: `C:\dev\app`,
		Cursor:      "abc123",
		LastRunAt:   runAt,
	}
	if err := db.SaveCollectorState(ctx, state); err != nil {
		t.Fatalf("SaveCollectorState() error = %v", err)
	}
	state.Cursor = "def456"
	if err := db.SaveCollectorState(ctx, state); err != nil {
		t.Fatalf("SaveCollectorState() update error = %v", err)
	}

	got, err := db.LoadCollectorState(ctx, "git", `C:\dev\app`)
	if err != nil {
		t.Fatalf("LoadCollectorState() error = %v", err)
	}
	if got.Cursor != "def456" {
		t.Fatalf("cursor = %q, want def456", got.Cursor)
	}
}

func TestDuplicateEventIDUpdatesRow(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	started := time.Date(2026, time.August, 31, 10, 0, 0, 0, time.UTC)
	event := sampleEvent("same", `C:\dev\app`, started, events.EventType("git.commit"))
	if err := db.InsertEvent(ctx, event); err != nil {
		t.Fatalf("InsertEvent() error = %v", err)
	}
	event.Summary = "updated"
	if err := db.InsertEvent(ctx, event); err != nil {
		t.Fatalf("InsertEvent() update error = %v", err)
	}

	got, err := db.SearchEvents(ctx, events.EventFilter{})
	if err != nil {
		t.Fatalf("SearchEvents() error = %v", err)
	}
	if len(got) != 1 || got[0].Summary != "updated" {
		t.Fatalf("duplicate insert = %#v", got)
	}
}
