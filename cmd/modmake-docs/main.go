package main

import (
	"embed"
	_ "embed"
	"github.com/saylorsolutions/modmake/cmd/modmake-docs/internal/templates"
	"log"
	"os"
)

var (
	//go:embed img/*
	imgFS embed.FS
)

func main() {
	subCommands := map[string]string{
		"generate": "Generates modmake documentation in the current directory",
		"serve":    "Serves documentation as a website",
	}
	flags := GlobalFlags(subCommands)
	args := os.Args[1:]
	if len(args) == 0 {
		flags.Error("No subcommand specified")
		os.Exit(1)
	}
	if err := flags.Parse(args); err != nil {
		flags.Error("Failed to parse flags: %v", err)
		os.Exit(1)
	}
	if flags.helpFlag {
		flags.Error("Help requested")
		return
	}
	params := flags.LoadParams()
	params.Content.AddSection(
		templates.IntroSection(params),
		templates.BuildModelSection(params),
		templates.ModmakeCLISection(params),
		templates.UtilitiesSection(),
	)
	var command string
	if flags.NArg() > 0 {
		command = flags.Arg(0)
		if err := flags.Parse(args[1:]); err != nil {
			flags.Error("Failed to parse flags: %v", err)
			os.Exit(1)
		}
	} else {
		flags.Error("No command specified")
		os.Exit(1)
	}

	var cmdErr error
	switch command {
	case "generate":
		cmdErr = doGenerate(params)
	case "serve":
		cmdErr = doServe(flags, params)
	default:
		log.Println("Command not recognized:", command)
		os.Exit(1)
	}
	if cmdErr != nil {
		flags.Error("Failed to run command '%s': %v", command, cmdErr)
		os.Exit(1)
	}
}
