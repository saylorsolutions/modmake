package modmake

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
)

// TarArchive represents a Runner that performs operations on a tar archive that uses gzip compression.
// The Runner is created with Tar.
type TarArchive struct {
	err      error
	path     PathString
	addFiles map[PathString]string
}

// Tar will create a new TarArchive to contextualize follow-on operations that act on a tar.gz file.
// Adding a ".tar.gz" suffix to the location path string is recommended, but not required.
func Tar(location PathString) *TarArchive {
	if len(location) == 0 {
		panic("empty location")
	}
	return &TarArchive{
		path:     location,
		addFiles: map[PathString]string{},
	}
}

// AddFile adds the referenced file with the same archive path as what is given.
// The archive path will be converted to slash format.
func (t *TarArchive) AddFile(sourcePath PathString) *TarArchive {
	t.AddFileWithPath(sourcePath, sourcePath)
	return t
}

// AddFileWithPath adds the referenced file with an archive path specified.
// The archive path will be converted to slash format.
func (t *TarArchive) AddFileWithPath(sourcePath, archivePath PathString) *TarArchive {
	if t.err != nil {
		return t
	}
	if len(sourcePath) == 0 {
		panic("empty source path")
	}
	if len(archivePath) == 0 {
		panic("empty target path")
	}
	archivePathStr := archivePath.ToSlash()
	t.addFiles[sourcePath] = archivePathStr
	return t
}

// Create will return a Runner that creates a new tar file with the given files loaded.
// If a file with the given name already exists, then it will be truncated first.
// Ensure that all files referenced with AddFile (or AddFileWithPath) and directories exist before running this Runner, because it doesn't try to create them.
func (t *TarArchive) Create() Task {
	runner := Task(func(ctx context.Context) error {
		ctx, log := WithGroup(ctx, "tar create")
		tarFile, err := t.path.Create()
		if err != nil {
			return log.WrapErr(err)
		}
		defer func() {
			_ = tarFile.Close()
		}()
		gz := gzip.NewWriter(tarFile)
		defer func() {
			_ = gz.Close()
		}()
		tw := tar.NewWriter(gz)
		defer func() {
			_ = tw.Close()
		}()
		err = t.writeFilesToTarArchive(ctx, tw)
		if err != nil {
			return log.WrapErr(err)
		}
		return nil
	})
	return ContextAware(runner)
}

// Update will return a Runner that creates a new tar file with the given files loaded.
// If a file with the given name already exists, then it will be updated.
// If a file with the given name does not exist, then this Runner will return an error.
// Ensure that all files referenced with AddFile (or AddFileWithPath) and directories exist before running this Runner, because it doesn't try to create them.
func (t *TarArchive) Update() Task {
	runner := Task(func(ctx context.Context) error {
		ctx, log := WithGroup(ctx, "tar update")
		tarFile, err := t.path.OpenFile(os.O_RDWR, 0600)
		if err != nil {
			return log.WrapErr(err)
		}
		defer func() {
			_ = tarFile.Close()
		}()
		gz := gzip.NewWriter(tarFile)
		defer func() {
			_ = gz.Close()
		}()
		tw := tar.NewWriter(gz)
		defer func() {
			_ = tw.Close()
		}()
		err = t.writeFilesToTarArchive(ctx, tw)
		if err != nil {
			return log.WrapErr(err)
		}
		return nil
	})
	return ContextAware(runner)
}

func (t *TarArchive) writeFilesToTarArchive(ctx context.Context, tw *tar.Writer) error {
	ctx, log := WithGroup(ctx, "write files")
	for source, target := range t.addFiles {
		if err := ctx.Err(); err != nil {
			return log.WrapErr(err)
		}
		source, target := source, target
		err := func() error {
			if !source.Exists() {
				return fmt.Errorf("unable to locate source file: %s", source)
			}
			f, err := source.Open()
			if err != nil {
				return err
			}
			defer func() {
				_ = f.Close()
			}()
			fi, err := f.Stat()
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(fi, target)
			if err != nil {
				return fmt.Errorf("failed to get file info for '%s': %w", source, err)
			}
			if err := tw.WriteHeader(header); err != nil {
				return err
			}
			_, err = io.Copy(tw, f)
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

// Extract will extract the named tar archive to the given directory.
// Any errors encountered while doing so will be immediately returned.
func (t *TarArchive) Extract(extractDir PathString) Task {
	runner := Task(func(ctx context.Context) error {
		ctx, log := WithGroup(ctx, "tar extract")
		src, err := t.path.Open()
		if err != nil {
			return log.WrapErr(fmt.Errorf("unable to open tar archive: %w", err))
		}
		defer func() {
			_ = src.Close()
		}()
		err = extractDir.MkdirAll(0700)
		if err != nil {
			return log.WrapErr(fmt.Errorf("unable to create extraction directory: %w", err))
		}
		gz, err := gzip.NewReader(src)
		if err != nil {
			return log.WrapErr(fmt.Errorf("unable to create gzip reader for '%s': %w", t.path, err))
		}
		tr := tar.NewReader(gz)
		for {
			header, err := tr.Next()
			if err == io.EOF {
				// End of archive
				break
			}
			if err != nil {
				return err
			}
			if header.FileInfo().IsDir() {
				continue
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			err = func() error {
				output := extractDir.Join(header.Name)
				outputDir := output.Dir()
				if err := outputDir.MkdirAll(0700); err != nil {
					return fmt.Errorf("failed to make parent directory for file '%s' at '%s': %w", header.Name, outputDir, err)
				}
				out, err := output.Create()
				if err != nil {
					return fmt.Errorf("failed to create file '%s': %w", output, err)
				}
				defer func() {
					_ = out.Close()
				}()
				_, err = io.CopyN(out, tr, header.Size)
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
