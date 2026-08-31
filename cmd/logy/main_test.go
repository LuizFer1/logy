package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestRunShowsHelpForHelpCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "help flag", args: []string{"--help"}},
		{name: "help command", args: []string{"help"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := run(tt.args, &stdout, &stderr)
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0", exitCode)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}

			output := stdout.String()
			for _, command := range commandNames {
				if !strings.Contains(output, command) {
					t.Fatalf("help output missing command %q: %q", command, output)
				}
			}
			if !strings.Contains(output, "Usage:") {
				t.Fatalf("help output missing usage text: %q", output)
			}
		})
	}
}

func TestRunRecognizesCommands(t *testing.T) {
	t.Parallel()

	skip := map[string]bool{
		"start":     true,
		"status":    true,
		"stop":      true,
		"root":      true,
		"project":   true,
		"doctor":    true,
		"exec":      true,
		"note":      true,
		"ask":       true,
		"today":     true,
		"week":      true,
		"month":     true,
		"events":    true,
		"summarize": true,
		"purge":     true,
		"startup":   true,
		"version":   true,
	}

	for _, command := range commandNames {
		if skip[command] {
			continue
		}
		command := command
		t.Run(command, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := run([]string{command}, &stdout, &stderr)
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0", exitCode)
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestRunRejectsMissingAndUnknownCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		args          []string
		wantSubstring string
	}{
		{
			name:          "missing command",
			args:          nil,
			wantSubstring: "missing command",
		},
		{
			name:          "unknown command",
			args:          []string{"launch"},
			wantSubstring: "unknown command: launch",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var stdout bytes.Buffer
			var stderr bytes.Buffer

			exitCode := run(tt.args, &stdout, &stderr)
			if exitCode == 0 {
				t.Fatalf("exit code = 0, want non-zero")
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout = %q, want empty", stdout.String())
			}

			output := stderr.String()
			if !strings.Contains(output, tt.wantSubstring) {
				t.Fatalf("stderr = %q, want substring %q", output, tt.wantSubstring)
			}
			if !strings.Contains(output, "Usage:") {
				t.Fatalf("stderr missing usage text: %q", output)
			}
		})
	}
}
