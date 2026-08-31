package events

import (
	"time"
)

type EventFilter struct {
	From         time.Time
	To           time.Time
	ProjectPath  string
	Types        []EventType
	ExcludeGlobs []string
}

func (filter EventFilter) Matches(event Event) bool {
	normalized := Normalize(event)

	if !filter.From.IsZero() && normalized.EndedAt.Before(filter.From) {
		return false
	}
	if !filter.To.IsZero() && normalized.StartedAt.After(filter.To) {
		return false
	}
	if filter.ProjectPath != "" && !samePath(filter.ProjectPath, normalized.ProjectPath) {
		return false
	}
	if !filter.typeMatch(normalized.Type) {
		return false
	}
	if matchesAnyGlob(filter.ExcludeGlobs, normalized.ProjectPath, normalized.Directory) {
		return false
	}

	return true
}

func (filter EventFilter) typeMatch(eventType EventType) bool {
	if len(filter.Types) == 0 {
		return true
	}

	for _, allowed := range filter.Types {
		if allowed == eventType {
			return true
		}
	}

	return false
}
