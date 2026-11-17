package local

import (
	"os/exec"
	"syscall"
)

// detach makes the kernel stop llama-server if nox-local dies without
// running Close (crash, kill -9), so no 6 GB process is left behind.
func detach(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
}
