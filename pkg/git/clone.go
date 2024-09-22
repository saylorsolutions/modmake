package git

import (
	"github.com/saylorsolutions/modmake"
)

// CloneAt will clone the repository identified by repo at the given destination.
func CloneAt(repo string, destination modmake.PathString) modmake.Task {
	return Exec("clone", repo, destination.String()).Run
}
