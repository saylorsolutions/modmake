package modmake

import (
	"os"
	"path/filepath"
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
