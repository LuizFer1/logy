package events

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"
)

type Event struct {
	ID          string
	StartedAt   time.Time
	EndedAt     time.Time
	ProjectPath string
	Directory   string
	Type        EventType
	Summary     string
	Payload     json.RawMessage
	Source      string
	Sensitivity Sensitivity
}

type EventType string

type Sensitivity string

const (
	SensitivityNormal   Sensitivity = "normal"
	SensitivityRedacted Sensitivity = "redacted"
)

func Normalize(event Event) Event {
	normalized := event
	normalized.ID = strings.TrimSpace(normalized.ID)
	normalized.StartedAt = normalizeTime(normalized.StartedAt)
	normalized.EndedAt = normalizeTime(normalized.EndedAt)
	normalized.ProjectPath = cleanPath(normalized.ProjectPath)
	normalized.Directory = cleanPath(normalized.Directory)
	normalized.Type = EventType(strings.TrimSpace(string(normalized.Type)))
	normalized.Summary = strings.TrimSpace(normalized.Summary)
	normalized.Payload = normalizePayload(normalized.Payload)
	normalized.Source = strings.TrimSpace(normalized.Source)
	normalized.Sensitivity = normalizeSensitivity(normalized.Sensitivity)

	if normalized.EndedAt.IsZero() && !normalized.StartedAt.IsZero() {
		normalized.EndedAt = normalized.StartedAt
	}
	if !normalized.StartedAt.IsZero() && !normalized.EndedAt.IsZero() && normalized.EndedAt.Before(normalized.StartedAt) {
		normalized.EndedAt = normalized.StartedAt
	}

	return normalized
}

func normalizeTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}

	return value.UTC().Round(0)
}

func normalizeSensitivity(value Sensitivity) Sensitivity {
	switch Sensitivity(strings.TrimSpace(string(value))) {
	case SensitivityRedacted:
		return SensitivityRedacted
	default:
		return SensitivityNormal
	}
}

func cleanPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	cleaned := filepath.Clean(value)
	if cleaned == "." {
		return ""
	}

	return cleaned
}

func normalizePayload(payload json.RawMessage) json.RawMessage {
	trimmed := bytes.TrimSpace(payload)
	if len(trimmed) == 0 {
		return nil
	}

	var compacted bytes.Buffer
	if err := json.Compact(&compacted, trimmed); err == nil {
		return append(json.RawMessage(nil), compacted.Bytes()...)
	}

	quoted, err := json.Marshal(string(trimmed))
	if err != nil {
		return nil
	}

	return append(json.RawMessage(nil), quoted...)
}
