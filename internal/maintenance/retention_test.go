package maintenance

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"logy/internal/events"
	"logy/internal/storage"
)

func openTestDB(t *testing.T) *storage.DB {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "logy.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func sampleEvent(id, project string, started time.Time) events.Event {
	return events.Normalize(events.Event{
		ID:          id,
		StartedAt:   started,
		EndedAt:     started.Add(time.Minute),
		ProjectPath: project,
		Directory:   project,
		Type:        events.EventType("git.commit"),
		Summary:     "did work",
		Payload:     json.RawMessage(`{"files":1}`),
		Source:      "git",
		Sensitivity: events.SensitivityNormal,
	})
}

func TestPurgeEventsBoundaries(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()

	cutoff := time.Date(2026, time.August, 15, 0, 0, 0, 0, time.UTC)
	old := time.Date(2026, time.August, 1, 12, 0, 0, 0, time.UTC)
	newer := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)
	boundary := cutoff

	if err := db.InsertEvents(ctx, []events.Event{
		sampleEvent("old-1", `C:\dev\app`, old),
		sampleEvent("old-2", `C:\dev\other`, old.Add(time.Hour)),
		sampleEvent("boundary", `C:\dev\app`, boundary),
		sampleEvent("new-1", `C:\dev\app`, newer),
	}); err != nil {
		t.Fatalf("InsertEvents() error = %v", err)
	}

	if err := db.AddRoot(ctx, `C:\dev`); err != nil {
		t.Fatalf("AddRoot() error = %v", err)
	}
	if err := db.AddNote(ctx, `C:\dev\app`, "keep this note"); err != nil {
		t.Fatalf("AddNote() error = %v", err)
	}
	if err := db.UpsertProject(ctx, storage.Project{Path: `C:\dev\app`, Name: "app"}); err != nil {
		t.Fatalf("UpsertProject() error = %v", err)
	}
	if err := db.IgnoreProject(ctx, `C:\dev\app`); err != nil {
		t.Fatalf("IgnoreProject() error = %v", err)
	}

	res, err := PurgeEvents(ctx, db, RetentionOptions{OlderThan: cutoff})
	if err != nil {
		t.Fatalf("PurgeEvents() error = %v", err)
	}
	if res.DryRun {
		t.Fatal("DryRun should be false")
	}
	if res.Deleted < 2 {
		t.Fatalf("Deleted = %d, want at least old events", res.Deleted)
	}

	got, err := db.Search(ctx, events.EventFilter{})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	ids := map[string]bool{}
	for _, ev := range got {
		ids[ev.ID] = true
	}
	if ids["old-1"] || ids["old-2"] {
		t.Fatalf("old events still present: %#v", ids)
	}
	if !ids["new-1"] {
		t.Fatalf("new event missing: %#v", ids)
	}

	roots, err := db.ListRoots(ctx)
	if err != nil {
		t.Fatalf("ListRoots() error = %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("roots deleted: %#v", roots)
	}
	notes, err := db.ListNotes(ctx, "", time.Time{}, time.Time{})
	if err != nil {
		t.Fatalf("ListNotes() error = %v", err)
	}
	if len(notes) != 1 || notes[0].Content != "keep this note" {
		t.Fatalf("notes deleted or changed: %#v", notes)
	}
	ignored, err := db.IsIgnored(ctx, `C:\dev\app`)
	if err != nil {
		t.Fatalf("IsIgnored() error = %v", err)
	}
	if !ignored {
		t.Fatal("ignored_projects state was cleared")
	}
}

func TestPurgeEventsProjectFilter(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	cutoff := time.Date(2026, time.August, 15, 0, 0, 0, 0, time.UTC)
	old := time.Date(2026, time.August, 1, 12, 0, 0, 0, time.UTC)

	if err := db.InsertEvents(ctx, []events.Event{
		sampleEvent("app-old", `C:\dev\app`, old),
		sampleEvent("other-old", `C:\dev\other`, old),
	}); err != nil {
		t.Fatalf("InsertEvents() error = %v", err)
	}

	res, err := PurgeEvents(ctx, db, RetentionOptions{
		OlderThan:   cutoff,
		ProjectPath: `C:\dev\app`,
	})
	if err != nil {
		t.Fatalf("PurgeEvents() error = %v", err)
	}
	if res.Deleted != 1 {
		t.Fatalf("Deleted = %d, want 1", res.Deleted)
	}

	got, err := db.Search(ctx, events.EventFilter{})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != "other-old" {
		t.Fatalf("remaining = %#v, want other-old", got)
	}
}

func TestPurgeEventsDryRun(t *testing.T) {
	t.Parallel()
	db := openTestDB(t)
	ctx := context.Background()
	cutoff := time.Date(2026, time.August, 15, 0, 0, 0, 0, time.UTC)
	old := time.Date(2026, time.August, 1, 12, 0, 0, 0, time.UTC)
	newer := time.Date(2026, time.August, 20, 12, 0, 0, 0, time.UTC)

	if err := db.InsertEvents(ctx, []events.Event{
		sampleEvent("old-1", `C:\dev\app`, old),
		sampleEvent("old-2", `C:\dev\app`, old.Add(time.Hour)),
		sampleEvent("new-1", `C:\dev\app`, newer),
	}); err != nil {
		t.Fatalf("InsertEvents() error = %v", err)
	}

	res, err := PurgeEvents(ctx, db, RetentionOptions{
		OlderThan: cutoff,
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("PurgeEvents() error = %v", err)
	}
	if !res.DryRun {
		t.Fatal("DryRun should be true")
	}
	if res.Deleted != 2 {
		t.Fatalf("Deleted (would-be) = %d, want 2", res.Deleted)
	}

	got, err := db.Search(ctx, events.EventFilter{})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("dry-run deleted events: remaining=%d want 3", len(got))
	}
}
