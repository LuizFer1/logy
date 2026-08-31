package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpGeneralListsCommandsAndAliases(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run([]string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("help exit = %d, stderr = %q", code, stderr.String())
	}
	out := stdout.String()
	for _, want := range []string{"start", "str", "stop", "stp", "summarize", "sm", "Usage:"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help missing %q: %q", want, out)
		}
	}
}

func TestHelpCommandStart(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run([]string{"help", "start"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("help start exit = %d, stderr = %q", code, stderr.String())
	}
	out := strings.ToLower(stdout.String())
	if !strings.Contains(out, "start") || !strings.Contains(out, "foreground") {
		t.Fatalf("help start = %q, want start/--foreground details", stdout.String())
	}
	if !strings.Contains(out, "str") {
		t.Fatalf("help start = %q, want alias str", stdout.String())
	}
}

func TestHelpUnknownCommand(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	code := run([]string{"help", "nope"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("help nope exit = 0, want non-zero")
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "unknown") {
		t.Fatalf("stderr = %q, want unknown", stderr.String())
	}
}

func TestAliasStartAndSummarize(t *testing.T) {
	t.Parallel()
	if got := resolveCommand("str"); got != "start" {
		t.Fatalf("resolveCommand(str) = %q, want start", got)
	}
	if got := resolveCommand("stp"); got != "stop" {
		t.Fatalf("resolveCommand(stp) = %q, want stop", got)
	}
	if got := resolveCommand("sm"); got != "summarize" {
		t.Fatalf("resolveCommand(sm) = %q, want summarize", got)
	}
	if got := resolveCommand("summarize"); got != "summarize" {
		t.Fatalf("resolveCommand(summarize) = %q, want summarize", got)
	}
}

func TestAliasStopRunsStopHandler(t *testing.T) {
	t.Parallel()
	var stdout, stderr bytes.Buffer
	// stp with daemon down should fail like stop, not "unknown command"
	code := runWith(cliOptions{Home: t.TempDir(), Pipe: testCLIPipeName(t), NonInteractive: true}, []string{"stp"}, &stdout, &stderr)
	if code == 0 {
		t.Fatal("stp exit = 0, want non-zero when daemon stopped")
	}
	if strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("stp treated as unknown: %q", stderr.String())
	}
	if !strings.Contains(strings.ToLower(stderr.String()), "not running") {
		t.Fatalf("stp stderr = %q, want not running", stderr.String())
	}
}
