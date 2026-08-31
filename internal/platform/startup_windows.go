//go:build windows

package platform

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

const runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`

// EnableStartup registers Logy to start with Windows (HKCU Run). Idempotent.
func EnableStartup(home string) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	cmd := startupCommand(exe, home)

	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("open Run key: %w", err)
	}
	defer key.Close()

	if err := key.SetStringValue(startupValueName, cmd); err != nil {
		return fmt.Errorf("set Run value: %w", err)
	}
	return nil
}

// DisableStartup removes the Logy Run value. Idempotent if already absent.
func DisableStartup() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open Run key: %w", err)
	}
	defer key.Close()

	err = key.DeleteValue(startupValueName)
	if err == nil || err == registry.ErrNotExist {
		return nil
	}
	return fmt.Errorf("delete Run value: %w", err)
}

// StartupEnabled reports whether the Logy Run value is present.
func StartupEnabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("open Run key: %w", err)
	}
	defer key.Close()

	_, _, err = key.GetStringValue(startupValueName)
	if err == registry.ErrNotExist {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read Run value: %w", err)
	}
	return true, nil
}
