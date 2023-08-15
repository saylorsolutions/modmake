package modmake

import (
	"context"
	"github.com/bitfield/script"
	"os"
	"path/filepath"
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

// SkipIfExists will skip executing the Runner if the given file exists.
// This is similar to the default Make behavior of skipping a task if the target file already exists.
func SkipIfExists(file string, r Runner) Runner {
	return RunnerFunc(func(ctx context.Context) error {
		_, err := os.Stat(file)
		if err != nil {
			if err == os.ErrNotExist {
				return r.Run(ctx)
			}
			return err
		}
		return nil
	})
}

// CopyFile creates a Runner that copies a source file to target.
// The source and target file names are expected to be relative to the build's working directory, unless they are absolute paths.
// The target file will be created or truncated as appropriate.
func CopyFile(source, target string) Runner {
	return RunnerFunc(func(ctx context.Context) error {
		val := ctx.Value(CtxWorkdir)
		workdir := val.(string)
		if !filepath.IsAbs(source) {
			source = filepath.Join(workdir, source)
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(workdir, target)
		}
		_, err := script.File(source).WriteFile(target)
		return err
	})
}

// MoveFile creates a Runner that will move the file indicated by source to target.
// The source and target file names are expected to be relative to the build's working directory, unless they are absolute paths.
// The target file will be created or truncated as appropriate.
func MoveFile(source, target string) Runner {
	return RunnerFunc(func(ctx context.Context) error {
		val := ctx.Value(CtxWorkdir)
		workdir := val.(string)
		if !filepath.IsAbs(target) {
			target = filepath.Join(workdir, target)
		}
		if err := CopyFile(source, target).Run(ctx); err != nil {
			return err
		}
		return os.Remove(target)
	})
}
