package modmake

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// Script will execute each Task in order, returning the first error.
func Script(fns ...Runner) Task {
	return func(ctx context.Context) error {
		ctx, _ = WithGroup(ctx, "script")
		for i, fn := range fns {
			ctx, log := WithGroup(ctx, fmt.Sprintf("%d", i))
			run := ContextAware(fn)
			if err := run.Run(ctx); err != nil {
				return log.WrapErr(err)
			}
		}
		return nil
	}
}

// IfNotExists will skip executing the Runner if the given file exists, returning nil.
// This is similar to the default Make behavior of skipping a task if the target file already exists.
func IfNotExists(file PathString, r Runner) Task {
	return func(ctx context.Context) error {
		if !file.Exists() {
			return r.Run(ctx)
		}
		return nil
	}
}

// IfExists will execute the Runner if the file exists, returning nil otherwise.
func IfExists(file PathString, r Runner) Task {
	return func(ctx context.Context) error {
		if file.Exists() {
			return r.Run(ctx)
		}
		return nil
	}
}

// IfError will create a Runner with an error handler Runner that is only executed if the base Runner returns an error.
// If both the base Runner and the handler return an error, then the handler's error will be returned.
func IfError(canErr Runner, handler Runner) Task {
	return func(ctx context.Context) error {
		if err := canErr.Run(ctx); err != nil {
			return handler.Run(ctx)
		}
		return nil
	}
}

// Print will create a Runner that logs an informational message.
func Print(msg string, args ...any) Task {
	return WithoutErr(func(ctx context.Context) {
		_, log := WithGroup(ctx, "print")
		log.Info(msg, args...)
	})
}

// Chdir will change the current working directory to newWorkdir and run the Runner in that context.
// Whether the Runner executes successfully or not, the working directory will be reset back to its original state.
func Chdir(newWorkdir PathString, runner Runner) Task {
	return func(ctx context.Context) (err error) {
		ctx, log := WithGroup(ctx, "chdir")
		cwd, err := Getwd()
		if err != nil {
			return log.WrapErr(fmt.Errorf("failed to get current working directory: %w", err))
		}
		if err = newWorkdir.Chdir(); err != nil {
			return log.WrapErr(fmt.Errorf("failed to change the working directory to '%s': %w", newWorkdir, err))
		}
		defer func() {
			if _err := cwd.Chdir(); _err != nil {
				_err = fmt.Errorf("failed to reset the working directory to '%s': %w", cwd, _err)
				if err != nil {
					err = log.WrapErr(errors.Join(err, _err))
				}
			}
		}()
		return runner.Run(ctx)
	}
}

// CopyFile creates a Runner that copies a source file to target.
// The source and target file names are expected to be relative to the build's working directory, unless they are absolute paths.
// The target file will be created or truncated as appropriate.
func CopyFile(source, target PathString) Task {
	return func(ctx context.Context) error {
		_, log := WithGroup(ctx, "copy file")
		return log.WrapErr(source.CopyTo(target))
	}
}

// MoveFile creates a Runner that will move the file indicated by source to target.
// The source and target file names are expected to be relative to the build's working directory, unless they are absolute paths.
// The target file will be created or truncated as appropriate.
func MoveFile(source, target PathString) Task {
	return func(ctx context.Context) error {
		return CopyFile(source, target).Then(Task(func(ctx context.Context) error {
			_, log := WithGroup(ctx, "remove file")
			return log.WrapErr(source.Remove())
		})).Run(ctx)
	}
}

// Mkdir makes a directory named dir, if it doesn't exist already.
// If the directory already exists, then nothing is done and err will be nil.
// Any error encountered while making the directory is returned.
func Mkdir(dir PathString, perm os.FileMode) Task {
	return func(ctx context.Context) error {
		_, log := WithGroup(ctx, "mkdir")
		if dir.Exists() {
			return nil
		}
		err := dir.Mkdir(perm)
		if err != nil {
			if errors.Is(err, fs.ErrExist) {
				return nil
			}
			return log.WrapErr(err)
		}
		return nil
	}
}

// MkdirAll makes the target directory, and any directories in between.
// Any error encountered while making the directories is returned.
func MkdirAll(dir PathString, perm os.FileMode) Task {
	return func(ctx context.Context) error {
		_, log := WithGroup(ctx, "mkdirall")
		return log.WrapErr(dir.MkdirAll(perm))
	}
}

// RemoveFile will create a Runner that removes the specified file from the filesystem.
// If the file doesn't exist, then this Runner returns nil.
// Any other error encountered while removing the file is returned.
func RemoveFile(file PathString) Task {
	return func(ctx context.Context) error {
		_, log := WithGroup(ctx, "removefile")
		if !file.Exists() {
			return nil
		}
		if file.IsDir() {
			return log.WrapErr(fmt.Errorf("file '%s' is a directory, use RemoveDir instead", file))
		}
		return log.WrapErr(file.Remove())
	}
}

// RemoveDir will create a Runner that removes the directory specified and all of its contents from the filesystem.
// If the directory doesn't exist, then this Runner returns nil.
// Any other error encountered while removing the directory is returned.
func RemoveDir(file PathString) Task {
	return func(ctx context.Context) error {
		_, log := WithGroup(ctx, "removedir")
		if !file.Exists() {
			return nil
		}
		if file.IsFile() {
			return log.WrapErr(fmt.Errorf("file '%s' is a file, use RemoveFile instead", file))
		}
		return log.WrapErr(file.RemoveAll())
	}
}
