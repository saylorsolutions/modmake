package main

import (
	"fmt"
	"github.com/saylorsolutions/modmake"
	flag "github.com/spf13/pflag"
	"strings"
	"time"
)

type appFlags struct {
	*flag.FlagSet
	help          bool
	envVars       []string
	rootOverride  string
	buildOverride string
	printVersion  bool
	watchDir      string
	watchInterval time.Duration
	watchSubdirs  bool
}

func setupFlags() *appFlags {
	flags := &appFlags{
		FlagSet: flag.NewFlagSet("modmake", flag.ContinueOnError),
	}
	flags.BoolVarP(&flags.help, "help", "h", false, "Prints this usage information")
	flags.StringArrayVarP(&flags.envVars, "set-env", "e", nil, "Sets one or more environment variables before calling the build")
	flags.StringVarP(&flags.rootOverride, "workdir", "w", "", "Overrides the default logic of setting the working directory to the root of the module. Assumed to be a path relative to the module root")
	flags.StringVarP(&flags.buildOverride, "build", "b", "", "Overrides the build location resolution logic and specifies where the build file is located")
	flags.BoolVar(&flags.printVersion, "version", false, "Prints the git branch and hash from which the CLI was built")
	flags.StringVar(&flags.watchDir, "watch", "", "Watches a directory for changes and re-runs the given step when a file changes. A comma-separated filename glob pattern list can be added to the watch path to only restart the task when matching files are changed. Filename globs, if used, should be separated from the path by ':'.")
	flags.BoolVar(&flags.watchSubdirs, "subdirs", false, "Used with 'watch' to also watch sub-directories for file changes. Sub-directories created after watching has started will not be watched for file changes.")
	flags.DurationVar(&flags.watchInterval, "debounce", 200*time.Millisecond, "Sets the debounce interval for watched tasks. This only applies if the 'watch' flag is used. Must be greater than zero, and may need to be set higher if files matching a pattern are generated while the step(s) run.")

	flags.Usage = func() {
		fmt.Printf(`modmake is a convenience CLI that allows easily auto-discovering and running a modmake build.
It's not strictly necessary and, if you're more comfortable with plain 'go run' and terminal commands, it might just get in your way.

Here are some reasons you may want to use this:
* You don't want to think about go tools or terminal semantics.
* You don't want to type as much.
* You want a consistent build resolution behavior (modmake/build.go then build.go).
* You want the build's working directory to default to the root of your module.
* You want to easily and consistently override the working directory for the build.
* You want to easily set one or more environment variables for the build, maybe so they act as parameters.
* You want an easier way to invoke the build multiple times with different Go versions (-e GOROOT=/other/go/root).

USAGE: modmake MODMAKE_FLAGS [-- BUILD_FLAGS] BUILD_STEPS

>>> Note that if you use any BUILD_FLAGS, '--' is necessary to disambiguate between flags for this CLI and flags for the build.
    Running modmake with no flags/arguments will print this usage information.

MODMAKE_FLAGS:  Described in the FLAGS section below.
BUILD_FLAGS:    Any flag that you want passed to your build. Must be separated with '--' to disambiguate from modmake CLI flags.
BUILD_STEPS:    Build steps that should be called.

BUILD RESOLUTION:
By default, a build file will be resolved in the following order.
Later resolution steps will not be done if an earlier condition is satisfied.

1. Checking if the build location override has been set, and running that.
2. Checking for the existence of a 'modmake' directory in the working directory, and running from here if it exists.
3. Checking for the existence of a 'build.go' file in the working directory, and running it.

FLAGS:
%s
`, flags.FlagUsages())
	}
	return flags
}

func (f *appFlags) watchDirectory() modmake.PathString {
	if len(f.watchDir) == 0 {
		return ""
	}
	idx := strings.LastIndex(f.watchDir, ":")
	if idx == -1 {
		return modmake.Path(f.watchDir)
	}
	return modmake.Path(f.watchDir[:idx])
}

func (f *appFlags) watchPatterns() []string {
	if len(f.watchDir) == 0 {
		return nil
	}
	idx := strings.LastIndex(f.watchDir, ":")
	if idx == -1 {
		return nil
	}
	patternStr := strings.TrimSpace(f.watchDir[idx+1:])
	if len(patternStr) == 0 {
		return nil
	}
	return strings.Split(patternStr, ",")
}
