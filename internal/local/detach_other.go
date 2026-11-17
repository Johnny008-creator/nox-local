//go:build !linux

package local

import "os/exec"

func detach(c *exec.Cmd) {}
