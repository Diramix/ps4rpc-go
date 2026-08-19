//go:build !windows

package daemon

import (
	"os/exec"
	"syscall"
)

func detachedCommand(exe string, args ...string) *exec.Cmd {
	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	return cmd
}
