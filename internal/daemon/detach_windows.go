//go:build windows

package daemon

import (
	"os/exec"
	"syscall"
)

const (
	// DETACHED_PROCESS conflicts with CREATE_NO_WINDOW on Windows; do not combine them.
	detachedProcess       = 0x00000008
	createNewProcessGroup = 0x00000200
	createNoWindow        = 0x08000000
)

// Detach configures cmd to survive after the parent exits without flashing a console.
func Detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow | createNewProcessGroup,
	}
}
