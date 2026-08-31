//go:build windows

package daemon

import (
	"os/exec"
	"testing"
)

func TestDetachUsesCreateNoWindow(t *testing.T) {
	t.Parallel()
	cmd := exec.Command("logy", "start", "--foreground")
	Detach(cmd)
	if cmd.SysProcAttr == nil {
		t.Fatal("SysProcAttr is nil")
	}
	if cmd.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatalf("CreationFlags = %#x, want CREATE_NO_WINDOW (%#x)", cmd.SysProcAttr.CreationFlags, createNoWindow)
	}
	if cmd.SysProcAttr.CreationFlags&detachedProcess != 0 {
		t.Fatalf("CreationFlags includes DETACHED_PROCESS, which conflicts with CREATE_NO_WINDOW")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("HideWindow = false, want true")
	}
}
