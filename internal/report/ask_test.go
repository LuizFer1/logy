package report

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"logy/internal/events"
)

func TestAskPortugueseYesterdayProject(t *testing.T) {
	loc := time.FixedZone("BRT", -3*3600)
	now := time.Date(2026, 8, 31, 15, 0, 0, 0, loc)
	yesterday := time.Date(2026, 8, 30, 10, 0, 0, 0, loc)
	todayEvt := time.Date(2026, 8, 31, 9, 0, 0, 0, loc)
	logyPath := filepath.Clean(`C:\dev\Logy`)
	otherPath := filepath.Clean(`C:\dev\other`)

	s := &fakeSearcher{events: []events.Event{
		{
			ID: "y1", StartedAt: yesterday, EndedAt: yesterday,
			ProjectPath: logyPath, Type: "git.commit", Summary: "implement ask",
		},
		{
			ID: "y2", StartedAt: yesterday.Add(time.Hour), EndedAt: yesterday.Add(time.Hour),
			ProjectPath: logyPath, Type: "git.branch", Summary: "main",
		},
		{
			ID: "y-other", StartedAt: yesterday, EndedAt: yesterday,
			ProjectPath: otherPath, Type: "git.commit", Summary: "unrelated",
		},
		{
			ID: "t1", StartedAt: todayEvt, EndedAt: todayEvt,
			ProjectPath: logyPath, Type: "git.commit", Summary: "today only",
		},
	}}

	ans, err := Ask(context.Background(), s, "o que fiz no projeto Logy ontem?", now)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !ans.Deterministic {
		t.Fatal("expected Deterministic=true")
	}
	if ans.Text == "" {
		t.Fatal("expected non-empty Text")
	}
	if strings.Contains(strings.ToLower(ans.Text), "unrelated") {
		t.Fatalf("answer leaked other project: %q", ans.Text)
	}
	if strings.Contains(strings.ToLower(ans.Text), "today only") {
		t.Fatalf("answer leaked today event: %q", ans.Text)
	}

	ids := map[string]bool{}
	for _, e := range ans.Evidence {
		ids[e.EventID] = true
		if e.StartedAt.IsZero() {
			t.Fatalf("evidence %s missing timestamp", e.EventID)
		}
	}
	if !ids["y1"] || !ids["y2"] {
		t.Fatalf("expected y1 and y2 in evidence, got %+v", ans.Evidence)
	}
	if ids["y-other"] || ids["t1"] {
		t.Fatalf("unexpected evidence: %+v", ans.Evidence)
	}

	wantFrom, wantTo := PeriodToday(yesterday)
	if !s.last.From.Equal(wantFrom) {
		t.Fatalf("Search From = %v, want %v", s.last.From, wantFrom)
	}
	if !s.last.To.Equal(wantTo) {
		t.Fatalf("Search To = %v, want %v", s.last.To, wantTo)
	}
}

func TestAskHojeSemanaMes(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	s := &fakeSearcher{events: []events.Event{}}

	todayFrom, todayTo := PeriodToday(now)
	weekFrom, weekTo := PeriodWeek(now)
	monthFrom, monthTo := PeriodMonth(now)

	cases := []struct {
		q        string
		wantFrom time.Time
		wantTo   time.Time
	}{
		{"o que fiz hoje?", todayFrom, todayTo},
		{"resumo da semana", weekFrom, weekTo},
		{"atividade do mês", monthFrom, monthTo},
		{"atividade do mes", monthFrom, monthTo},
	}
	for _, tc := range cases {
		_, err := Ask(context.Background(), s, tc.q, now)
		if err != nil {
			t.Fatalf("Ask(%q): %v", tc.q, err)
		}
		if !s.last.From.Equal(tc.wantFrom) || !s.last.To.Equal(tc.wantTo) {
			t.Fatalf("Ask(%q) filter From/To = %v/%v, want %v/%v",
				tc.q, s.last.From, s.last.To, tc.wantFrom, tc.wantTo)
		}
	}
}

func TestAskPathLikeProject(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	path := filepath.Clean(`C:\dev\Logy`)
	s := &fakeSearcher{events: []events.Event{
		{
			ID: "1", StartedAt: now, EndedAt: now,
			ProjectPath: path, Type: "git.commit", Summary: "work",
		},
	}}

	ans, err := Ask(context.Background(), s, `commits em C:\dev\Logy hoje`, now)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if len(ans.Evidence) != 1 || ans.Evidence[0].EventID != "1" {
		t.Fatalf("evidence = %+v", ans.Evidence)
	}
	if s.last.ProjectPath == "" {
		t.Fatal("expected ProjectPath set for path-like token")
	}
}

func TestAskEmptyResults(t *testing.T) {
	now := time.Date(2026, 8, 31, 12, 0, 0, 0, time.UTC)
	s := &fakeSearcher{events: nil}

	ans, err := Ask(context.Background(), s, "o que fiz no projeto Logy ontem?", now)
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if !ans.Deterministic {
		t.Fatal("expected Deterministic=true")
	}
	if len(ans.Evidence) != 0 {
		t.Fatalf("Evidence = %+v", ans.Evidence)
	}
	lower := strings.ToLower(ans.Text)
	if !strings.Contains(lower, "nenhum") && !strings.Contains(lower, "nothing") && !strings.Contains(lower, "no event") {
		t.Fatalf("expected empty-result message, got %q", ans.Text)
	}
}
