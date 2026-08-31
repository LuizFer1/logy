package report

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"logy/internal/events"
)

// Searcher is the read surface Summarize and Ask need from storage.
type Searcher interface {
	Search(ctx context.Context, filter events.EventFilter) ([]events.Event, error)
}

type Summary struct {
	Period   string
	From, To time.Time
	Text     string
	Projects []ProjectLine
	Evidence []Evidence
}

type ProjectLine struct {
	Path    string
	Commits int
	Other   int
	Branch  string
}

type Evidence struct {
	EventID   string
	StartedAt time.Time
	Summary   string
	Type      events.EventType
}

// Summarize aggregates matching events into a deterministic project summary.
func Summarize(ctx context.Context, s Searcher, filter events.EventFilter, periodName string) (Summary, error) {
	evts, err := s.Search(ctx, filter)
	if err != nil {
		return Summary{}, err
	}

	type agg struct {
		commits  int
		other    int
		branch   string
		branchAt time.Time
	}
	byPath := map[string]*agg{}
	var evidence []Evidence

	for _, ev := range evts {
		ev = events.Normalize(ev)
		path := ev.ProjectPath
		if path == "" {
			path = "(unknown)"
		}
		a, ok := byPath[path]
		if !ok {
			a = &agg{}
			byPath[path] = a
		}

		switch ev.Type {
		case "git.commit":
			a.commits++
			evidence = append(evidence, toEvidence(ev))
		case "git.branch":
			a.other++
			branch := branchName(ev)
			if branch != "" && (a.branch == "" || !ev.StartedAt.Before(a.branchAt)) {
				a.branch = branch
				a.branchAt = ev.StartedAt
			}
			evidence = append(evidence, toEvidence(ev))
		default:
			a.other++
		}
	}

	paths := make([]string, 0, len(byPath))
	for p := range byPath {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	projects := make([]ProjectLine, 0, len(paths))
	var lines []string
	for _, p := range paths {
		a := byPath[p]
		line := ProjectLine{Path: p, Commits: a.commits, Other: a.other, Branch: a.branch}
		projects = append(projects, line)
		lines = append(lines, formatProjectLine(line))
	}

	text := strings.Join(lines, "\n")
	if text == "" {
		text = "Nenhum evento encontrado."
	}

	return Summary{
		Period:   periodName,
		From:     filter.From,
		To:       filter.To,
		Text:     text,
		Projects: projects,
		Evidence: evidence,
	}, nil
}

func toEvidence(ev events.Event) Evidence {
	return Evidence{
		EventID:   ev.ID,
		StartedAt: ev.StartedAt,
		Summary:   ev.Summary,
		Type:      ev.Type,
	}
}

func branchName(ev events.Event) string {
	if name := strings.TrimSpace(ev.Summary); name != "" {
		return name
	}
	if len(ev.Payload) == 0 {
		return ""
	}
	var payload struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Name)
}

func formatProjectLine(line ProjectLine) string {
	base := fmt.Sprintf("%s: %d commits, %d outros", line.Path, line.Commits, line.Other)
	if line.Branch != "" {
		base += fmt.Sprintf(" (branch: %s)", line.Branch)
	}
	return base
}
