package discovery

import (
	"path/filepath"
	"strings"
)

// Changes computes what repositories were added or removed compared to a previous scan.
type Changes struct {
	Added   []Result
	Removed []Result
}

// Diff compares two scan results and returns what changed.
func Diff(previous, current []Result) Changes {
	prevMap := make(map[string]Result)
	currMap := make(map[string]Result)

	for _, p := range previous {
		norm := strings.ToLower(filepath.Clean(p.Path))
		prevMap[norm] = p
	}

	for _, c := range current {
		norm := strings.ToLower(filepath.Clean(c.Path))
		currMap[norm] = c
	}

	var added []Result
	var removed []Result

	for norm, c := range currMap {
		if _, ok := prevMap[norm]; !ok {
			added = append(added, c)
		}
	}

	for norm, p := range prevMap {
		if _, ok := currMap[norm]; !ok {
			removed = append(removed, p)
		}
	}

	return Changes{
		Added:   added,
		Removed: removed,
	}
}
