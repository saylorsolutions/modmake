package main

import (
	"fmt"
	flag "github.com/spf13/pflag"
)

type appFlags struct {
	*flag.FlagSet
	help          bool
	envVars       []string
	rootOverride  string
	buildOverride string
	printVersion  bool
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
BUILD_FLAGS:    Any flag that you want passed to your build.
BUILD_STEPS:    Build steps that should be called.

BUILD RESOLUTION:
By default, a build file will be resolved in the following order.
Later resolution steps will not be done if an earlier condition is satisfied.

1. Checking if the build location override has been set, and running that.
2. Checking for the existence of a modmake directory in the working directory, and running from here if it exists.
3. Checking for the existence of a build.go file in the working directory, and running it.

FLAGS:
%s
`, flags.FlagUsages())
	}
	return flags
}
