package discovery

import (
	"testing"
)

func TestDiff(t *testing.T) {
	t.Parallel()

	r1 := Result{Path: "/a/b/r1", Name: "r1"}
	r2 := Result{Path: "/a/b/r2", Name: "r2"}
	r3 := Result{Path: "/a/b/r3", Name: "r3"}

	t.Run("No changes", func(t *testing.T) {
		t.Parallel()
		prev := []Result{r1, r2}
		curr := []Result{r1, r2}
		changes := Diff(prev, curr)
		if len(changes.Added) != 0 || len(changes.Removed) != 0 {
			t.Errorf("expected empty changes, got %v", changes)
		}
	})

	t.Run("All new", func(t *testing.T) {
		t.Parallel()
		prev := []Result{}
		curr := []Result{r1, r2}
		changes := Diff(prev, curr)
		if len(changes.Added) != 2 || len(changes.Removed) != 0 {
			t.Errorf("expected 2 added, 0 removed, got %v", changes)
		}
	})

	t.Run("All removed", func(t *testing.T) {
		t.Parallel()
		prev := []Result{r1, r2}
		curr := []Result{}
		changes := Diff(prev, curr)
		if len(changes.Added) != 0 || len(changes.Removed) != 2 {
			t.Errorf("expected 0 added, 2 removed, got %v", changes)
		}
	})

	t.Run("Mixed", func(t *testing.T) {
		t.Parallel()
		prev := []Result{r1, r2}
		curr := []Result{r2, r3}
		changes := Diff(prev, curr)
		if len(changes.Added) != 1 || changes.Added[0].Name != "r3" {
			t.Errorf("expected r3 added, got %v", changes.Added)
		}
		if len(changes.Removed) != 1 || changes.Removed[0].Name != "r1" {
			t.Errorf("expected r1 removed, got %v", changes.Removed)
		}
	})
	
	t.Run("Case insensitive normalization", func(t *testing.T) {
		t.Parallel()
		prev := []Result{{Path: "/A/B/r1", Name: "r1"}}
		curr := []Result{{Path: "/a/b/R1", Name: "R1"}}
		changes := Diff(prev, curr)
		if len(changes.Added) != 0 || len(changes.Removed) != 0 {
			t.Errorf("expected empty changes due to case insensitivity, got %v", changes)
		}
	})
}
