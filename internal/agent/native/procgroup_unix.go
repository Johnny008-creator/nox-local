//go:build !windows

package native

import (
	"os/exec"
	"syscall"
)

// ownGroup runs cmd in its own process group, and cancelling kills the
// whole group: a server started by the shell must die with it, or it keeps
// the output pipe open and the tool call never returns.
func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return killGroup(cmd.Process.Pid) }
}

func killGroup(pid int) error { return syscall.Kill(-pid, syscall.SIGKILL) }
