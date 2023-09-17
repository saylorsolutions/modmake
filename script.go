package modmake

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Script will execute each RunFunc in order, returning the first error.
func Script(fns ...Runner) Runner {
	return RunFunc(func(ctx context.Context) error {
		for i, fn := range fns {
			run := ContextAware(fn)
			if err := run.Run(ctx); err != nil {
				return fmt.Errorf("script[%d]: %w", i, err)
			}
		}
		return nil
	})
}

func fileExists(file string) (bool, error) {
	_, err := os.Stat(file)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// IfNotExists will skip executing the Runner if the given file exists, returning nil.
// This is similar to the default Make behavior of skipping a task if the target file already exists.
func IfNotExists(file string, r Runner) Runner {
	return RunFunc(func(ctx context.Context) error {
		ok, err := fileExists(file)
		if err != nil {
			return err
		}
		if !ok {
			return r.Run(ctx)
		}
		return nil
	})
}

// IfExists will execute the Runner if the file exists, returning nil otherwise.
func IfExists(file string, r Runner) Runner {
	return RunFunc(func(ctx context.Context) error {
		ok, err := fileExists(file)
		if err != nil {
			return err
		}
		if ok {
			return r.Run(ctx)
		}
		return nil
	})
}

// IfError will create a Runner with an error handler Runner that is only executed if the base Runner returns an error.
// If both the base Runner and the handler return an error, then the handler's error will be returned.
func IfError(canErr Runner, handler Runner) Runner {
	return RunFunc(func(ctx context.Context) error {
		if err := canErr.Run(ctx); err != nil {
			return handler.Run(ctx)
		}
		return nil
	})
}

// Print will create a Runner that logs a message.
func Print(msg string, args ...any) Runner {
	return RunFunc(func(ctx context.Context) error {
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		log.Printf(msg, args...)
		return nil
	})
}

// Chdir will change the current working directory to newWorkdir and run the Runner in that context.
// Whether the Runner executes successfully or not, the working directory will be reset back to its original state.
func Chdir(newWorkdir string, runner Runner) Runner {
	return RunFunc(func(ctx context.Context) error {
		curwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
		if err := os.Chdir(newWorkdir); err != nil {
			return fmt.Errorf("failed to change the working directory to '%s': %w", newWorkdir, err)
		}
		err = runner.Run(ctx)
		if err := os.Chdir(curwd); err != nil {
			return fmt.Errorf("failed to reset the working directory to '%s': %w", curwd, err)
		}
		return err
	})
}

func cpFile(source, target string) error {
	srcFile, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open source file '%s': %w", source, err)
	}
	defer func() {
		_ = srcFile.Close()
	}()

	trgFile, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("failed to open/create target file '%s': %w", target, err)
	}
	defer func() {
		_ = trgFile.Close()
	}()

	_, err = io.Copy(trgFile, srcFile)
	return err
}

// CopyFile creates a Runner that copies a source file to target.
// The source and target file names are expected to be relative to the build's working directory, unless they are absolute paths.
// The target file will be created or truncated as appropriate.
func CopyFile(source, target string) Runner {
	return RunFunc(func(ctx context.Context) error {
		workdir, err := os.Getwd()
		if err != nil {
			return err
		}
		return cpFile(relativeToWorkdir(workdir, source), relativeToWorkdir(workdir, target))
	})
}

// MoveFile creates a Runner that will move the file indicated by source to target.
// The source and target file names are expected to be relative to the build's working directory, unless they are absolute paths.
// The target file will be created or truncated as appropriate.
func MoveFile(source, target string) Runner {
	return RunFunc(func(ctx context.Context) error {
		workdir, err := os.Getwd()
		if err != nil {
			return err
		}

		source = relativeToWorkdir(workdir, source)
		if err := CopyFile(source, target).Run(ctx); err != nil {
			return err
		}
		return os.Remove(source)
	})
}

// Mkdir makes a directory named dir.
// Any error encountered while making the directory is returned.
func Mkdir(dir string, perm os.FileMode) Runner {
	return RunFunc(func(ctx context.Context) error {
		return os.Mkdir(dir, perm)
	})
}

// MkdirAll makes the target directory, and any directories in between.
// Any error encountered while making the directories is returned.
func MkdirAll(dir string, perm os.FileMode) Runner {
	return RunFunc(func(ctx context.Context) error {
		return os.MkdirAll(dir, perm)
	})
}

// Remove will create a Runner that removes the specified file from the filesystem.
// If the file doesn't exist, then this Runner returns nil.
// Any other error encountered while removing the file is returned.
func Remove(file string) Runner {
	return RunFunc(func(ctx context.Context) error {
		fi, err := os.Stat(file)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if fi.IsDir() {
			return fmt.Errorf("file '%s' is a directory, use RemoveDir instead", file)
		}
		return os.Remove(file)
	})
}

// RemoveDir will create a Runner that removes the directory specified and all of its contents from the filesystem.
// If the directory doesn't exist, then this Runner returns nil.
// Any other error encountered while removing the directory is returned.
func RemoveDir(file string) Runner {
	return RunFunc(func(ctx context.Context) error {
		fi, err := os.Stat(file)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if !fi.IsDir() {
			return fmt.Errorf("file '%s' is a file, use Remove instead", file)
		}
		return os.RemoveAll(file)
	})
}

// relativeToWorkdir will return a path relative to the workdir if file is relative.
// Otherwise, it will return the absolute path file.
func relativeToWorkdir(workdir, file string) string {
	file = strings.TrimSpace(file)
	if filepath.IsAbs(file) {
		return file
	}
	return filepath.Join(workdir, file)
}
