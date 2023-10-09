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

// PathString is a string that represents a filesystem path.
//
// String inputs to all PathString functions, including Path, is expected to be a filesystem path with forward slash ('/') separators.
// These will be translated to actual OS filesystem path strings, making them incompatible with module path strings on Windows.
type PathString string

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

// Base calls filepath.Base on the string representation of this PathString, returning the last element of the path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (p PathString) Base() string {
	return filepath.Base(string(p))
}

// Dir calls filepath.Dir on the string representation of this PathString, returning all but the last element of the path.
// After dropping the final element, Dir calls filepath.Clean on the path and trailing
// slashes are removed.
// If the path is empty, Dir returns ".".
// If the path consists entirely of separators, Dir returns a single separator.
// The returned path does not end in a separator unless it is the root directory.
func (p PathString) Dir() PathString {
	return Path(filepath.Dir(string(p)))
}

// Exists returns true if the path references an existing file or directory.
func (p PathString) Exists() bool {
	_, err := os.Stat(string(p))
	return err == nil
}

// IsDir returns true if this PathString references an existing directory.
func (p PathString) IsDir() bool {
	fi, err := os.Stat(string(p))
	if err != nil {
		return false
	}
	return fi.IsDir()
}

// IsFile returns true if this PathString references a file that exists, and it is not a directory.
func (p PathString) IsFile() bool {
	fi, err := os.Stat(string(p))
	if err != nil {
		return false
	}
	return !fi.IsDir()
}

// String returns this PathString as a string.
func (p PathString) String() string {
	return string(p)
}

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
	return func(ctx context.Context) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}
		if err := os.Chdir(newWorkdir.String()); err != nil {
			return fmt.Errorf("failed to change the working directory to '%s': %w", newWorkdir, err)
		}
		err = runner.Run(ctx)
		if err := os.Chdir(cwd); err != nil {
			return fmt.Errorf("failed to reset the working directory to '%s': %w", cwd, err)
		}
		return err
	}
}

func cpFile(source, target PathString) error {
	if !source.Exists() {
		return fmt.Errorf("source file does not exist: '%s'", source)
	}
	srcFile, err := os.Open(source.String())
	if err != nil {
		return fmt.Errorf("failed to open source file '%s': %w", source, err)
	}
	defer func() {
		_ = srcFile.Close()
	}()

	trgFile, err := os.Create(target.String())
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
func CopyFile(source, target PathString) Task {
	return WithoutContext(func() error {
		return cpFile(source, target)
	})
}

// MoveFile creates a Runner that will move the file indicated by source to target.
// The source and target file names are expected to be relative to the build's working directory, unless they are absolute paths.
// The target file will be created or truncated as appropriate.
func MoveFile(source, target PathString) Task {
	return func(ctx context.Context) error {
		return CopyFile(source, target).Then(Task(func(_ context.Context) error {
			return os.Remove(source.String())
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
		err := os.Mkdir(dir.String(), perm)
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
		return os.MkdirAll(dir.String(), perm)
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
		return os.Remove(file.String())
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
		return os.RemoveAll(file.String())
	})
}
