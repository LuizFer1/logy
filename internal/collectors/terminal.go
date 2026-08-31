package collectors

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"logy/internal/events"
)

const outputPreviewLimit = 200

// Exec runs command without a shell, discards stdin, and records an event.
func Exec(ctx context.Context, project Project, command string, args []string) (events.Event, error) {
	started := time.Now().UTC()
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = project.Path
	stdin, err := os.Open(os.DevNull)
	if err != nil {
		cmd.Stdin = bytes.NewReader(nil)
	} else {
		cmd.Stdin = stdin
		defer stdin.Close()
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	ended := time.Now().UTC()
	exitCode := 0
	if runErr != nil {
		var ee *exec.ExitError
		if !errors.As(runErr, &ee) {
			return events.Event{}, runErr
		}
		exitCode = ee.ExitCode()
	}

	preview := strings.TrimSpace(stdout.String())
	if preview == "" {
		preview = strings.TrimSpace(stderr.String())
	}
	if len(preview) > outputPreviewLimit {
		preview = preview[:outputPreviewLimit]
	}
	summary := command
	if preview != "" {
		summary = command + ": " + preview
	}

	payload, _ := json.Marshal(map[string]any{
		"command":     command,
		"args":        args,
		"exit_code":   exitCode,
		"duration_ms": ended.Sub(started).Milliseconds(),
	})

	evt := events.Normalize(events.Event{
		ID:          fmt.Sprintf("terminal:%s:%d", command, started.UnixNano()),
		StartedAt:   started,
		EndedAt:     ended,
		ProjectPath: project.Path,
		Directory:   project.Path,
		Type:        "terminal.command",
		Summary:     summary,
		Payload:     payload,
		Source:      "terminal",
	})
	return evt, nil
}
