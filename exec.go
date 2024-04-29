package modmake

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

type Command struct {
	err         error
	workdir     string
	cmd         string
	initialArgs []string
	args        []string
	env         []string
	stdout      io.Writer
	stderr      io.Writer
	stdin       io.Reader
}

// Exec creates a new Command representing running an external application.
func Exec(cmdAndInitArgs ...string) *Command {
	i := &Command{
		env:    os.Environ(),
		stdout: os.Stdout,
		stderr: os.Stderr,
		stdin:  nil,
	}
	work, err := filepath.Abs(".")
	if err != nil {
		i.err = err
		return i
	}
	i.workdir = work
	switch len(cmdAndInitArgs) {
	case 0:
		i.err = errors.New("no command specified")
	case 1:
		i.cmd = cmdAndInitArgs[0]
	default:
		i.cmd = cmdAndInitArgs[0]
		i.initialArgs = cmdAndInitArgs[1:]
	}
	return i
}

// OptArg will add the specified arg(s) if the condition evaluates to true.
func (i *Command) OptArg(condition bool, args ...string) *Command {
	if i.err != nil {
		return i
	}
	if condition {
		i.Arg(args...)
	}
	return i
}

// Arg adds the given arguments to the Command.
func (i *Command) Arg(args ...string) *Command {
	if i.err != nil {
		return i
	}
	if len(args) == 0 {
		return i
	}
	i.args = append(i.args, args...)
	if len(i.args) == cap(i.args) {
		newArgs := make([]string, len(i.args), 2*len(i.args))
		copy(newArgs, i.args)
		i.args = newArgs
	}
	return i
}

// Env sets an environment variable for the running Command.
func (i *Command) Env(key, value string) *Command {
	if i.err != nil {
		return i
	}
	if len(key) == 0 {
		i.err = errors.New("attempt to add environment variable with empty key")
		return i
	}
	i.env = append(i.env, fmt.Sprintf("%s=%s", key, value))
	return i
}

// WorkDir sets the working directory in which to execute the Command.
func (i *Command) WorkDir(workdir PathString) *Command {
	if i.err != nil {
		return i
	}
	work, err := workdir.Abs()
	if err != nil {
		i.err = err
		return i
	}
	i.workdir = work.String()
	return i
}

// Silent will prevent command output.
func (i *Command) Silent() *Command {
	if i.err != nil {
		return i
	}
	null, err := os.Open(os.DevNull)
	if err != nil {
		i.err = err
		return i
	}
	i.stderr = null
	i.stdout = null
	return i
}

// CaptureStdin will make the Command pass os.Stdin to the executed process.
func (i *Command) CaptureStdin() *Command {
	if i.err != nil {
		return i
	}
	i.stdin = os.Stdin
	return i
}

// Stdout will capture all data written to the Command's stdout stream and write it to w.
func (i *Command) Stdout(w io.Writer) *Command {
	if i.err != nil {
		return i
	}
	i.stdout = w
	return i
}

// Stderr will capture all data written to the Command's stderr stream and write it to w.
func (i *Command) Stderr(w io.Writer) *Command {
	if i.err != nil {
		return i
	}
	i.stderr = w
	return i
}

// Output will redirect all data written to either stdout or stderr to w.
func (i *Command) Output(w io.Writer) *Command {
	return i.Stdout(w).Stderr(w)
}

func (i *Command) Run(ctx context.Context) error {
	if i.err != nil {
		return i.err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		cmd := exec.CommandContext(ctx, i.cmd, append(i.initialArgs, i.args...)...)
		cmd.Env = i.env
		cmd.Stdout = i.stdout
		cmd.Stderr = i.stderr
		cmd.Stdin = i.stdin
		cmd.Dir = i.workdir
		customizeCmd(cmd)
		cmd.Cancel = cancelIncludeChildren(cmd)
		if err := cmd.Run(); err != nil {
			return err
		}
		return nil
	}
}

func (i *Command) Task() Task {
	return i.Run
}
