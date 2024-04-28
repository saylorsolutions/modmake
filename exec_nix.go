//go:build !windows

package modmake

import (
	"os/exec"
	"syscall"
)

func cancelIncludeChildren(cmd *exec.Cmd) func() error {
	return func() error {
		if cmd.Process != nil {
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
		}
		return nil
	}
}
