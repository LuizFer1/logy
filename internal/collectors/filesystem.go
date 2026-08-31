package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"logy/internal/events"
)

const (
	defaultFilesystemDebounce = 500 * time.Millisecond
	filesystemSampleLimit     = 10
)

// Filesystem optionally scans a project for recent/current file changes.
// Disabled by default; when enabled, changes are grouped into one event.
type Filesystem struct {
	Debounce time.Duration // default 500ms; Collect groups without sleeping
	Enabled  bool          // default false
}

func (Filesystem) Name() string { return "filesystem" }

func (f Filesystem) Collect(ctx context.Context, project Project) ([]events.Event, error) {
	if !f.Enabled {
		return nil, nil
	}
	if f.Debounce <= 0 {
		f.Debounce = defaultFilesystemDebounce
	}

	root := project.Path
	if root == "" {
		return nil, nil
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return nil, err
	}

	var changed []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		relSlash := filepath.ToSlash(rel)
		if d.IsDir() {
			if shouldIgnoreFilesystemPath(relSlash) {
				return fs.SkipDir
			}
			return nil
		}
		if shouldIgnoreFilesystemPath(relSlash) {
			return nil
		}
		changed = append(changed, relSlash)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(changed) == 0 {
		return nil, nil
	}

	sample := changed
	if len(sample) > filesystemSampleLimit {
		sample = sample[:filesystemSampleLimit]
	}
	now := time.Now().UTC()
	payload, _ := json.Marshal(map[string]any{
		"changed": len(changed),
		"sample":  sample,
	})
	summary := fmt.Sprintf("%d changed", len(changed))
	evt := events.Normalize(events.Event{
		ID:          fmt.Sprintf("filesystem.change:%s:%d", root, now.UnixNano()),
		StartedAt:   now,
		EndedAt:     now,
		ProjectPath: root,
		Directory:   root,
		Type:        "filesystem.change",
		Summary:     summary,
		Payload:     payload,
		Source:      "filesystem",
	})
	return []events.Event{evt}, nil
}

func shouldIgnoreFilesystemPath(relSlash string) bool {
	relSlash = strings.TrimPrefix(relSlash, "./")
	base := filepath.Base(relSlash)
	parts := strings.Split(relSlash, "/")

	for _, part := range parts {
		switch part {
		case "node_modules", ".git", "vendor", "secrets":
			return true
		}
	}
	return strings.HasPrefix(base, ".env")
}
