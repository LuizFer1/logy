package report

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"logy/internal/events"
)

type fakeSearcher struct {
	events []events.Event
	err    error
	last   events.EventFilter
}

func (f *fakeSearcher) Search(ctx context.Context, filter events.EventFilter) ([]events.Event, error) {
	f.last = filter
	if f.err != nil {
		return nil, f.err
	}
	var out []events.Event
	for _, ev := range f.events {
		if filter.Matches(ev) {
			out = append(out, ev)
		}
	}
	if out == nil {
		out = []events.Event{}
	}
	return out, nil
}

func TestSummarizeGroupsByProjectAndCountsCommits(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	logy := `C:\dev\Logy`
	other := `C:\dev\other`

	branchPayload, _ := json.Marshal(map[string]any{"name": "feat/ask", "sha": "abc"})
	s := &fakeSearcher{events: []events.Event{
		{
			ID: "c1", StartedAt: now.Add(-2 * time.Hour), EndedAt: now.Add(-2 * time.Hour),
			ProjectPath: logy, Type: "git.commit", Summary: "add ask",
		},
		{
			ID: "c2", StartedAt: now.Add(-1 * time.Hour), EndedAt: now.Add(-1 * time.Hour),
			ProjectPath: logy, Type: "git.commit", Summary: "fix summary",
		},
		{
			ID: "b1", StartedAt: now.Add(-30 * time.Minute), EndedAt: now.Add(-30 * time.Minute),
			ProjectPath: logy, Type: "git.branch", Summary: "feat/ask", Payload: branchPayload,
		},
		{
			ID: "t1", StartedAt: now.Add(-20 * time.Minute), EndedAt: now.Add(-20 * time.Minute),
			ProjectPath: logy, Type: "terminal.command", Summary: "go test",
		},
		{
			ID: "c3", StartedAt: now.Add(-10 * time.Minute), EndedAt: now.Add(-10 * time.Minute),
			ProjectPath: other, Type: "git.commit", Summary: "docs",
		},
	}}

	from, to := PeriodToday(now)
	filter := events.EventFilter{From: from, To: to}
	sum, err := Summarize(context.Background(), s, filter, "today")
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if sum.Period != "today" {
		t.Fatalf("Period = %q, want today", sum.Period)
	}
	if !sum.From.Equal(from) || !sum.To.Equal(to) {
		t.Fatalf("From/To = %v/%v, want %v/%v", sum.From, sum.To, from, to)
	}
	if len(sum.Projects) != 2 {
		t.Fatalf("Projects len = %d, want 2: %+v", len(sum.Projects), sum.Projects)
	}

	byPath := map[string]ProjectLine{}
	for _, p := range sum.Projects {
		byPath[p.Path] = p
	}
	logyLine, ok := byPath[logy]
	if !ok {
		t.Fatalf("missing project %s", logy)
	}
	if logyLine.Commits != 2 {
		t.Fatalf("Logy Commits = %d, want 2", logyLine.Commits)
	}
	if logyLine.Other != 2 { // branch + terminal
		t.Fatalf("Logy Other = %d, want 2", logyLine.Other)
	}
	if logyLine.Branch != "feat/ask" {
		t.Fatalf("Logy Branch = %q, want feat/ask", logyLine.Branch)
	}
	otherLine := byPath[other]
	if otherLine.Commits != 1 || otherLine.Other != 0 {
		t.Fatalf("other line = %+v", otherLine)
	}

	if sum.Text == "" {
		t.Fatal("expected non-empty Text")
	}
	if !strings.Contains(sum.Text, "Logy") && !strings.Contains(sum.Text, logy) {
		t.Fatalf("Text missing project mention: %q", sum.Text)
	}
	if !strings.Contains(sum.Text, "2") {
		t.Fatalf("Text missing commit count: %q", sum.Text)
	}

	ids := map[string]bool{}
	for _, e := range sum.Evidence {
		ids[e.EventID] = true
		if e.StartedAt.IsZero() {
			t.Fatalf("evidence %s missing StartedAt", e.EventID)
		}
		if e.Type == "" {
			t.Fatalf("evidence %s missing Type", e.EventID)
		}
	}
	for _, want := range []string{"c1", "c2", "b1", "c3"} {
		if !ids[want] {
			t.Fatalf("evidence missing %s: %+v", want, sum.Evidence)
		}
	}
}

func TestSummarizeBranchFromLatestEvent(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	path := `C:\dev\app`
	s := &fakeSearcher{events: []events.Event{
		{
			ID: "b-old", StartedAt: now.Add(-2 * time.Hour), EndedAt: now.Add(-2 * time.Hour),
			ProjectPath: path, Type: "git.branch", Summary: "main",
		},
		{
			ID: "b-new", StartedAt: now.Add(-1 * time.Hour), EndedAt: now.Add(-1 * time.Hour),
			ProjectPath: path, Type: "git.branch", Summary: "develop",
		},
		{
			ID: "c1", StartedAt: now.Add(-30 * time.Minute), EndedAt: now.Add(-30 * time.Minute),
			ProjectPath: path, Type: "git.commit", Summary: "wip",
		},
	}}

	sum, err := Summarize(context.Background(), s, events.EventFilter{}, "week")
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if len(sum.Projects) != 1 {
		t.Fatalf("Projects = %+v", sum.Projects)
	}
	if sum.Projects[0].Branch != "develop" {
		t.Fatalf("Branch = %q, want develop", sum.Projects[0].Branch)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	s := &fakeSearcher{events: nil}
	sum, err := Summarize(context.Background(), s, events.EventFilter{}, "today")
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	if len(sum.Projects) != 0 {
		t.Fatalf("Projects = %+v, want empty", sum.Projects)
	}
	if len(sum.Evidence) != 0 {
		t.Fatalf("Evidence = %+v, want empty", sum.Evidence)
	}
	if sum.Text == "" {
		t.Fatal("expected Text describing empty result")
	}
}
