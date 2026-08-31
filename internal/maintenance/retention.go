package maintenance

import (
	"context"
	"fmt"
	"time"

	"logy/internal/events"
	"logy/internal/storage"
)

// RetentionOptions selects which events to purge. Notes, roots, ignored projects,
// and config are never deleted.
type RetentionOptions struct {
	OlderThan   time.Time // delete events with started_at at or before this
	ProjectPath string    // optional
	DryRun      bool
}

// RetentionResult reports how many events were deleted (or would be, for dry-run).
type RetentionResult struct {
	Deleted int64
	DryRun  bool
}

// PurgeEvents deletes matching events via storage.DeleteEvents.
// DryRun counts matches with Search and deletes nothing.
func PurgeEvents(ctx context.Context, db *storage.DB, opts RetentionOptions) (RetentionResult, error) {
	if db == nil {
		return RetentionResult{}, fmt.Errorf("db is nil")
	}
	if opts.OlderThan.IsZero() {
		return RetentionResult{}, fmt.Errorf("OlderThan is required")
	}

	filter := events.EventFilter{
		To:          opts.OlderThan,
		ProjectPath: opts.ProjectPath,
	}

	if opts.DryRun {
		matched, err := db.Search(ctx, filter)
		if err != nil {
			return RetentionResult{}, fmt.Errorf("dry-run search: %w", err)
		}
		return RetentionResult{Deleted: int64(len(matched)), DryRun: true}, nil
	}

	n, err := db.DeleteEvents(ctx, filter)
	if err != nil {
		return RetentionResult{}, fmt.Errorf("purge events: %w", err)
	}
	return RetentionResult{Deleted: n, DryRun: false}, nil
}
