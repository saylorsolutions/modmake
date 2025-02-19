package main

import (
	"fmt"
	"github.com/saylorsolutions/modmake/cmd/modmake-docs/internal/templates"
	flag "github.com/spf13/pflag"
	"strings"
)

type AppFlags struct {
	flags             *flag.FlagSet
	helpFlag          bool
	basePath          string
	latestGo          string
	modmakeVersion    string
	latestSupportedGo string
}

func (f *AppFlags) Error(msg string, args ...any) {
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	fmt.Printf(msg, args...)
	f.flags.Usage()
}

func (f *AppFlags) LoadParams(params *templates.Params) {
	params.LatestGoVersion = f.latestGo
	params.LatestSupportedGoVersion = f.latestSupportedGo
	params.BasePath = f.basePath
	params.ModmakeVersion = f.modmakeVersion
}

func GlobalFlags(subCommands map[string]string) *AppFlags {
	af := new(AppFlags)
	flags := flag.NewFlagSet("global", flag.ContinueOnError)
	flags.BoolVarP(&af.helpFlag, "help", "h", false, "Shows this help message")
	flags.StringVar(&af.basePath, "base-path", "", "Sets the base path for page resource references")
	flags.StringVar(&af.latestGo, "latest-go", "", "Sets the latest Go version")
	flags.StringVar(&af.latestSupportedGo, "latest-supported", "", "Sets the latest supported Go version")
	flags.StringVar(&af.modmakeVersion, "modmake-version", "", "Specifies the modmake version to reference")

	flags.Usage = globalUsage(flags, subCommands)
	af.flags = flags
	return af
}

func globalUsage(flags *flag.FlagSet, subCommands map[string]string) func() {
	return func() {
		var subs []string
		for cmd, desc := range subCommands {
			subs = append(subs, fmt.Sprintf("%-10s - %s", cmd, desc))
		}
		fmt.Printf(`Used to generate and serve documentation for modmake.

USAGE: modmake-docs SUBCOMMAND FLAGS

SUBCOMMANDS:
%s

FLAGS:
%s
`, strings.Join(subs, "\n"), flags.FlagUsages())
	}
}

func generateUsage(flags *flag.FlagSet) func() {
	return func() {
		fmt.Printf(`Generates HTML documentation for modmake in the current directory.

USAGE: modmake-docs generate

FLAGS:
%s
`, flags.FlagUsages())
	}
}
