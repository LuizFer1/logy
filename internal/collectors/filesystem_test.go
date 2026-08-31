package collectors

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFilesystemName(t *testing.T) {
	t.Parallel()
	var f Filesystem
	if f.Name() != "filesystem" {
		t.Fatalf("Name() = %q, want filesystem", f.Name())
	}
}

func TestFilesystemDisabledByDefault(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	evts, err := (Filesystem{}).Collect(ctx, Project{Path: dir, Name: "demo"})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if evts != nil {
		t.Fatalf("Collect() = %#v, want nil when disabled", evts)
	}
}

func TestFilesystemGroupsFileWrites(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.go"), []byte("package a\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.go"), []byte("package b\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "pkg"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pkg", "c.go"), []byte("package pkg\n"), 0644); err != nil {
		t.Fatal(err)
	}

	f := Filesystem{Enabled: true, Debounce: time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	evts, err := f.Collect(ctx, Project{Path: dir, Name: "demo"})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(evts) != 1 {
		t.Fatalf("len(evts) = %d, want 1 grouped event", len(evts))
	}
	evt := evts[0]
	if evt.Type != "filesystem.change" {
		t.Fatalf("Type = %q, want filesystem.change", evt.Type)
	}
	if evt.Source != "filesystem" {
		t.Fatalf("Source = %q, want filesystem", evt.Source)
	}
	if evt.ProjectPath != dir {
		t.Fatalf("ProjectPath = %q, want %q", evt.ProjectPath, dir)
	}

	var payload map[string]any
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		t.Fatalf("payload: %v", err)
	}
	count, ok := payload["changed"].(float64)
	if !ok {
		t.Fatalf("payload missing changed: %#v", payload)
	}
	if int(count) != 3 {
		t.Fatalf("changed = %v, want 3", count)
	}
	sample, ok := payload["sample"].([]any)
	if !ok || len(sample) == 0 {
		t.Fatalf("payload missing sample paths: %#v", payload)
	}
}

func TestFilesystemIgnoresNodeModulesGitAndEnv(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	write := func(rel string, data string) {
		t.Helper()
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("src/main.go", "package main\n")
	write("node_modules/leftpad/index.js", "module.exports=1\n")
	write(".git/HEAD", "ref: refs/heads/main\n")
	write(".env", "SECRET=1\n")
	write(".env.local", "SECRET=2\n")

	f := Filesystem{Enabled: true, Debounce: time.Millisecond}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	evts, err := f.Collect(ctx, Project{Path: dir, Name: "demo"})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(evts) != 1 {
		t.Fatalf("len(evts) = %d, want 1", len(evts))
	}

	var payload map[string]any
	if err := json.Unmarshal(evts[0].Payload, &payload); err != nil {
		t.Fatalf("payload: %v", err)
	}
	count, _ := payload["changed"].(float64)
	if int(count) != 1 {
		t.Fatalf("changed = %v, want 1 (only src/main.go)", count)
	}
	sample, _ := payload["sample"].([]any)
	joined := ""
	for _, s := range sample {
		joined += strings.ToLower(filepath.ToSlash(s.(string))) + ";"
	}
	for _, banned := range []string{"node_modules", "/.git/", ".env"} {
		if strings.Contains(joined, banned) {
			t.Fatalf("sample includes ignored path %q: %#v", banned, sample)
		}
	}
}
