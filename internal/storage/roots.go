package storage

import (
	"context"
	"fmt"
	"time"
)

type Root struct {
	ID      int64
	Path    string
	AddedAt time.Time
}

// AddRoot adds a new root to the database.
func (d *DB) AddRoot(ctx context.Context, path string) error {
	_, err := d.db.ExecContext(ctx, "INSERT OR IGNORE INTO roots (path) VALUES (?)", path)
	if err != nil {
		return fmt.Errorf("add root: %w", err)
	}
	return nil
}

// ListRoots returns all roots from the database.
func (d *DB) ListRoots(ctx context.Context) ([]Root, error) {
	rows, err := d.db.QueryContext(ctx, "SELECT id, path, added_at FROM roots ORDER BY path")
	if err != nil {
		return nil, fmt.Errorf("list roots: %w", err)
	}
	defer rows.Close()

	var result []Root
	for rows.Next() {
		var r Root
		var addedAtStr string
		if err := rows.Scan(&r.ID, &r.Path, &addedAtStr); err != nil {
			return nil, fmt.Errorf("scan root: %w", err)
		}
		
		r.AddedAt = parseTime(addedAtStr)
		
		result = append(result, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	
	if result == nil {
		result = []Root{}
	}
	return result, nil
}

// RemoveRoot removes a root from the database.
func (d *DB) RemoveRoot(ctx context.Context, path string) error {
	_, err := d.db.ExecContext(ctx, "DELETE FROM roots WHERE path = ?", path)
	if err != nil {
		return fmt.Errorf("remove root: %w", err)
	}
	return nil
}
