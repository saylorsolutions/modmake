package modmake

import (
	"context"
	"errors"
	"fmt"
	"github.com/bitfield/script"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Script will execute each RunFunc in order, returning the first error.
func Script(fns ...RunFunc) Runner {
	return RunFunc(func(ctx context.Context) error {
		for _, fn := range fns {
			run := ContextAware(fn)
			if err := run.Run(ctx); err != nil {
				return err
			}
		}
		return nil
	})
}

// IfNotExists will skip executing the Runner if the given file exists, returning nil.
// This is similar to the default Make behavior of skipping a task if the target file already exists.
func IfNotExists(file string, r Runner) Runner {
	return RunFunc(func(ctx context.Context) error {
		if err := script.IfExists(file).Error(); err != nil {
			if err == os.ErrNotExist {
				return r.Run(ctx)
			}
			return err
		}
		return nil
	})
}

// IfExists will execute the Runner if the file exists, returning nil otherwise.
func IfExists(file string, r Runner) Runner {
	return RunFunc(func(ctx context.Context) error {
		if err := script.IfExists(file).Error(); err == nil {
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

// CopyFile creates a Runner that copies a source file to target.
// The source and target file names are expected to be relative to the build's working directory, unless they are absolute paths.
// The target file will be created or truncated as appropriate.
func CopyFile(source, target string) Runner {
	return RunFunc(func(ctx context.Context) error {
		workdir, err := getWorkdir(ctx)
		if err != nil {
			return err
		}
		source = relativeToWorkdir(workdir, source)
		target = relativeToWorkdir(workdir, target)
		_, err = script.File(source).WriteFile(target)
		return err
	})
}

// MoveFile creates a Runner that will move the file indicated by source to target.
// The source and target file names are expected to be relative to the build's working directory, unless they are absolute paths.
// The target file will be created or truncated as appropriate.
func MoveFile(source, target string) Runner {
	return RunFunc(func(ctx context.Context) error {
		workdir, err := getWorkdir(ctx)
		if err != nil {
			return err
		}
		source = relativeToWorkdir(workdir, source)
		target = relativeToWorkdir(workdir, target)
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

// Remove will remove the file specified from the filesystem.
// If the file doesn't exist, then this returns nil.
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

// RemoveDir will remove the directory specified from the filesystem.
// If the directory doesn't exist, then this returns nil.
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

// getWorkdir will return the working directory value from the context, if it exists.
// If the type of CtxWorkdir is not a string, or if it doesn't exist, then ErrMissingWorkdir will be returned with an empty string.
func getWorkdir(ctx context.Context) (string, error) {
	val := ctx.Value(CtxWorkdir)
	if val == nil {
		return "", ErrMissingWorkdir
	}
	str, ok := val.(string)
	if !ok {
		return "", ErrMissingWorkdir
	}
	return str, nil
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
