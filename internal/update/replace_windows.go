//go:build windows

package update

import "os"

// ReplaceExecutable replaces the running binary at exePath with the file at newPath.
// On Windows: rename exe→.old, new→exe, then best-effort remove .old.
func ReplaceExecutable(exePath, newPath string) error {
	old := exePath + ".old"
	_ = os.Remove(old)
	if err := os.Rename(exePath, old); err != nil {
		return err
	}
	if err := os.Rename(newPath, exePath); err != nil {
		// Try to restore the original binary.
		_ = os.Rename(old, exePath)
		return err
	}
	_ = os.Remove(old)
	return nil
}
