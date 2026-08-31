package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRootAddWithoutArgsRunsInteractiveLoop(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	root1 := filepath.Join(home, "trabalho")
	root2 := filepath.Join(home, "pessoal")
	if err := os.MkdirAll(root1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(root2, 0755); err != nil {
		t.Fatal(err)
	}

	input := strings.NewReader(root1 + "\n" + root2 + "\n\n")
	var stdout, stderr bytes.Buffer
	code := runWith(cliOptions{Home: home, Stdin: input, Interactive: true}, []string{"root", "add"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("root add interactive exit = %d, stderr = %q", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = runWith(cliOptions{Home: home}, []string{"root", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("root list exit = %d, stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, root1) || !strings.Contains(out, root2) {
		t.Fatalf("root list = %q, want both roots", out)
	}
}

func TestStartPromptsForRootsWhenNoneConfigured(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	root := filepath.Join(home, "trabalho")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	repo := filepath.Join(root, "app")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	input := strings.NewReader(root + "\n\n")
	fgDone := make(chan int, 1)
	opts := cliOptions{
		Home:        home,
		Pipe:        testCLIPipeName(t),
		Stdin:       input,
		Interactive: true,
		StartBackground: func(home, pipe string) error {
			go func() {
				var stdout, stderr bytes.Buffer
				fgDone <- runWith(cliOptions{Home: home, Pipe: pipe}, []string{"start", "--foreground"}, &stdout, &stderr)
			}()
			return nil
		},
	}
	t.Cleanup(func() {
		stopDaemon(t, home, opts.Pipe)
		select {
		case <-fgDone:
		default:
		}
	})

	var stdout, stderr bytes.Buffer
	code := runWith(opts, []string{"start"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("start exit = %d, stderr = %q stdout = %q", code, stderr.String(), stdout.String())
	}
	combined := stdout.String()
	if !strings.Contains(strings.ToLower(combined), "raiz") && !strings.Contains(strings.ToLower(combined), "pasta") {
		t.Fatalf("start output = %q, want onboarding prompt about roots/folders", combined)
	}

	stdout.Reset()
	stderr.Reset()
	code = runWith(cliOptions{Home: home}, []string{"root", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("root list exit = %d, stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), root) {
		t.Fatalf("root list = %q, want %q", stdout.String(), root)
	}
}

func TestStartWithoutRootsNonInteractiveFails(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	var stdout, stderr bytes.Buffer
	code := runWith(cliOptions{Home: home, NonInteractive: true}, []string{"start"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("start without roots in non-interactive mode exit = 0, want non-zero")
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "root add") {
		t.Fatalf("stderr = %q, want hint to root add", stderr.String())
	}
}
