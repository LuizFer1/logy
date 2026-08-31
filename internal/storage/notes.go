package storage

import (
	"context"
	"fmt"
	"time"
)

// Note is a user-authored memory attached to an optional project.
type Note struct {
	ID          int64
	ProjectPath string
	Content     string
	CreatedAt   time.Time
}

// AddNote stores a manual note.
func (d *DB) AddNote(ctx context.Context, projectPath, content string) error {
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO notes (project_path, content) VALUES (?, ?)
	`, projectPath, content)
	if err != nil {
		return fmt.Errorf("add note: %w", err)
	}
	return nil
}

// ListNotes returns notes, optionally filtered by project path and created_at range.
func (d *DB) ListNotes(ctx context.Context, projectPath string, from, to time.Time) ([]Note, error) {
	query := `SELECT id, project_path, content, created_at FROM notes WHERE 1=1`
	var args []any
	if projectPath != "" {
		query += ` AND project_path = ?`
		args = append(args, projectPath)
	}
	if !from.IsZero() {
		query += ` AND created_at >= ?`
		args = append(args, from.UTC().Format(time.RFC3339))
	}
	if !to.IsZero() {
		query += ` AND created_at <= ?`
		args = append(args, to.UTC().Format(time.RFC3339))
	}
	query += ` ORDER BY created_at ASC, id ASC`

	rows, err := d.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list notes: %w", err)
	}
	defer rows.Close()

	var result []Note
	for rows.Next() {
		var n Note
		var createdAt string
		if err := rows.Scan(&n.ID, &n.ProjectPath, &n.Content, &createdAt); err != nil {
			return nil, fmt.Errorf("scan note: %w", err)
		}
		n.CreatedAt = parseTime(createdAt)
		result = append(result, n)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list notes rows: %w", err)
	}
	if result == nil {
		result = []Note{}
	}
	return result, nil
}
