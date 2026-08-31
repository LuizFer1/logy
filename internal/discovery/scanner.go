package discovery

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Result represents a discovered Git repository.
type Result struct {
	Path string // absolute path to the repo directory (parent of .git)
	Name string // base name of the directory
}

// ScanOptions controls the discovery behavior.
type ScanOptions struct {
	MaxDepth int      // max directory depth to search, default 3
	Ignored  []string // absolute paths to skip
}

// Scan walks the given root directories looking for .git folders.
// It returns all discovered repositories, skipping ignored paths.
// It respects MaxDepth to avoid traversing too deep.
// Errors in individual directories are logged/skipped, not fatal.
func Scan(roots []string, opts ScanOptions) ([]Result, error) {
	var results []Result
	maxDepth := opts.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 3
	}

	// Normalize ignored paths for case-insensitive comparison on Windows
	ignoredMap := make(map[string]struct{})
	for _, ign := range opts.Ignored {
		ignoredMap[strings.ToLower(filepath.Clean(ign))] = struct{}{}
	}

	for _, root := range roots {
		rootStr := filepath.Clean(root)
		_, err := os.Stat(rootStr)
		if os.IsNotExist(err) {
			continue
		}

		err = filepath.WalkDir(rootStr, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // Skip errors
			}

			// Check ignored
			cleanPath := strings.ToLower(filepath.Clean(path))
			if _, ok := ignoredMap[cleanPath]; ok {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Calculate depth
			rel, err := filepath.Rel(rootStr, path)
			if err != nil {
				return nil
			}
			depth := 0
			if rel != "." {
				depth = strings.Count(rel, string(os.PathSeparator)) + 1
			}

			if depth > maxDepth {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip symlinks
			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}

			if d.IsDir() {
				// .git may be a directory or a gitfile (worktree/submodule).
				gitPath := filepath.Join(path, ".git")
				if info, err := os.Stat(gitPath); err == nil {
					results = append(results, Result{
						Path: path,
						Name: filepath.Base(path),
					})
					_ = info
					return filepath.SkipDir // stop under this repo; siblings still scanned
				}
			}

			return nil
		})
		if err != nil {
			continue
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Path < results[j].Path
	})

	return results, nil
}
