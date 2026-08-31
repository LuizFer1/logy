package ai

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"logy/internal/events"
)

func TestBuildContextSanitizesSecretsAndRespectsExclusions(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, time.August, 31, 10, 0, 0, 0, time.UTC)
	evts := []events.Event{
		{
			ID:          "keep-1",
			StartedAt:   started,
			ProjectPath: `C:\dev\app`,
			Directory:   `C:\dev\app`,
			Type:        events.EventType("git.commit"),
			Summary:     "token=supersecret password=hunter2 api_key=xyz keep=ok",
			Payload:     json.RawMessage(`{"token":"supersecret","keep":"ok"}`),
			Source:      "git",
		},
		{
			ID:          "excluded-1",
			StartedAt:   started.Add(time.Minute),
			ProjectPath: `C:\dev\vendor\pkg`,
			Directory:   `C:\dev\vendor\pkg`,
			Type:        events.EventType("git.commit"),
			Summary:     "should not appear",
			Source:      "git",
		},
	}

	prompt := BuildContext(evts, ContextRules{
		ExcludeGlobs: []string{`C:/dev/vendor/*`},
		MaxEvents:    40,
		MaxChars:     8000,
	})

	if prompt.System == "" {
		t.Fatal("System prompt is empty")
	}
	lowerSys := strings.ToLower(prompt.System)
	if !strings.Contains(lowerSys, "evidence") {
		t.Fatalf("System prompt should mention evidence: %q", prompt.System)
	}
	if !strings.Contains(lowerSys, "command") {
		t.Fatalf("System prompt should discourage commands: %q", prompt.System)
	}

	if strings.Contains(prompt.User, "supersecret") || strings.Contains(prompt.User, "hunter2") {
		t.Fatalf("User prompt leaked secrets: %q", prompt.User)
	}
	if !strings.Contains(prompt.User, "[REDACTED]") {
		t.Fatalf("User prompt missing redaction marker: %q", prompt.User)
	}
	if strings.Contains(prompt.User, "excluded-1") || strings.Contains(prompt.User, "should not appear") {
		t.Fatalf("User prompt includes excluded event: %q", prompt.User)
	}
	if !strings.Contains(prompt.User, "keep-1") {
		t.Fatalf("User prompt missing kept event id: %q", prompt.User)
	}
}

func TestBuildContextBoundsSize(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, time.August, 31, 10, 0, 0, 0, time.UTC)
	var evts []events.Event
	for i := 0; i < 20; i++ {
		evts = append(evts, events.Event{
			ID:          "evt-" + strings.Repeat("x", 8) + string(rune('a'+i%26)) + string(rune('0'+i%10)),
			StartedAt:   started.Add(time.Duration(i) * time.Minute),
			ProjectPath: `C:\dev\app`,
			Directory:   `C:\dev\app`,
			Type:        events.EventType("git.commit"),
			Summary:     strings.Repeat("word ", 40),
			Source:      "git",
		})
	}
	// Stable unique IDs
	for i := range evts {
		evts[i].ID = "evt-" + string(rune('a'+i))
	}

	prompt := BuildContext(evts, ContextRules{
		MaxEvents: 5,
		MaxChars:  8000,
	})
	count := strings.Count(prompt.User, "evt-")
	if count > 5 {
		t.Fatalf("expected at most 5 events in user prompt, found ~%d: %q", count, prompt.User)
	}

	short := BuildContext(evts, ContextRules{
		MaxEvents: 40,
		MaxChars:  200,
	})
	if len(short.User) > 200+100 { // allow small overhead for formatting, but must be bounded
		// MaxChars applies to evidence body; ensure we clearly stayed near the limit
		if len(short.User) > 2000 {
			t.Fatalf("MaxChars not respected: user len=%d", len(short.User))
		}
	}
	// Strict: evidence portion should not wildly exceed MaxChars
	if len(short.User) > 500 {
		t.Fatalf("expected MaxChars=200 to keep user prompt small, got len=%d", len(short.User))
	}
}

type fakeProvider struct {
	calls int
	reply string
	err   error
}

func (f *fakeProvider) Generate(ctx context.Context, prompt Prompt) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	return f.reply, nil
}

func TestAnswerWithOptionalAI(t *testing.T) {
	t.Parallel()

	deterministic := "offline answer"
	prompt := Prompt{System: "sys", User: "user"}

	t.Run("skips when useAI false", func(t *testing.T) {
		t.Parallel()
		fp := &fakeProvider{reply: "ai answer"}
		got := AnswerWithOptionalAI(deterministic, false, fp, prompt)
		if got != deterministic {
			t.Fatalf("got %q, want deterministic", got)
		}
		if fp.calls != 0 {
			t.Fatalf("Generate called %d times, want 0", fp.calls)
		}
	})

	t.Run("skips when provider nil", func(t *testing.T) {
		t.Parallel()
		got := AnswerWithOptionalAI(deterministic, true, nil, prompt)
		if got != deterministic {
			t.Fatalf("got %q, want deterministic", got)
		}
	})

	t.Run("calls provider when wired", func(t *testing.T) {
		t.Parallel()
		fp := &fakeProvider{reply: "ai answer"}
		got := AnswerWithOptionalAI(deterministic, true, fp, prompt)
		if got != "ai answer" {
			t.Fatalf("got %q, want ai answer", got)
		}
		if fp.calls != 1 {
			t.Fatalf("Generate called %d times, want 1", fp.calls)
		}
	})

	t.Run("falls back on provider failure", func(t *testing.T) {
		t.Parallel()
		fp := &fakeProvider{err: context.DeadlineExceeded}
		got := AnswerWithOptionalAI(deterministic, true, fp, prompt)
		if !strings.Contains(got, deterministic) {
			t.Fatalf("fallback missing deterministic answer: %q", got)
		}
		if got == deterministic {
			// must include a note about AI failure
			t.Fatalf("expected note about AI failure alongside deterministic answer, got exact deterministic only")
		}
		if fp.calls != 1 {
			t.Fatalf("Generate called %d times, want 1", fp.calls)
		}
	})
}
