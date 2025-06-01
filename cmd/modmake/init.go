package main

import (
	"context"
	_ "embed"
	"fmt"
	. "github.com/saylorsolutions/modmake" //nolint:staticcheck // This is a DSL-type API
	"log"
	"os"
	"strings"
	"text/template"
	"time"
)

var (
	//go:embed init.got
	initText  string
	initTempl = template.Must(template.New("init.got").Parse(initText))
)

func doInit(modRoot PathString) error {
	const (
		modmakeModuleName = "github.com/saylorsolutions/modmake"
	)
	newBuildLoc := modRoot.Join("modmake", "build.go")
	if newBuildLoc.Exists() {
		return fmt.Errorf("cannot init because '%s' already exists", newBuildLoc.String())
	}
	initScript := Script(
		Chdir(modRoot, WithoutContext(func() error {
			log.Println("Checking for module Modmake dependency...")
			goMod := Path("go.mod")
			modFileContent, err := os.ReadFile(goMod.String())
			if err != nil {
				return err
			}
			if !strings.Contains(string(modFileContent), modmakeModuleName) {
				version := runtimeVersion
				if version == unknownVersion {
					log.Println("Init is best used with a pre-built version of the Modmake CLI to enable pinning compatible runtime versions")
					log.Println("Pinning to latest as the most reasonable default")
					version = "latest"
				} else {
					version = "v" + version
				}

				if err := Go().Get(modmakeModuleName + "@" + version).Run(context.Background()); err != nil {
					return err
				}
				log.Printf("Added Modmake version '%s' to module", version)
				return nil
			}
			log.Println("Modmake is already a dependency")
			return nil
		})),
		Mkdir(newBuildLoc.Dir(), 0755),
		WithoutContext(func() error {
			log.Println("Creating template build...")
			fd, err := newBuildLoc.Create()
			if err != nil {
				return err
			}
			return initTempl.Execute(fd, nil)
		}),
	)
	timeout, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return initScript.Run(timeout)
}
