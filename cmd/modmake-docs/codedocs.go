package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/saylorsolutions/modmake/cmd/modmake-docs/internal/docparser"
	"github.com/saylorsolutions/modmake/cmd/modmake-docs/internal/templates"
	"os"
	"path/filepath"
)

func generateCodeDocs(ctx context.Context, params templates.Params) error {
	genPath := params.GenDir
	directories := params.GoDocDirs
	mod := docparser.NewModule()
	for _, dir := range directories {
		if err := mod.ParsePackageDir(dir); err != nil {
			return err
		}
	}
	var godocbuf bytes.Buffer
	if err := templates.GoDocPage(params, mod).Render(ctx, &godocbuf); err != nil {
		return fmt.Errorf("failed to render godoc page: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(genPath, "godoc"), 0755); err != nil {
		return fmt.Errorf("failed to create godoc directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(genPath, "godoc", "index.html"), godocbuf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write generated godoc index.html: %w", err)
	}
	for _, pkg := range mod.Packages {
		genPath := filepath.Join(genPath, "godoc", filepath.FromSlash(pkg.ImportName))
		if err := os.MkdirAll(genPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory for go doc generation '%s': %w", genPath, err)
		}
		var buf bytes.Buffer
		if err := templates.PkgPage(params, pkg).Render(ctx, &buf); err != nil {
			return fmt.Errorf("failed to render index page for package '%s': %w", pkg.ImportName, err)
		}
		if err := os.WriteFile(filepath.Join(genPath, "index.html"), buf.Bytes(), 0644); err != nil {
			return fmt.Errorf("failed to write data to new package index file in '%s': %w", genPath, err)
		}
	}
	return nil
}
