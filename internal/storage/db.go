package storage

import (
	"context"
	"database/sql"
	"fmt"

	"logy/internal/events"

	_ "modernc.org/sqlite"
)

// QueryService is the stable read API used by CLI and future TUI.
type QueryService interface {
	Search(ctx context.Context, filter events.EventFilter) ([]events.Event, error)
}

var _ QueryService = (*DB)(nil)


// DB wraps a sql.DB with Logy-specific operations.
type DB struct {
	db *sql.DB
}

// Open creates or opens the SQLite database at the given path.
// It enables WAL mode and creates all tables if they don't exist.
func Open(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable wal mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA busy_timeout=5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set busy timeout: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("init schema: %w", err)
	}

	return &DB{db: db}, nil
}

// Close closes the database.
func (d *DB) Close() error {
	return d.db.Close()
}
