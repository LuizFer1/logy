package ai

import (
	"fmt"
	"strings"
	"time"

	"logy/internal/events"
)

// ContextRules control filtering and size limits for AI prompts.
type ContextRules struct {
	ExcludeGlobs []string
	MaxEvents    int // default 40
	MaxChars     int // default 8000
}

const (
	defaultMaxEvents = 40
	defaultMaxChars  = 8000
)

const systemPrompt = "Answer only from the provided evidence. Do not invent facts. Do not suggest or execute commands. Do not run shell or file operations."

// BuildContext filters, redacts, and bounds events into a provider Prompt.
func BuildContext(evts []events.Event, rules ContextRules) Prompt {
	maxEvents := rules.MaxEvents
	if maxEvents <= 0 {
		maxEvents = defaultMaxEvents
	}
	maxChars := rules.MaxChars
	if maxChars <= 0 {
		maxChars = defaultMaxChars
	}

	filter := events.EventFilter{ExcludeGlobs: rules.ExcludeGlobs}
	redactRules := events.RedactionRules{}

	var prepared []events.Event
	for _, evt := range evts {
		if !filter.Matches(evt) {
			continue
		}
		prepared = append(prepared, events.Redact(evt, redactRules))
	}

	if len(prepared) > maxEvents {
		prepared = prepared[len(prepared)-maxEvents:]
	}

	var b strings.Builder
	b.WriteString("Evidence:\n")
	for i, evt := range prepared {
		line := formatEvidenceLine(i+1, evt)
		needed := len(line) + 1
		if b.Len()+needed > maxChars {
			if i == 0 {
				remain := maxChars - b.Len() - 1
				if remain < 16 {
					remain = 16
				}
				if len(line) > remain {
					line = line[:remain] + "…"
				}
				b.WriteString(line)
			}
			break
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}

	return Prompt{
		System: systemPrompt,
		User:   strings.TrimRight(b.String(), "\n"),
	}
}

func formatEvidenceLine(n int, evt events.Event) string {
	ts := evt.StartedAt.UTC().Format(time.RFC3339)
	summary := strings.TrimSpace(evt.Summary)
	return fmt.Sprintf("%d. [id=%s at=%s] %s", n, evt.ID, ts, summary)
}
