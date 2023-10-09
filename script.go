package modmake

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strings"
)

// Script will execute each Task in order, returning the first error.
func Script(fns ...Runner) Runner {
	return Task(func(ctx context.Context) error {
		for i, fn := range fns {
			run := ContextAware(fn)
			if err := run.Run(ctx); err != nil {
				return fmt.Errorf("script[%d]: %w", i, err)
			}
		}
		return nil
	})
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

// Print will create a Runner that logs a message.
func Print(msg string, args ...any) Task {
	return Plain(func() {
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		log.Printf(msg, args...)
	})
}

// Chdir will change the current working directory to newWorkdir and run the Runner in that context.
// Whether the Runner executes successfully or not, the working directory will be reset back to its original state.
func Chdir(newWorkdir PathString, runner Runner) Task {
	return func(ctx context.Context) (err error) {
		cwd, err := Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
		if err = newWorkdir.Chdir(); err != nil {
			return fmt.Errorf("failed to change the working directory to '%s': %w", newWorkdir, err)
		}
		defer func() {
			if _err := cwd.Chdir(); _err != nil {
				_err = fmt.Errorf("failed to reset the working directory to '%s': %w", cwd, _err)
				if err != nil {
					err = fmt.Errorf("%w, %v", err, _err)
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
	return WithoutContext(func() error {
		return source.CopyTo(target)
	})
}

// MoveFile creates a Runner that will move the file indicated by source to target.
// The source and target file names are expected to be relative to the build's working directory, unless they are absolute paths.
// The target file will be created or truncated as appropriate.
func MoveFile(source, target PathString) Task {
	return func(ctx context.Context) error {
		return CopyFile(source, target).Then(Task(func(_ context.Context) error {
			return source.Remove()
		})).Run(ctx)
	}
}

// Mkdir makes a directory named dir, if it doesn't exist already.
// If the directory already exists, then nothing is done and err will be nil.
// Any error encountered while making the directory is returned.
func Mkdir(dir PathString, perm os.FileMode) Task {
	return WithoutContext(func() error {
		if dir.Exists() {
			return nil
		}
		err := dir.Mkdir(perm)
		if err != nil {
			if errors.Is(err, fs.ErrExist) {
				return nil
			}
			return err
		}
		return nil
	})
}

// MkdirAll makes the target directory, and any directories in between.
// Any error encountered while making the directories is returned.
func MkdirAll(dir PathString, perm os.FileMode) Task {
	return WithoutContext(func() error {
		return dir.MkdirAll(perm)
	})
}

// RemoveFile will create a Runner that removes the specified file from the filesystem.
// If the file doesn't exist, then this Runner returns nil.
// Any other error encountered while removing the file is returned.
func RemoveFile(file PathString) Task {
	return WithoutContext(func() error {
		if !file.Exists() {
			return nil
		}
		if file.IsDir() {
			return fmt.Errorf("file '%s' is a directory, use RemoveDir instead", file)
		}
		return file.Remove()
	})
}

// RemoveDir will create a Runner that removes the directory specified and all of its contents from the filesystem.
// If the directory doesn't exist, then this Runner returns nil.
// Any other error encountered while removing the directory is returned.
func RemoveDir(file PathString) Task {
	return WithoutContext(func() error {
		if !file.Exists() {
			return nil
		}
		if file.IsFile() {
			return fmt.Errorf("file '%s' is a file, use RemoveFile instead", file)
		}
		return file.RemoveAll()
	})
}
