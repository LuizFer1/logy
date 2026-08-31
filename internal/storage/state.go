package storage

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CollectorState stores a collector cursor so collection can resume without duplicates.
type CollectorState struct {
	Collector   string
	ProjectPath string
	Cursor      string
	LastRunAt   time.Time
}

// SaveCollectorState upserts a collector cursor for a project.
func (d *DB) SaveCollectorState(ctx context.Context, state CollectorState) error {
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO collector_state (collector, project_path, cursor, last_run_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(collector, project_path) DO UPDATE SET
			cursor = excluded.cursor,
			last_run_at = excluded.last_run_at
	`, state.Collector, state.ProjectPath, state.Cursor, formatTime(state.LastRunAt))
	if err != nil {
		return fmt.Errorf("save collector state: %w", err)
	}
	return nil
}

// LoadCollectorState returns the stored cursor for a collector and project.
func (d *DB) LoadCollectorState(ctx context.Context, collector, projectPath string) (CollectorState, error) {
	var state CollectorState
	var lastRunAt string
	err := d.db.QueryRowContext(ctx, `
		SELECT collector, project_path, cursor, last_run_at
		FROM collector_state
		WHERE collector = ? AND project_path = ?
	`, collector, projectPath).Scan(&state.Collector, &state.ProjectPath, &state.Cursor, &lastRunAt)
	if err == sql.ErrNoRows {
		return CollectorState{}, err
	}
	if err != nil {
		return CollectorState{}, fmt.Errorf("load collector state: %w", err)
	}
	state.LastRunAt = parseTime(lastRunAt)
	return state, nil
}
