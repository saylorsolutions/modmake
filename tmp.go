package modmake

import (
	"fmt"
	"os"
	"path/filepath"
)

// TempDir will create a temporary directory at the system's default temp location with the provided pattern, like [os.MkdirTemp].
// The created [PathString] will be passed to the given function for use.
//
// The temp directory will be removed after this [Task] returns, or if the function panics.
func TempDir(pattern string, fn func(tmp PathString) Task) Task {
	return TempDirAt("", pattern, fn)
}

// TempDirAt will operate just like [TempDir], but allows specifying a location for the creation of the temp directory.
func TempDirAt(location, pattern string, fn func(tmp PathString) Task) Task {
	dir, err := os.MkdirTemp(location, pattern)
	if err != nil {
		panic(fmt.Sprintf("Failed to create temp directory with pattern '%s' and location '%s': %v", pattern, location, err))
	}
	dirPath := Path(filepath.ToSlash(dir))
	defer func() {
		if r := recover(); r != nil {
			_ = os.RemoveAll(dir)
			panic(fmt.Sprintf("Encountered panic in TempDir function using temp dir '%s': %v", dir, r))
		}
	}()
	return fn(dirPath).LogGroup("temp dir").Finally(func(_ error) error {
		return dirPath.RemoveAll()
	})
}
