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
func IfNotExists(file string, r Runner) Task {
	return func(ctx context.Context) error {
		ok, err := fileExists(file)
		if err != nil {
			return err
		}
		if !ok {
			return r.Run(ctx)
		}
		return nil
	}
}

// IfExists will execute the Runner if the file exists, returning nil otherwise.
func IfExists(file string, r Runner) Task {
	return func(ctx context.Context) error {
		ok, err := fileExists(file)
		if err != nil {
			return err
		}
		if ok {
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
func Chdir(newWorkdir string, runner Runner) Task {
	return func(ctx context.Context) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
		if err := os.Chdir(newWorkdir); err != nil {
			return fmt.Errorf("failed to change the working directory to '%s': %w", newWorkdir, err)
		}
		err = runner.Run(ctx)
		if err := os.Chdir(cwd); err != nil {
			return fmt.Errorf("failed to reset the working directory to '%s': %w", cwd, err)
		}
		return err
	}
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
func CopyFile(source, target string) Task {
	return WithoutContext(func() error {
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
func MoveFile(source, target string) Task {
	return func(ctx context.Context) error {
		workdir, err := os.Getwd()
		if err != nil {
			return err
		}

		source = relativeToWorkdir(workdir, source)
		if err := CopyFile(source, target).Run(ctx); err != nil {
			return err
		}
		return os.Remove(source)
	}
}

// Mkdir makes a directory named dir, if it doesn't exist already.
// If the directory already exists, then nothing is done and err will be nil.
// Any error encountered while making the directory is returned.
func Mkdir(dir string, perm os.FileMode) Task {
	return WithoutContext(func() error {
		err := os.Mkdir(dir, perm)
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
func MkdirAll(dir string, perm os.FileMode) Task {
	return WithoutContext(func() error {
		return os.MkdirAll(dir, perm)
	})
}

// RemoveFile will create a Runner that removes the specified file from the filesystem.
// If the file doesn't exist, then this Runner returns nil.
// Any other error encountered while removing the file is returned.
func RemoveFile(file string) Task {
	return WithoutContext(func() error {
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
func RemoveDir(file string) Task {
	return WithoutContext(func() error {
		fi, err := os.Stat(file)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}
			return err
		}
		if !fi.IsDir() {
			return fmt.Errorf("file '%s' is a file, use RemoveFile instead", file)
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

// PathString is a string that represents a filesystem path.
//
// String inputs to all PathString functions, including Path, is expected to be a filesystem path with forward slash ('/') separators.
// These will be translated to actual OS filesystem path strings, making them incompatible with module path strings on Windows.
type PathString string

// Join will append path segments to this PathString and return a new PathString.
func (p PathString) Join(segments ...string) PathString {
	if len(segments) == 0 {
		return p
	}
	_segments := make([]string, len(segments))
	for i, segment := range segments {
		_segments[i] = filepath.FromSlash(segment)
	}
	return PathString(filepath.Join(append([]string{string(p)}, _segments...)...))
}

// JoinPath will append PathString segments to this PathString and return a new PathString.
func (p PathString) JoinPath(segments ...PathString) PathString {
	var newPath = p
	for _, segment := range segments {
		newPath = newPath.Join(string(segment))
	}
	return newPath
}

func (p PathString) Base() string {
	return filepath.Base(string(p))
}

func (p PathString) Dir() PathString {
	return Path(filepath.Dir(string(p)))
}

func (p PathString) String() string {
	return string(p)
}

// Path creates a new PathString from the input path segments.
func Path(path string, segments ...string) PathString {
	path = filepath.FromSlash(path)
	if len(segments) == 0 {
		return PathString(path)
	}
	_segments := make([]string, len(segments))
	for i, segment := range segments {
		_segments[i] = filepath.FromSlash(segment)
	}
	return PathString(filepath.Join(append([]string{path}, _segments...)...))
}
