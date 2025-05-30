package modmake

import (
	"errors"
	"io"
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
		newPath = newPath.Join(segment.ToSlash())
	}
	return newPath
}

// ToSlash will change the PathString to use slash separators if the OS representation is different.
func (p PathString) ToSlash() string {
	return filepath.ToSlash(string(p))
}

// Base calls filepath.Base on the string representation of this PathString, returning the last element of the path.
// Trailing path separators are removed before extracting the last element.
// If the path is empty, Base returns ".".
// If the path consists entirely of separators, Base returns a single separator.
func (p PathString) Base() PathString {
	return PathString(filepath.Base(string(p)))
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

// Abs attempts to translate the PathString into an absolute path using filepath.Abs.
func (p PathString) Abs() (PathString, error) {
	abs, err := filepath.Abs(p.String())
	return Path(abs), err
}

// Rel attempts to construct a relative path to other, with the current PathString as the base, much like filepath.Rel.
func (p PathString) Rel(other PathString) (PathString, error) {
	a, err := p.Abs()
	if err != nil {
		return "", err
	}
	b, err := other.Abs()
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(a.String(), b.String())
	if err != nil {
		return "", err
	}
	return Path(rel), nil
}

// Exists returns true if the path references an existing file or directory.
func (p PathString) Exists() bool {
	_, err := os.Stat(string(p))
	return err == nil
}

// Stat will return os.FileInfo for the file referenced by this PathString like os.Stat.
func (p PathString) Stat() (os.FileInfo, error) {
	return os.Stat(string(p))
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

// Create will create or truncate the named file like os.Create.
func (p PathString) Create() (*os.File, error) {
	return os.Create(string(p))
}

// Open opens the named file for reading like os.Open.
func (p PathString) Open() (*os.File, error) {
	return os.Open(string(p))
}

// OpenFile is a generalized open call like os.OpenFile.
func (p PathString) OpenFile(flag int, mode os.FileMode) (*os.File, error) {
	return os.OpenFile(string(p), flag, mode) //nolint:gosec // This is specifically intended to allow opening a user-defined file.
}

// Mkdir creates a new directory with the specified name and permissions like os.Mkdir.
func (p PathString) Mkdir(mode os.FileMode) error {
	return os.Mkdir(string(p), mode)
}

// MkdirAll creates the named directory and any non-existent path in between like os.MkdirAll.
func (p PathString) MkdirAll(mode os.FileMode) error {
	return os.MkdirAll(string(p), mode)
}

// Remove removes the named file or directory like os.Remove.
func (p PathString) Remove() error {
	return os.Remove(string(p))
}

// RemoveAll removes the path and any children it contains like os.RemoveAll.
func (p PathString) RemoveAll() error {
	return os.RemoveAll(string(p))
}

// Chdir changes the current working directory to the named directory like os.Chdir.
func (p PathString) Chdir() error {
	return os.Chdir(string(p))
}

// CopyTo copies the contents of the file referenced by this PathString to the file referenced by other, creating or truncating the file.
func (p PathString) CopyTo(other PathString) error {
	in, err := p.Open()
	if err != nil {
		return err
	}
	defer func() {
		_ = in.Close()
	}()
	out, err := other.Create()
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return nil
}

// Cat will - assuming the PathString points to a file - read all data from the file and return it as a byte slice.
func (p PathString) Cat() ([]byte, error) {
	if p.IsDir() {
		return nil, errors.New("path references a directory")
	}
	f, err := p.Open()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (p PathString) ReadFile() ([]byte, error) {
	return os.ReadFile(p.String())
}

func (p PathString) WriteFile(data []byte, perm os.FileMode) error {
	return os.WriteFile(p.String(), data, perm)
}

// Ext returns the file name extension used by path.
// The extension is the suffix beginning at the final dot in the final element of path; it is empty if there is no dot.
func (p PathString) Ext() string {
	return filepath.Ext(string(p))
}

// Getwd gets the current working directory as a PathString like os.Getwd.
func Getwd() (PathString, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return Path(cwd), nil
}

// UserHomeDir will return the current user's home directory as a PathString.
func UserHomeDir() (PathString, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return Path(home), nil
}
