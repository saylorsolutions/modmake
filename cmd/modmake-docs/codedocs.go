package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/saylorsolutions/modmake/cmd/modmake-docs/internal/docparser"
	"github.com/saylorsolutions/modmake/cmd/modmake-docs/internal/templates"
	"log"
	"os"
	"path/filepath"
)

func generateCodeDocs(ctx context.Context, params templates.Params) error {
	genPath := params.GenDir
	directories := params.GoDocDirs
	mod := docparser.NewModule()
	for _, dir := range directories {
		log.Println("Parsing module directory:", dir)
		if err := mod.ParsePackageDir(dir); err != nil {
			return err
		}
	}
	var godocbuf bytes.Buffer
	if err := templates.GoDocPage(params, mod).Render(ctx, &godocbuf); err != nil {
		return fmt.Errorf("failed to render godoc page: %w", err)
	}
	godocDir := filepath.Join(genPath, "godoc")
	log.Println("Creating directory:", godocDir)
	if err := os.MkdirAll(godocDir, 0700); err != nil {
		return fmt.Errorf("failed to create godoc directory: %w", err)
	}
	godocIndex := filepath.Join(godocDir, "index.html")
	log.Println("Creating file:", godocIndex)
	if err := os.WriteFile(godocIndex, godocbuf.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed to write generated godoc index.html: %w", err)
	}
	for _, pkg := range mod.Packages {
		genPath := filepath.Join(genPath, "godoc", filepath.FromSlash(pkg.ImportName))
		log.Println("Creating directory:", genPath)
		if err := os.MkdirAll(genPath, 0700); err != nil {
			return fmt.Errorf("failed to create directory for go doc generation '%s': %w", genPath, err)
		}
		var buf bytes.Buffer
		if err := templates.PkgPage(params, pkg).Render(ctx, &buf); err != nil {
			return fmt.Errorf("failed to render index page for package '%s': %w", pkg.ImportName, err)
		}
		genFilePath := filepath.Join(genPath, "index.html")
		log.Println("Creating file:", genFilePath)
		if err := os.WriteFile(genFilePath, buf.Bytes(), 0600); err != nil {
			return fmt.Errorf("failed to write data to new package index file in '%s': %w", genPath, err)
		}
	}
	return nil
}
