package collectors

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"logy/internal/events"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
}

func initGitRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "logy@example.com")
	runGit(t, dir, "config", "user.name", "Logy Test")
	runGit(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "initial commit")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func payloadOf(t *testing.T, evts []events.Event, typ events.EventType) map[string]any {
	t.Helper()
	for _, evt := range evts {
		if evt.Type == typ {
			var payload map[string]any
			if len(evt.Payload) > 0 {
				if err := json.Unmarshal(evt.Payload, &payload); err != nil {
					t.Fatalf("payload json: %v (%s)", err, evt.Payload)
				}
			}
			return payload
		}
	}
	t.Fatalf("missing event type %q", typ)
	return nil
}

func TestGitName(t *testing.T) {
	t.Parallel()
	var c Git
	if c.Name() != "git" {
		t.Fatalf("Name() = %q, want git", c.Name())
	}
}

func TestGitCollectsHeadCommitBranchAndStatus(t *testing.T) {
	t.Parallel()
	repo := initGitRepo(t)
	c := Git{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	evts, err := c.Collect(ctx, Project{Path: repo, Name: "demo"})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if len(evts) == 0 {
		t.Fatal("Collect() returned no events")
	}

	types := map[events.EventType]bool{}
	var commitSummary string
	for _, evt := range evts {
		types[evt.Type] = true
		if evt.ProjectPath != repo {
			t.Fatalf("ProjectPath = %q, want %q", evt.ProjectPath, repo)
		}
		if evt.Source != "git" {
			t.Fatalf("Source = %q, want git", evt.Source)
		}
		if evt.Type == "git.commit" {
			commitSummary = evt.Summary
		}
	}
	for _, want := range []events.EventType{"git.commit", "git.branch", "git.status"} {
		if !types[want] {
			t.Fatalf("missing %s event, types=%v", want, types)
		}
	}
	if !strings.Contains(commitSummary, "initial commit") {
		t.Fatalf("commit summary = %q, want initial commit", commitSummary)
	}

	commit := payloadOf(t, evts, "git.commit")
	if sha, _ := commit["sha"].(string); sha == "" {
		t.Fatalf("git.commit payload missing sha: %#v", commit)
	}
	branch := payloadOf(t, evts, "git.branch")
	name, _ := branch["name"].(string)
	if name != "master" && name != "main" {
		t.Fatalf("branch name = %q, want master or main", name)
	}
	status := payloadOf(t, evts, "git.status")
	if dirty, _ := status["dirty"].(bool); dirty {
		t.Fatalf("clean repo reported dirty: %#v", status)
	}
}

func TestGitCollectsDirtyStatusAndDiffStat(t *testing.T) {
	t.Parallel()
	repo := initGitRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hello\nworld\n"), 0644); err != nil {
		t.Fatal(err)
	}

	c := Git{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	evts, err := c.Collect(ctx, Project{Path: repo, Name: "demo"})
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	status := payloadOf(t, evts, "git.status")
	if dirty, _ := status["dirty"].(bool); !dirty {
		t.Fatalf("expected dirty status, got %#v", status)
	}

	stat := payloadOf(t, evts, "git.diffstat")
	if _, ok := stat["insertions"]; !ok {
		t.Fatalf("git.diffstat missing insertions: %#v", stat)
	}
	if _, ok := stat["deletions"]; !ok {
		t.Fatalf("git.diffstat missing deletions: %#v", stat)
	}
}
