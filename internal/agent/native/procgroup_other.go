//go:build windows

package native

import (
	"os"
	"os/exec"
)

func ownGroup(cmd *exec.Cmd) {}

func killGroup(pid int) error {
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
