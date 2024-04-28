package modmake

import (
	"os/exec"
)

func cancelIncludeChildren(cmd *exec.Cmd) func() error {
	return func() error {
		if cmd.Process != nil {
			return exec.Command("TASKKILL", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
		}
		return nil
	}
}
