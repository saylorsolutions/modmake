package modmake

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
)

// ZipArchive represents a Runner that performs operations on a tar archive that uses gzip compression.
// The Runner is created with Zip.
type ZipArchive struct {
	err      error
	path     PathString
	addFiles map[PathString]string
}

// Zip will create a new ZipArchive to contextualize follow-on operations that act on a zip file.
// Adding a ".zip" suffix to the location path string is recommended, but not required.
func Zip(location PathString) *ZipArchive {
	if len(location) == 0 {
		panic("empty location")
	}
	return &ZipArchive{
		path:     location,
		addFiles: map[PathString]string{},
	}
}

// AddFile adds the referenced file with the same archive path as what is given.
// The archive path will be converted to slash format.
func (z *ZipArchive) AddFile(sourcePath PathString) *ZipArchive {
	z.AddFileWithPath(sourcePath, sourcePath)
	return z
}

// AddFileWithPath adds the referenced file with an archive path specified.
// The archive path will be converted to slash format.
func (z *ZipArchive) AddFileWithPath(sourcePath, archivePath PathString) *ZipArchive {
	if z.err != nil {
		return z
	}
	if len(sourcePath) == 0 {
		panic("empty source path")
	}
	if len(archivePath) == 0 {
		panic("empty target path")
	}
	z.addFiles[sourcePath] = archivePath.ToSlash()
	return z
}

// Create will return a Runner that creates a new zip file with the given files loaded.
// If a file with the given name already exists, then it will be truncated first.
// Ensure that all files referenced with AddFile (or AddFileWithPath) and directories exist before running this Runner, because it doesn't try to create them.
func (z *ZipArchive) Create() Task {
	runner := Task(func(ctx context.Context) error {
		ctx, log := WithGroup(ctx, "zip create")
		zipFile, err := z.path.Create()
		if err != nil {
			return log.WrapErr(err)
		}
		defer func() {
			_ = zipFile.Close()
		}()
		zw := zip.NewWriter(zipFile)
		defer func() {
			_ = zw.Close()
		}()
		if err := z.writeFilesToZipArchive(ctx, zw); err != nil {
			return log.WrapErr(err)
		}
		return nil
	})
	return ContextAware(runner)
}

// Update will return a Runner that creates a new zip file with the given files loaded.
// If a file with the given name already exists, then it will be updated.
// If a file with the given name does not exist, then this Runner will return an error.
// Ensure that all files referenced with AddFile (or AddFileWithPath) and directories exist before running this Runner, because it doesn't try to create them.
func (z *ZipArchive) Update() Task {
	runner := Task(func(ctx context.Context) error {
		ctx, log := WithGroup(ctx, "zip update")
		zipFile, err := z.path.OpenFile(os.O_RDWR, 0644)
		if err != nil {
			return log.WrapErr(err)
		}
		defer func() {
			_ = zipFile.Close()
		}()
		zw := zip.NewWriter(zipFile)
		defer func() {
			_ = zw.Close()
		}()
		if err := z.writeFilesToZipArchive(ctx, zw); err != nil {
			return log.WrapErr(err)
		}
		return nil
	})
	return ContextAware(runner)
}

func (z *ZipArchive) writeFilesToZipArchive(ctx context.Context, zw *zip.Writer) error {
	ctx, log := WithGroup(ctx, "write files")
	for source, target := range z.addFiles {
		if err := ctx.Err(); err != nil {
			return log.WrapErr(err)
		}
		source, target := source, target
		err := func() error {
			f, err := source.Open()
			if err != nil {
				return err
			}
			defer func() {
				_ = f.Close()
			}()
			t, err := zw.Create(target)
			if err != nil {
				return err
			}
			_, err = io.Copy(t, f)
			if err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			return log.WrapErr(err)
		}
	}
	return nil
}

// Extract will extract the named zip archive to the given directory.
// Any errors encountered while doing so will be immediately returned.
func (z *ZipArchive) Extract(extractDir PathString) Task {
	runner := Task(func(ctx context.Context) error {
		ctx, log := WithGroup(ctx, "zip extract")
		src, err := z.path.Open()
		if err != nil {
			return log.WrapErr(fmt.Errorf("unable to open zip archive: %w", err))
		}
		defer func() {
			_ = src.Close()
		}()
		fi, err := z.path.Stat()
		if err != nil {
			return log.WrapErr(fmt.Errorf("unable to get file information for the source zip file: %w", err))
		}
		err = extractDir.MkdirAll(0755)
		if err != nil {
			return log.WrapErr(fmt.Errorf("unable to create extraction directory: %w", err))
		}
		zr, err := zip.NewReader(src, fi.Size())
		if err != nil {
			return log.WrapErr(fmt.Errorf("failed to open '%s' for reading: %w", z.path, err))
		}
		for _, f := range zr.File {
			f := f
			if err := ctx.Err(); err != nil {
				return err
			}
			err := func() error {
				output := extractDir.Join(f.Name)
				if strings.HasSuffix(output.String(), "/") {
					err := output.MkdirAll(0755)
					if err != nil {
						return fmt.Errorf("failed to create parent directory '%s': %w", output, err)
					}
					return nil
				}
				outputDir := output.Dir()
				if err := outputDir.MkdirAll(0755); err != nil {
					return fmt.Errorf("failed to make parent directory for file '%s' at '%s': %w", f.Name, outputDir, err)
				}
				zipFile, err := f.Open()
				if err != nil {
					return fmt.Errorf("failed to open compressed file '%s' for reading: %w", f.Name, err)
				}
				defer func() {
					_ = zipFile.Close()
				}()
				out, err := output.Create()
				if err != nil {
					return fmt.Errorf("failed to create file '%s': %w", output, err)
				}
				defer func() {
					_ = out.Close()
				}()
				_, err = io.Copy(out, zipFile)
				if err != nil {
					return fmt.Errorf("failed to extract '%s': %w", output, err)
				}
				return nil
			}()
			if err != nil {
				return log.WrapErr(err)
			}
		}
		return nil
	})
	return ContextAware(runner)
}
