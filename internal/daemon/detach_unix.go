//go:build unix

package daemon

import (
	"os/exec"
	"syscall"
)

// Detach configures cmd to survive after the parent process exits.
func Detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
