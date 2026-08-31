package collectors

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"logy/internal/events"
)

const gitTimeout = 5 * time.Second

// Git collects HEAD, branch, status, and diff statistics.
type Git struct {
	Timeout time.Duration
}

func (Git) Name() string { return "git" }

func (g Git) Collect(ctx context.Context, project Project) ([]events.Event, error) {
	timeout := g.Timeout
	if timeout <= 0 {
		timeout = gitTimeout
	}
	now := time.Now().UTC()

	sha, err := g.run(ctx, timeout, project.Path, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	sha = strings.TrimSpace(sha)

	message, err := g.run(ctx, timeout, project.Path, "log", "-1", "--format=%s")
	if err != nil {
		return nil, err
	}
	message = strings.TrimSpace(message)

	branch, err := g.run(ctx, timeout, project.Path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return nil, err
	}
	branch = strings.TrimSpace(branch)

	porcelain, err := g.run(ctx, timeout, project.Path, "status", "--porcelain")
	if err != nil {
		return nil, err
	}
	dirty := strings.TrimSpace(porcelain) != ""
	changed := 0
	if dirty {
		changed = len(splitLines(porcelain))
	}

	numstat, err := g.run(ctx, timeout, project.Path, "diff", "--numstat")
	if err != nil {
		return nil, err
	}
	insertions, deletions := parseNumstat(numstat)

	base := events.Event{
		StartedAt:   now,
		EndedAt:     now,
		ProjectPath: project.Path,
		Directory:   project.Path,
		Source:      "git",
	}

	commitPayload, _ := json.Marshal(map[string]any{"sha": sha, "message": message})
	branchPayload, _ := json.Marshal(map[string]any{"name": branch, "sha": sha})
	statusPayload, _ := json.Marshal(map[string]any{"dirty": dirty, "changed": changed})
	statPayload, _ := json.Marshal(map[string]any{"insertions": insertions, "deletions": deletions})

	out := []events.Event{
		events.Normalize(events.Event{
			ID: "git.commit:" + project.Path + ":" + sha, Type: "git.commit", Summary: message, Payload: commitPayload,
			StartedAt: base.StartedAt, EndedAt: base.EndedAt, ProjectPath: base.ProjectPath, Directory: base.Directory, Source: base.Source,
		}),
		events.Normalize(events.Event{
			ID: "git.branch:" + project.Path + ":" + branch, Type: "git.branch", Summary: branch, Payload: branchPayload,
			StartedAt: base.StartedAt, EndedAt: base.EndedAt, ProjectPath: base.ProjectPath, Directory: base.Directory, Source: base.Source,
		}),
		events.Normalize(events.Event{
			ID: "git.status:" + project.Path, Type: "git.status", Summary: statusSummary(dirty, changed), Payload: statusPayload,
			StartedAt: base.StartedAt, EndedAt: base.EndedAt, ProjectPath: base.ProjectPath, Directory: base.Directory, Source: base.Source,
		}),
		events.Normalize(events.Event{
			ID: "git.diffstat:" + project.Path, Type: "git.diffstat", Summary: "diffstat", Payload: statPayload,
			StartedAt: base.StartedAt, EndedAt: base.EndedAt, ProjectPath: base.ProjectPath, Directory: base.Directory, Source: base.Source,
		}),
	}
	return out, nil
}

func (Git) run(ctx context.Context, timeout time.Duration, dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Stdin = nil
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return stdout.String(), nil
}

func splitLines(value string) []string {
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func parseNumstat(value string) (insertions, deletions int) {
	for _, line := range splitLines(value) {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		ins, _ := strconv.Atoi(fields[0])
		del, _ := strconv.Atoi(fields[1])
		insertions += ins
		deletions += del
	}
	return insertions, deletions
}

func statusSummary(dirty bool, changed int) string {
	if !dirty {
		return "clean"
	}
	return strconv.Itoa(changed) + " changed"
}
