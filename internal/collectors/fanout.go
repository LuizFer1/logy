package collectors

import (
	"context"

	"logy/internal/events"
)

// FanOut runs a per-project collector across many projects.
type FanOut struct {
	Collector Collector
	Projects  []Project
}

func (f FanOut) Name() string {
	if f.Collector == nil {
		return ""
	}
	return f.Collector.Name()
}

func (f FanOut) Collect(ctx context.Context) ([]events.Event, error) {
	var all []events.Event
	for _, project := range f.Projects {
		if ctx.Err() != nil {
			break
		}
		evts, err := f.Collector.Collect(ctx, project)
		if err != nil {
			continue
		}
		all = append(all, evts...)
	}
	return all, nil
}
