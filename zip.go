package modmake

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ZipArchive represents a Runner that performs operations on a tar archive that uses gzip compression.
// The Runner is created with Zip.
type ZipArchive struct {
	err      error
	path     string
	addFiles map[string]string
}

// Zip will create a new ZipArchive to contextualize follow-on operations that act on a zip file.
// Adding a ".zip" suffix to the location path string is recommended, but not required.
func Zip(location string) *ZipArchive {
	location = strings.TrimSpace(location)
	if len(location) == 0 {
		panic("empty location")
	}
	return &ZipArchive{
		path:     location,
		addFiles: map[string]string{},
	}
}

// AddFile adds the referenced file with the same archive path as what is given.
// The archive path will be converted to slash format.
func (z *ZipArchive) AddFile(sourcePath string) *ZipArchive {
	z.AddFileWithPath(sourcePath, sourcePath)
	return z
}

// AddFileWithPath adds the referenced file with an archive path specified.
// The archive path will be converted to slash format.
func (z *ZipArchive) AddFileWithPath(sourcePath, archivePath string) *ZipArchive {
	if z.err != nil {
		return z
	}
	sourcePath, archivePath = strings.TrimSpace(sourcePath), strings.TrimSpace(archivePath)
	if len(sourcePath) == 0 {
		panic("empty source path")
	}
	if len(archivePath) == 0 {
		panic("empty target path")
	}
	z.addFiles[sourcePath] = filepath.ToSlash(archivePath)
	return z
}

// Create will return a Runner that creates a new zip file with the given files loaded.
// If a file with the given name already exists, then it will be truncated first.
// Ensure that all files referenced with AddFile (or AddFileWithPath) and directories exist before running this Runner, because it doesn't try to create them.
func (z *ZipArchive) Create() Runner {
	runner := RunFunc(func(ctx context.Context) error {
		zipFile, err := os.Create(z.path)
		if err != nil {
			return err
		}
		defer func() {
			_ = zipFile.Close()
		}()
		zw := zip.NewWriter(zipFile)
		defer func() {
			_ = zw.Close()
		}()
		if err := z.writeFilesToZipArchive(ctx, zw); err != nil {
			return err
		}
		return nil
	})
	return ContextAware(runner)
}

// Update will return a Runner that creates a new zip file with the given files loaded.
// If a file with the given name already exists, then it will be updated.
// If a file with the given name does not exist, then this Runner will return an error.
// Ensure that all files referenced with AddFile (or AddFileWithPath) and directories exist before running this Runner, because it doesn't try to create them.
func (z *ZipArchive) Update() Runner {
	runner := RunFunc(func(ctx context.Context) error {
		zipFile, err := os.OpenFile(z.path, os.O_RDWR, 0644)
		if err != nil {
			return err
		}
		defer func() {
			_ = zipFile.Close()
		}()
		zw := zip.NewWriter(zipFile)
		defer func() {
			_ = zw.Close()
		}()
		if err := z.writeFilesToZipArchive(ctx, zw); err != nil {
			return err
		}
		return nil
	})
	return ContextAware(runner)
}

func (z *ZipArchive) writeFilesToZipArchive(ctx context.Context, zw *zip.Writer) error {
	for source, target := range z.addFiles {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			source, target := source, target
			err := func() error {
				f, err := os.Open(source)
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
				return err
			}
		}
	}
	return nil
}

// Extract will extract the named zip archive to the given directory.
// Any errors encountered while doing so will be immediately returned.
func (z *ZipArchive) Extract(extractDir string) Runner {
	runner := RunFunc(func(ctx context.Context) error {
		extractDir = filepath.Clean(extractDir)
		err := os.MkdirAll(extractDir, 0755)
		if err != nil {
			return fmt.Errorf("unable to create extraction directory: %w", err)
		}
		fi, err := os.Stat(z.path)
		if err != nil {
			return fmt.Errorf("unable to get file information for the source zip file: %w", err)
		}
		src, err := os.Open(z.path)
		if err != nil {
			return fmt.Errorf("unable to open zip archive: %w", err)
		}
		defer func() {
			_ = src.Close()
		}()
		zr, err := zip.NewReader(src, fi.Size())
		if err != nil {
			return fmt.Errorf("failed to open '%s' for reading: %w", z.path, err)
		}
		for _, f := range zr.File {
			f := f
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				err := func() error {
					output := filepath.Join(extractDir, f.Name)
					if strings.HasSuffix(output, "/") {
						err := os.MkdirAll(output, 0755)
						if err != nil {
							return fmt.Errorf("failed to create parent directory '%s': %w", output, err)
						}
						return nil
					}
					outputDir := filepath.Dir(output)
					if err := os.MkdirAll(outputDir, 0755); err != nil {
						return fmt.Errorf("failed to make parent directory for file '%s' at '%s': %w", f.Name, outputDir, err)
					}
					zipFile, err := f.Open()
					if err != nil {
						return fmt.Errorf("failed to open compressed file '%s' for reading: %w", f.Name, err)
					}
					defer func() {
						_ = zipFile.Close()
					}()
					out, err := os.Create(output)
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
					return err
				}
			}
		}
		return nil
	})
	return ContextAware(runner)
}
