package events

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestRedact(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, time.August, 31, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		event Event
		rules RedactionRules
		want  Event
	}{
		{
			name: "masks sensitive json and summary values",
			event: Event{
				ID:          "event-1",
				StartedAt:   startedAt,
				ProjectPath: "C:\\repo\\project",
				Directory:   "C:\\repo\\project",
				Type:        EventType("build"),
				Summary:     "token=abc password: secret api_key=xyz keep=ok",
				Payload: json.RawMessage(`{
					"token":"abc",
					"nested":{"password":"secret"},
					"api_key":"xyz",
					"keep":"ok",
					"items":[{"token":"abc"}]
				}`),
				Source:      "terminal",
				Sensitivity: SensitivityNormal,
			},
			rules: RedactionRules{},
			want: Event{
				ID:          "event-1",
				StartedAt:   startedAt,
				EndedAt:     startedAt,
				ProjectPath: "C:\\repo\\project",
				Directory:   "C:\\repo\\project",
				Type:        EventType("build"),
				Summary:     "token=[REDACTED] password=[REDACTED] api_key=[REDACTED] keep=ok",
				Payload:     json.RawMessage(`{"api_key":"[REDACTED]","items":[{"token":"[REDACTED]"}],"keep":"ok","nested":{"password":"[REDACTED]"},"token":"[REDACTED]"}`),
				Source:      "terminal",
				Sensitivity: SensitivityRedacted,
			},
		},
		{
			name: "skips excluded project path",
			event: Event{
				StartedAt:   startedAt,
				ProjectPath: "C:\\repo\\vendor\\pkg",
				Directory:   "C:\\repo\\vendor\\pkg",
				Summary:     "token=abc",
				Payload:     json.RawMessage(`{"token":"abc"}`),
				Sensitivity: SensitivityNormal,
			},
			rules: RedactionRules{
				ExcludeGlobs: []string{"C:/repo/vendor/*"},
			},
			want: Event{
				StartedAt:   startedAt,
				EndedAt:     startedAt,
				ProjectPath: "C:\\repo\\vendor\\pkg",
				Directory:   "C:\\repo\\vendor\\pkg",
				Summary:     "token=abc",
				Payload:     json.RawMessage(`{"token":"abc"}`),
				Sensitivity: SensitivityNormal,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Redact(tt.event, tt.rules)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Redact() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
