package git

import (
	"context"
	"errors"
	"fmt"
	"github.com/saylorsolutions/modmake"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	ErrNoGit    = errors.New("unable to locate git executable")
	ExecTimeout = 5 * time.Second // Used to limit how long [Exec] should operate. May be overridden.
)

// Exec will produce a [modmake.Command] that executes a git command.
// This function will panic if a git executable cannot be located on the PATH.
func Exec(subcmd string, args ...string) *modmake.Command {
	path, err := exec.LookPath("git")
	if err != nil {
		panic(fmt.Errorf("%w: %v", ErrNoGit, err))
	}
	if len(path) == 0 {
		panic(ErrNoGit)
	}
	return modmake.Exec(append([]string{path, subcmd}, args...)...).LogGroup("git " + subcmd).CaptureStdin()
}

// ExecOutput will delegate to Exec and run the returned [modmake.Command], but will collect its output into a string.
// This will also use the execution limit of [ExecTimeout].
// If you want to override this timeout or exit after some other condition, then use [ExecOutputCtx].
func ExecOutput(subcmd string, args ...string) (string, error) {
	timeout, cancel := context.WithTimeout(context.Background(), ExecTimeout)
	defer cancel()
	return ExecOutputCtx(timeout, subcmd, args...)
}

// ExecOutputCtx will delegate to Exec and run the returned [modmake.Command], but will collect its output into a string.
// If the context is cancelled, then the command will exit.
func ExecOutputCtx(ctx context.Context, subcmd string, args ...string) (string, error) {
	var (
		buf strings.Builder
	)
	if err := Exec(subcmd, args...).Output(&buf).Run(ctx); err != nil {
		return buf.String(), err
	}
	return strings.TrimSpace(buf.String()), nil
}

// RepositoryRoot will attempt to locate the root path of the git repository.
func RepositoryRoot() string {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	prev := ""
	for {
		if prev == cwd {
			panic(fmt.Sprintf("directory '%s' is not in a git repository", cwd))
		}
		fi, err := os.Stat(filepath.Join(cwd, ".git"))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if fi.IsDir() {
			return cwd
		}
		prev = cwd
		cwd = filepath.Dir(cwd)
	}
}

// BranchName returns the name of the currently checked out branch.
func BranchName() string {
	output, err := ExecOutput("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		panic(err)
	}
	return output
}

// CommitHash returns the commit hash of the currently checked out commit.
func CommitHash() string {
	output, err := ExecOutput("rev-parse", "HEAD")
	if err != nil {
		panic(err)
	}
	return output
}
