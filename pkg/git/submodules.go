package git

import (
	"github.com/saylorsolutions/modmake"
)

// SubmoduleUpdateInit will run 'git submodule update --init <path>'.
func SubmoduleUpdateInit(path ...string) modmake.Task {
	args := []string{"update", "--init"}
	if len(path) > 0 {
		args = append(args, path[0])
	}
	return Exec("submodule", args...).Task()
}

// SubmoduleUpdateRemote will run 'git submodule update --remote <path>'.
func SubmoduleUpdateRemote(path ...string) modmake.Task {
	args := []string{"update", "--remote"}
	if len(path) > 0 {
		args = append(args, path[0])
	}
	return Exec("submodule", args...).Task()
}
