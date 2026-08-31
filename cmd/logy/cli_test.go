package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"logy/internal/control"
	"logy/internal/events"
	"logy/internal/storage"
)

func testCLIPipeName(t *testing.T) string {
	t.Helper()
	safe := strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ' ', ':':
			return '-'
		default:
			return r
		}
	}, t.Name())
	if runtime.GOOS == "windows" {
		return fmt.Sprintf(`\\.\pipe\logy-cli-%d-%s-%d`, os.Getpid(), safe, time.Now().UnixNano())
	}
	return filepath.Join(t.TempDir(), "logy.sock")
}

func waitRunning(t *testing.T, home, pipe string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		var stdout, stderr bytes.Buffer
		if runWith(cliOptions{Home: home, Pipe: pipe}, []string{"status"}, &stdout, &stderr) == 0 &&
			strings.Contains(stdout.String(), "running: true") {
			return
		}
		time.Sleep(30 * time.Millisecond)
	}
	t.Fatal("status never reported running: true after start")
}

func stopDaemon(t *testing.T, home, pipe string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	_ = runWith(cliOptions{Home: home, Pipe: pipe}, []string{"stop"}, &stdout, &stderr)
}

func TestRootAddAndList(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	root := filepath.Join(home, "workspace")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runWith(cliOptions{Home: home}, []string{"root", "add", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("root add exit = %d, stderr = %q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runWith(cliOptions{Home: home}, []string{"root", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("root list exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), root) {
		t.Fatalf("root list = %q, want path %q", stdout.String(), root)
	}
}

func TestRootRequiresSubcommand(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := runWith(cliOptions{Home: t.TempDir()}, []string{"root"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("root without subcommand exit = 0, want non-zero")
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "root") {
		t.Fatalf("stderr = %q, want usage mentioning root", stderr.String())
	}
}

func TestProjectIgnoreListUnignore(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	project := filepath.Join(home, "app")
	if err := os.MkdirAll(project, 0755); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runWith(cliOptions{Home: home}, []string{"project", "ignore", project}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project ignore exit = %d, stderr = %q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runWith(cliOptions{Home: home}, []string{"project", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project list exit = %d, stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, project) {
		t.Fatalf("project list = %q, want path %q", out, project)
	}
	if !strings.Contains(strings.ToLower(out), "ignored") {
		t.Fatalf("project list = %q, want ignored marker", out)
	}

	stdout.Reset()
	stderr.Reset()
	code = runWith(cliOptions{Home: home}, []string{"project", "unignore", project}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project unignore exit = %d, stderr = %q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runWith(cliOptions{Home: home}, []string{"project", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("project list after unignore exit = %d, stderr = %q", code, stderr.String())
	}
	out = stdout.String()
	if !strings.Contains(out, project) {
		t.Fatalf("project list = %q, want path %q", out, project)
	}
	if strings.Contains(strings.ToLower(out), "ignored") {
		t.Fatalf("project list = %q, want no ignored marker", out)
	}
}

func TestDoctorReportsLocalChecks(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	root := filepath.Join(home, "workspace")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := runWith(cliOptions{Home: home}, []string{"root", "add", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("root add exit = %d, stderr = %q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code := runWith(cliOptions{Home: home, Pipe: testCLIPipeName(t)}, []string{"doctor"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("doctor exit = %d, stderr = %q", code, stderr.String())
	}
	out := strings.ToLower(stdout.String())
	for _, want := range []string{"database", "root", "daemon"} {
		if !strings.Contains(out, want) {
			t.Fatalf("doctor output = %q, want %q", stdout.String(), want)
		}
	}
	if !strings.Contains(out, "not running") && !strings.Contains(out, "unavailable") {
		t.Fatalf("doctor output = %q, want daemon not running", stdout.String())
	}
}

func TestStatusWhenDaemonStopped(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := runWith(cliOptions{Home: t.TempDir(), Pipe: testCLIPipeName(t)}, []string{"status"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("status exit = 0, want non-zero when daemon is stopped")
	}
	if !strings.Contains(strings.ToLower(stderr.String()+stdout.String()), "not running") {
		t.Fatalf("status output stdout=%q stderr=%q, want not running", stdout.String(), stderr.String())
	}
}

func TestStatusWhenDaemonRunning(t *testing.T) {
	t.Parallel()
	name := testCLIPipeName(t)
	ln, err := control.ListenName(name)
	if err != nil {
		t.Fatalf("ListenName() error = %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() {
		_ = control.Serve(ctx, ln, control.Handler{
			Status: func() control.StatusPayload {
				return control.StatusPayload{Running: true, Collectors: []string{"git"}}
			},
		})
	}()

	var stdout, stderr bytes.Buffer
	code := runWith(cliOptions{Home: t.TempDir(), Pipe: name}, []string{"status"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("status exit = %d, stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "running: true") {
		t.Fatalf("status stdout = %q, want running: true", out)
	}
	if !strings.Contains(out, "git") {
		t.Fatalf("status stdout = %q, want collector git", out)
	}
}

func TestStartServesStatusThenStop(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	pipe := testCLIPipeName(t)

	done := make(chan int, 1)
	go func() {
		var stdout, stderr bytes.Buffer
		done <- runWith(cliOptions{Home: home, Pipe: pipe}, []string{"start"}, &stdout, &stderr)
	}()
	t.Cleanup(func() { stopDaemon(t, home, pipe) })
	waitRunning(t, home, pipe)

	var stopOut, stopErr bytes.Buffer
	if code := runWith(cliOptions{Home: home, Pipe: pipe}, []string{"stop"}, &stopOut, &stopErr); code != 0 {
		t.Fatalf("stop exit = %d, stderr = %q", code, stopErr.String())
	}

	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("start exit = %d, want 0 after stop", code)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("start did not return after stop")
	}
}

func TestStartRejectsDuplicate(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	pipe := testCLIPipeName(t)

	done := make(chan int, 1)
	go func() {
		var stdout, stderr bytes.Buffer
		done <- runWith(cliOptions{Home: home, Pipe: pipe}, []string{"start"}, &stdout, &stderr)
	}()
	t.Cleanup(func() {
		stopDaemon(t, home, pipe)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	})
	waitRunning(t, home, pipe)

	var stdout, stderr bytes.Buffer
	code := runWith(cliOptions{Home: home, Pipe: pipe}, []string{"start"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("second start exit = 0, want non-zero")
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "already running") {
		t.Fatalf("second start stderr = %q, want already running", stderr.String())
	}
}

func TestStartCollectsGitFromRoot(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	home := t.TempDir()
	root := filepath.Join(home, "workspace")
	repo := filepath.Join(root, "app")
	if err := os.MkdirAll(repo, 0755); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "logy@example.com")
	runGit(t, repo, "config", "user.name", "Logy Test")
	runGit(t, repo, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("hello\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "initial commit")

	var stdout, stderr bytes.Buffer
	if code := runWith(cliOptions{Home: home}, []string{"root", "add", root}, &stdout, &stderr); code != 0 {
		t.Fatalf("root add exit = %d, stderr = %q", code, stderr.String())
	}

	pipe := testCLIPipeName(t)
	done := make(chan int, 1)
	go func() {
		var out, errBuf bytes.Buffer
		done <- runWith(cliOptions{Home: home, Pipe: pipe}, []string{"start"}, &out, &errBuf)
	}()
	t.Cleanup(func() {
		stopDaemon(t, home, pipe)
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	})
	waitRunning(t, home, pipe)

	dbPath := filepath.Join(home, "data", "logy.db")
	deadline := time.Now().Add(8 * time.Second)
	var found int
	for time.Now().Before(deadline) {
		db, err := storage.Open(dbPath)
		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		evts, err := db.Search(context.Background(), events.EventFilter{Types: []events.EventType{"git.commit"}})
		_ = db.Close()
		if err == nil && len(evts) > 0 {
			found = len(evts)
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if found == 0 {
		t.Fatal("expected git.commit events after start")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func TestExecCLIPersistsTerminalEvent(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runWith(cliOptions{Home: home}, []string{"exec", "--", "go", "env", "GOVERSION"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exec exit = %d, stderr = %q", code, stderr.String())
	}

	db, err := storage.Open(filepath.Join(home, "data", "logy.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()
	evts, err := db.Search(context.Background(), events.EventFilter{Types: []events.EventType{"terminal.command"}})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(evts) != 1 {
		t.Fatalf("got %d terminal events, want 1", len(evts))
	}
	if evts[0].Source != "terminal" {
		t.Fatalf("source = %q, want terminal", evts[0].Source)
	}
}

func TestVersionPrintsDevByDefault(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run([]string{"version"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("version exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "dev") {
		t.Fatalf("version stdout = %q, want dev", stdout.String())
	}
}

func TestHelpListsRootProjectDoctor(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("help exit = %d", code)
	}
	out := stdout.String()
	for _, name := range []string{"root", "project", "doctor"} {
		if !strings.Contains(out, name) {
			t.Fatalf("help missing %q: %q", name, out)
		}
	}
}
