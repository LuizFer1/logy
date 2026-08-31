package events

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestNormalize(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, time.August, 31, 10, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(-2 * time.Hour)

	tests := []struct {
		name  string
		event Event
		want  Event
	}{
		{
			name: "normalizes metadata and compacts valid json",
			event: Event{
				ID:          "  event-1  ",
				StartedAt:   startedAt,
				EndedAt:     endedAt,
				ProjectPath: "C:\\repo\\..\\repo\\project\\",
				Directory:   "C:\\repo\\project\\sub\\.\\",
				Type:        EventType("  build  "),
				Summary:     "  build finished  ",
				Payload:     json.RawMessage("  {\"token\":\"abc\",\"nested\":{\"password\":\"secret\"}}  "),
				Source:      "  terminal  ",
				Sensitivity: Sensitivity(""),
			},
			want: Event{
				ID:          "event-1",
				StartedAt:   startedAt,
				EndedAt:     startedAt,
				ProjectPath: "C:\\repo\\project",
				Directory:   "C:\\repo\\project\\sub",
				Type:        EventType("build"),
				Summary:     "build finished",
				Payload:     json.RawMessage("{\"token\":\"abc\",\"nested\":{\"password\":\"secret\"}}"),
				Source:      "terminal",
				Sensitivity: SensitivityNormal,
			},
		},
		{
			name: "quotes invalid payload text",
			event: Event{
				StartedAt:   startedAt,
				ProjectPath: "C:\\repo\\project",
				Directory:   "C:\\repo\\project",
				Payload:     json.RawMessage("  hello world  "),
			},
			want: Event{
				StartedAt:   startedAt,
				EndedAt:     startedAt,
				ProjectPath: "C:\\repo\\project",
				Directory:   "C:\\repo\\project",
				Payload:     json.RawMessage(`"hello world"`),
				Sensitivity: SensitivityNormal,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			originalPayload := append([]byte(nil), tt.event.Payload...)
			got := Normalize(tt.event)

			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Normalize() = %#v, want %#v", got, tt.want)
			}

			if len(originalPayload) > 0 {
				tt.event.Payload[0] = 'X'
				if string(got.Payload) == string(tt.event.Payload) {
					t.Fatalf("Normalize() reused payload storage")
				}
			}
		})
	}
}

func TestEventFilterMatches(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, time.August, 31, 10, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(1 * time.Hour)

	event := Event{
		StartedAt:   startedAt,
		EndedAt:     endedAt,
		ProjectPath: "C:\\repo\\project",
		Directory:   "C:\\repo\\project\\sub",
		Type:        EventType("build"),
	}

	tests := []struct {
		name   string
		filter EventFilter
		want   bool
	}{
		{
			name:   "empty filter matches",
			filter: EventFilter{},
			want:   true,
		},
		{
			name: "date range overlaps start",
			filter: EventFilter{
				From: startedAt.Add(-30 * time.Minute),
				To:   startedAt.Add(30 * time.Minute),
			},
			want: true,
		},
		{
			name: "date range outside event",
			filter: EventFilter{
				From: endedAt.Add(1 * time.Minute),
				To:   endedAt.Add(2 * time.Hour),
			},
			want: false,
		},
		{
			name: "project filter matches normalized path",
			filter: EventFilter{
				ProjectPath: "C:/repo/project",
			},
			want: true,
		},
		{
			name: "project filter rejects different path",
			filter: EventFilter{
				ProjectPath: "C:/repo/other",
			},
			want: false,
		},
		{
			name: "type filter matches",
			filter: EventFilter{
				Types: []EventType{EventType("build"), EventType("note")},
			},
			want: true,
		},
		{
			name: "type filter rejects other type",
			filter: EventFilter{
				Types: []EventType{EventType("note")},
			},
			want: false,
		},
		{
			name: "glob exclusion rejects project",
			filter: EventFilter{
				ExcludeGlobs: []string{"C:/repo/project/*"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.filter.Matches(event)
			if got != tt.want {
				t.Fatalf("Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}
