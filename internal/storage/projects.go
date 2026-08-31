package storage

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"
)

// Project is a discovered Git repository tracked by Logy.
type Project struct {
	ID           int64
	Path         string
	Name         string
	RootID       int64
	DiscoveredAt time.Time
	LastSeenAt   time.Time
}

// UpsertProject inserts a project or refreshes last_seen_at when it already exists.
func (d *DB) UpsertProject(ctx context.Context, proj Project) error {
	if proj.Name == "" {
		proj.Name = filepath.Base(proj.Path)
	}
	now := formatTime(time.Now())
	var rootID any
	if proj.RootID != 0 {
		rootID = proj.RootID
	}
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO projects (path, name, root_id, discovered_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			name = excluded.name,
			root_id = COALESCE(excluded.root_id, projects.root_id),
			last_seen_at = excluded.last_seen_at
	`, proj.Path, proj.Name, rootID, now, now)
	if err != nil {
		return fmt.Errorf("upsert project: %w", err)
	}
	return nil
}

// ListProjects returns all tracked projects ordered by path.
func (d *DB) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT id, path, name, COALESCE(root_id, 0), discovered_at, last_seen_at
		FROM projects
		ORDER BY path
	`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var result []Project
	for rows.Next() {
		var p Project
		var discoveredAt, lastSeenAt string
		if err := rows.Scan(&p.ID, &p.Path, &p.Name, &p.RootID, &discoveredAt, &lastSeenAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		p.DiscoveredAt = parseTime(discoveredAt)
		p.LastSeenAt = parseTime(lastSeenAt)
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list projects rows: %w", err)
	}
	if result == nil {
		result = []Project{}
	}
	return result, nil
}

// IgnoreProject marks a project path as ignored. The project row is created if needed.
func (d *DB) IgnoreProject(ctx context.Context, path string) error {
	if err := d.UpsertProject(ctx, Project{Path: path, Name: filepath.Base(path)}); err != nil {
		return err
	}
	_, err := d.db.ExecContext(ctx, `
		INSERT INTO ignored_projects (project_id)
		SELECT id FROM projects WHERE path = ?
		ON CONFLICT(project_id) DO NOTHING
	`, path)
	if err != nil {
		return fmt.Errorf("ignore project: %w", err)
	}
	return nil
}

// UnignoreProject removes a persistent ignore so the project can be rediscovered.
func (d *DB) UnignoreProject(ctx context.Context, path string) error {
	_, err := d.db.ExecContext(ctx, `
		DELETE FROM ignored_projects
		WHERE project_id = (SELECT id FROM projects WHERE path = ?)
	`, path)
	if err != nil {
		return fmt.Errorf("unignore project: %w", err)
	}
	return nil
}

// IsIgnored reports whether the given project path is persistently ignored.
func (d *DB) IsIgnored(ctx context.Context, path string) (bool, error) {
	var ignored int
	err := d.db.QueryRowContext(ctx, `
		SELECT 1
		FROM ignored_projects i
		JOIN projects p ON p.id = i.project_id
		WHERE p.path = ?
	`, path).Scan(&ignored)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("is ignored: %w", err)
	}
	return true, nil
}

// ListIgnoredPaths returns ignored project paths.
func (d *DB) ListIgnoredPaths(ctx context.Context) ([]string, error) {
	rows, err := d.db.QueryContext(ctx, `
		SELECT p.path
		FROM ignored_projects i
		JOIN projects p ON p.id = i.project_id
		ORDER BY p.path
	`)
	if err != nil {
		return nil, fmt.Errorf("list ignored: %w", err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			return nil, fmt.Errorf("scan ignored path: %w", err)
		}
		result = append(result, path)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list ignored rows: %w", err)
	}
	if result == nil {
		result = []string{}
	}
	return result, nil
}
