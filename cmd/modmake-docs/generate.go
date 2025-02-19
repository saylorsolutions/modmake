package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/saylorsolutions/modmake/cmd/modmake-docs/internal/templates"
	"github.com/saylorsolutions/modmake/cmd/modmake-docs/static"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

func doGenerate(params templates.Params) error {
	log.Println("Generating documentation...")
	var buf bytes.Buffer
	if err := templates.Main(params).Render(context.Background(), &buf); err != nil {
		return fmt.Errorf("failed to generate index.html: %w", err)
	}

	err := writeFile("index.html", "generated HTML", &buf)
	if err != nil {
		return err
	}
	err = writeFile("main.css", "CSS", bytes.NewReader(static.MainCSS))
	if err != nil {
		return err
	}

	err = fs.WalkDir(imgFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == "." {
			return nil
		}
		fileName := filepath.Join(".", path)
		if d.IsDir() {
			log.Println("Creating dir:", fileName)
			if err := os.MkdirAll(fileName, 0755); err != nil {
				return fmt.Errorf("failed to create directory '%s': %w", path, err)
			}
			return nil
		}
		data, err := fs.ReadFile(imgFS, path)
		if err != nil {
			return fmt.Errorf("failed to read file '%s' from embedded FS: %w", d.Name(), err)
		}
		log.Println("Creating file:", fileName)
		return writeFile(fileName, "image", bytes.NewReader(data))
	})
	if err != nil {
		return err
	}

	log.Println("Documentation generated successfully")
	return nil
}

func writeFile(filename, desc string, data io.Reader) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create '%s' in current directory: %w", filename, err)
	}
	defer func() {
		_ = f.Close()
	}()
	_, err = io.Copy(f, data)
	if err != nil {
		return fmt.Errorf("failed to write %s to file: %w", desc, err)
	}
	return nil
}
