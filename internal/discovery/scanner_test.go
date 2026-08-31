package discovery

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func BenchmarkScan(b *testing.B) {
	root := b.TempDir()
	for i := 0; i < 20; i++ {
		repo := filepath.Join(root, "group", "repo-"+strconv.Itoa(i), "proj")
		if err := os.MkdirAll(filepath.Join(repo, ".git"), 0755); err != nil {
			b.Fatal(err)
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Scan([]string{root}, ScanOptions{MaxDepth: 4}); err != nil {
			b.Fatal(err)
		}
	}
}

func TestScanner(t *testing.T) {
	t.Parallel()

	t.Run("Finds repos at root level", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		repoDir := filepath.Join(root, "repo")
		os.MkdirAll(filepath.Join(repoDir, ".git"), 0755)

		results, err := Scan([]string{root}, ScanOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Name != "repo" {
			t.Errorf("expected name 'repo', got %s", results[0].Name)
		}
	})

	t.Run("Finds repos at nested depths", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		r1 := filepath.Join(root, "r1")
		r2 := filepath.Join(root, "sub", "r2")
		r3 := filepath.Join(root, "sub", "sub2", "r3")
		os.MkdirAll(filepath.Join(r1, ".git"), 0755)
		os.MkdirAll(filepath.Join(r2, ".git"), 0755)
		os.MkdirAll(filepath.Join(r3, ".git"), 0755)

		results, err := Scan([]string{root}, ScanOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
	})

	t.Run("Respects MaxDepth", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		r4 := filepath.Join(root, "1", "2", "3", "r4")
		os.MkdirAll(filepath.Join(r4, ".git"), 0755)

		results, err := Scan([]string{root}, ScanOptions{MaxDepth: 3})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("expected 0 results, got %d", len(results))
		}
	})

	t.Run("Skips ignored paths", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		r1 := filepath.Join(root, "r1")
		r2 := filepath.Join(root, "r2")
		os.MkdirAll(filepath.Join(r1, ".git"), 0755)
		os.MkdirAll(filepath.Join(r2, ".git"), 0755)

		results, err := Scan([]string{root}, ScanOptions{Ignored: []string{r2}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Name != "r1" {
			t.Errorf("expected name 'r1', got %s", results[0].Name)
		}
	})

	t.Run("Handles missing roots", func(t *testing.T) {
		t.Parallel()
		results, err := Scan([]string{filepath.Join(t.TempDir(), "missing")}, ScanOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("expected 0 results, got %d", len(results))
		}
	})

	t.Run("Multiple roots", func(t *testing.T) {
		t.Parallel()
		root1 := t.TempDir()
		root2 := t.TempDir()
		r1 := filepath.Join(root1, "r1")
		r2 := filepath.Join(root2, "r2")
		os.MkdirAll(filepath.Join(r1, ".git"), 0755)
		os.MkdirAll(filepath.Join(r2, ".git"), 0755)

		results, err := Scan([]string{root1, root2}, ScanOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 2 {
			t.Fatalf("expected 2 results, got %d", len(results))
		}
	})

	t.Run("Results are sorted", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		r1 := filepath.Join(root, "c")
		r2 := filepath.Join(root, "a")
		r3 := filepath.Join(root, "b")
		os.MkdirAll(filepath.Join(r1, ".git"), 0755)
		os.MkdirAll(filepath.Join(r2, ".git"), 0755)
		os.MkdirAll(filepath.Join(r3, ".git"), 0755)

		results, err := Scan([]string{root}, ScanOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 3 {
			t.Fatalf("expected 3 results, got %d", len(results))
		}
		if results[0].Name != "a" || results[1].Name != "b" || results[2].Name != "c" {
			t.Errorf("results not sorted alphabetically")
		}
	})

	t.Run("Does not descend into .git", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		repoDir := filepath.Join(root, "repo")
		gitDir := filepath.Join(repoDir, ".git")
		nestedGit := filepath.Join(gitDir, "modules", "nested", ".git")
		os.MkdirAll(nestedGit, 0755)

		results, err := Scan([]string{root}, ScanOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 1 {
			t.Fatalf("expected 1 result, got %d", len(results))
		}
		if results[0].Name != "repo" {
			t.Errorf("expected name 'repo', got %s", results[0].Name)
		}
	})

	t.Run("Empty root", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		results, err := Scan([]string{root}, ScanOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(results) != 0 {
			t.Fatalf("expected 0 results, got %d", len(results))
		}
	})
}
