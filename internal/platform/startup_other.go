//go:build !windows

package platform

import "fmt"

// EnableStartup is only supported on Windows.
func EnableStartup(home string) error {
	return fmt.Errorf("startup registration is only supported on Windows")
}

// DisableStartup is a no-op on non-Windows platforms.
func DisableStartup() error {
	return nil
}

// StartupEnabled always reports false outside Windows.
func StartupEnabled() (bool, error) {
	return false, nil
}
