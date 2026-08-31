package events

import (
	"path"
	"strings"
)

func samePath(left, right string) bool {
	left = cleanPath(left)
	right = cleanPath(right)
	if left == "" || right == "" {
		return left == right
	}

	return strings.EqualFold(filepathToSlash(left), filepathToSlash(right))
}

func filepathToSlash(value string) string {
	return strings.ReplaceAll(value, "\\", "/")
}

func matchesAnyGlob(globs []string, values ...string) bool {
	if len(globs) == 0 || len(values) == 0 {
		return false
	}

	for _, value := range values {
		value = cleanPath(value)
		if value == "" {
			continue
		}

		candidate := filepathToSlash(value)
		for _, glob := range globs {
			glob = strings.TrimSpace(glob)
			if glob == "" {
				continue
			}

			match, err := path.Match(filepathToSlash(glob), candidate)
			if err == nil && match {
				return true
			}
		}
	}

	return false
}
