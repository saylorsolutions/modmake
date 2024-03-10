package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	. "github.com/saylorsolutions/modmake"
)

var (
	gitBranch = "UNKNOWN BRANCH"
	gitHash   = "UNKNOWN COMMIT"
)

func main() {
	panicFatal("Failed to query module/filesystem details. Are you running this in a Go project with Go tools installed?", func() {
		Go()
	})
	modRoot := Go().ModuleRoot()
	errFatal(fmt.Sprintf("Failed to change the working directory to module root '%s'", modRoot), modRoot.Chdir())
	flags := setupFlags()
	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Println("Sorry, I don't understand what you mean:", err)
		log.Println("Did you include '--' in your command?")
		flags.Usage()
		os.Exit(1)
	}

	ctx := signalCtx()
	if err := run(ctx, flags); err != nil {
		log.Println("Failed to run modmake:", err)
		log.Fatalln("Try 'modmake --help' to get usage information")
	}
}

func run(ctx context.Context, flags *appFlags) error {
	modRoot := Go().ModuleRoot()
	if flags.help || len(os.Args) == 1 {
		flags.Usage()
		return nil
	}
	if flags.printVersion {
		fmt.Printf("modmake branch: '%s', commit hash: '%s'\n", gitBranch, gitHash)
		return nil
	}
	if flags.rootOverride != "" {
		if err := modRoot.Join(flags.rootOverride).Chdir(); err != nil {
			return err
		}
	}
	if flags.buildOverride != "" {
		override := modRoot.Join(flags.buildOverride)
		if !override.Exists() {
			log.Printf("Unable to locate build override '%s'\n", override)
		}
		log.Printf("Running build %s\n", flags.buildOverride)
		return runBuild(ctx, Go().ToModulePath(flags.buildOverride), flags)
	}
	if Path("modmake").IsDir() {
		log.Println("Running build from modmake")
		return runBuild(ctx, Go().ToModulePath("modmake"), flags)
	}
	if Path("build.go").Exists() {
		log.Println("Running build from build.go")
		return runBuild(ctx, ".", flags)
	}
	return errors.New("unable to resolve build, try providing a build override")
}

func runBuild(ctx context.Context, target string, flags *appFlags) error {
	run := Go().Run(target).Arg(flags.Args()...)
	for _, env := range flags.envVars {
		kv := strings.Split(env, "=")
		if len(kv) != 2 {
			return fmt.Errorf("invalid environment variable format, '%s' must be 'KEY=VALUE'", env)
		}
		run.Env(strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1]))
	}
	return run.Run(ctx)
}

func errFatal(msg string, err error) {
	if err != nil {
		if !strings.HasSuffix(msg, ":") {
			msg += ":"
		}
		log.Fatalln(msg, err)
	}
}

func panicFatal(msg string, fn func()) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("%s: %v", msg, r)
			log.Fatalln(err)
		}
	}()
	fn()
}
