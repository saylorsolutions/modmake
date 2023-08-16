package modmake

import (
	"context"
	"github.com/bitfield/script"
	"os"
	"path/filepath"
	"strings"
)

// Exec wraps execution of all commands in a Runner that executes them in a sequence, and returns the first error that occurs.
func Exec(cmds ...RunnerFunc) Runner {
	return RunnerFunc(func(ctx context.Context) error {
		for _, cmd := range cmds {
			err := cmd(ctx)
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// IfNotExists will skip executing the Runner if the given file exists, returning nil.
// This is similar to the default Make behavior of skipping a task if the target file already exists.
func IfNotExists(file string, r Runner) Runner {
	return RunnerFunc(func(ctx context.Context) error {
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
	return RunnerFunc(func(ctx context.Context) error {
		if err := script.IfExists(file).Error(); err == nil {
			return r.Run(ctx)
		}
		return nil
	})
}

// CopyFile creates a Runner that copies a source file to target.
// The source and target file names are expected to be relative to the build's working directory, unless they are absolute paths.
// The target file will be created or truncated as appropriate.
func CopyFile(source, target string) Runner {
	return RunnerFunc(func(ctx context.Context) error {
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
	return RunnerFunc(func(ctx context.Context) error {
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
