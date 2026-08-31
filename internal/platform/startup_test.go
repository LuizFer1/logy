package platform

import (
	"runtime"
	"strings"
	"testing"
)

func TestEnableStartupNonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows covered by TestStartupIdempotent")
	}
	err := EnableStartup(t.TempDir())
	if err == nil {
		t.Fatal("expected error on non-Windows")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "windows") {
		t.Fatalf("error should mention Windows: %v", err)
	}
}

func TestStartupIdempotent(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("startup registry only on Windows")
	}

	home := t.TempDir()

	// Ensure clean slate
	_ = DisableStartup()

	if err := EnableStartup(home); err != nil {
		t.Fatalf("EnableStartup() error = %v", err)
	}
	enabled, err := StartupEnabled()
	if err != nil {
		t.Fatalf("StartupEnabled() error = %v", err)
	}
	if !enabled {
		t.Fatal("expected startup enabled after EnableStartup")
	}

	// Idempotent re-enable
	if err := EnableStartup(home); err != nil {
		t.Fatalf("second EnableStartup() error = %v", err)
	}
	enabled, err = StartupEnabled()
	if err != nil {
		t.Fatalf("StartupEnabled() after re-enable error = %v", err)
	}
	if !enabled {
		t.Fatal("expected still enabled")
	}

	if err := DisableStartup(); err != nil {
		t.Fatalf("DisableStartup() error = %v", err)
	}
	enabled, err = StartupEnabled()
	if err != nil {
		t.Fatalf("StartupEnabled() after disable error = %v", err)
	}
	if enabled {
		t.Fatal("expected disabled after DisableStartup")
	}

	// Idempotent disable
	if err := DisableStartup(); err != nil {
		t.Fatalf("second DisableStartup() error = %v", err)
	}
}
