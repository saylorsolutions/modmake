package main

import (
	"fmt"
	"github.com/saylorsolutions/modmake/cmd/modmake-docs/internal/templates"
	"github.com/saylorsolutions/x/env"
	flag "github.com/spf13/pflag"
	"path/filepath"
	"strings"
)

const (
	EnvBasePath          = "MD_BASE_PATH"
	EnvLatestGo          = "MD_LATEST_GO"
	EnvSupportedGo       = "MD_SUPPORTED_GO"
	EnvModmakeVersion    = "MD_MODMAKE_VERSION"
	EnvGoDocsDirectories = "MD_GODOC_DIRS"
	EnvGenDirectory      = "MD_GEN_DIR"
)

type AppFlags struct {
	*flag.FlagSet
	helpFlag          bool
	basePath          string
	genDir            string
	latestGo          string
	modmakeVersion    string
	latestSupportedGo string
	goDocDirs         []string
}

func (f *AppFlags) Parse(args []string) error {
	if err := f.FlagSet.Parse(args); err != nil {
		return err
	}
	f.basePath = env.Val(EnvBasePath, f.basePath)
	f.latestGo = env.Val(EnvLatestGo, f.latestGo)
	f.latestSupportedGo = env.Val(EnvSupportedGo, f.latestSupportedGo)
	f.modmakeVersion = env.Val(EnvModmakeVersion, f.modmakeVersion)
	goDocDirs := env.InterpretSlice(EnvGoDocsDirectories, ",", ".", func(val string) (string, bool) {
		val = strings.TrimSpace(val)
		if len(val) == 0 {
			return "", false
		}
		return filepath.FromSlash(val), true
	})
	if len(goDocDirs) > 0 {
		f.goDocDirs = goDocDirs
	}
	f.genDir = filepath.FromSlash(env.Val(EnvGenDirectory, f.genDir))
	return nil
}

func (f *AppFlags) Error(msg string, args ...any) {
	if !strings.HasSuffix(msg, "\n") {
		msg += "\n"
	}
	fmt.Printf(msg, args...)
	f.Usage()
}

func (f *AppFlags) LoadParams() templates.Params {
	params := templates.Params{}
	params.LatestGoVersion = f.latestGo
	params.LatestSupportedGoVersion = f.latestSupportedGo
	params.BasePath = f.basePath
	params.ModmakeVersion = f.modmakeVersion
	params.GenDir = f.genDir
	params.GoDocDirs = f.goDocDirs
	return params
}

func GlobalFlags(subCommands map[string]string) *AppFlags {
	af := new(AppFlags)
	flags := flag.NewFlagSet("global", flag.ContinueOnError)
	flags.BoolVarP(&af.helpFlag, "help", "h", false, "Shows this help message")
	flags.StringVar(&af.basePath, "base-path", "", "Sets the base path for page resource references")
	flags.StringVar(&af.latestGo, "latest-go", "", "Sets the latest Go version")
	flags.StringVar(&af.latestSupportedGo, "latest-supported", "", "Sets the latest supported Go version")
	flags.StringVar(&af.modmakeVersion, "modmake-version", "", "Specifies the modmake version to reference")
	flags.StringSliceVar(&af.goDocDirs, "godoc-dirs", nil, "Specifies directories to source for go doc generation")
	flags.StringVar(&af.genDir, "gen-dir", ".", "Specifies the base path where files should be generated")

	flags.Usage = globalUsage(flags, subCommands)
	af.FlagSet = flags
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
