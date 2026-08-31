//go:build !windows

package update

import "os"

// ReplaceExecutable replaces the running binary at exePath with the file at newPath.
// On Unix, rename within the same directory is atomic.
func ReplaceExecutable(exePath, newPath string) error {
	return os.Rename(newPath, exePath)
}
