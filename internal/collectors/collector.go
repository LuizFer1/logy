package collectors

import (
	"context"

	"logy/internal/events"
)

// Project is a Git repository the collectors inspect.
type Project struct {
	Path string
	Name string
}

// Collector reads activity for a single project.
type Collector interface {
	Name() string
	Collect(ctx context.Context, project Project) ([]events.Event, error)
}
