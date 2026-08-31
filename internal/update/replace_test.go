package update

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReplaceExecutable(t *testing.T) {
	dir := t.TempDir()
	exe := "logy"
	if runtime.GOOS == "windows" {
		exe = "logy.exe"
	}
	exePath := filepath.Join(dir, exe)
	newPath := exePath + ".new"

	if err := os.WriteFile(exePath, []byte("old-bin"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newPath, []byte("new-bin"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := ReplaceExecutable(exePath, newPath); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new-bin" {
		t.Fatalf("exe content %q", got)
	}
	if _, err := os.Stat(newPath); !os.IsNotExist(err) {
		t.Fatalf(".new should be gone, err=%v", err)
	}
}

func TestCleanupStaleOld(t *testing.T) {
	dir := t.TempDir()
	exePath := filepath.Join(dir, "logy.exe")
	oldPath := exePath + ".old"
	if err := os.WriteFile(oldPath, []byte("stale"), 0644); err != nil {
		t.Fatal(err)
	}
	CleanupStaleOld(exePath)
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Fatalf(".old still present: %v", err)
	}
	// Missing .old should be a no-op.
	CleanupStaleOld(exePath)
}
