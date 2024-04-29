//go:build !windows

package modmake

import (
	"os/exec"
	"syscall"
)

func customizeCmd(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return
}

func cancelIncludeChildren(cmd *exec.Cmd) func() error {
	return func() error {
		if cmd.Process != nil {
			return syscall.Kill(-cmd.Process.Pid, syscall.SIGINT)
		}
		return nil
	}
}
