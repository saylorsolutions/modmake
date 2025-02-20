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

func generateCodeDocs(ctx context.Context, params templates.Params, genPath string, directories ...string) error {
	mod := docparser.NewModule()
	for _, dir := range directories {
		if err := mod.ParsePackageDir(dir); err != nil {
			return err
		}
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
