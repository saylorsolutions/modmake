package modmake

import (
	"context"
	"errors"
	"fmt"
	flag "github.com/spf13/pflag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

const (
	CtxWorkdir = "modmake_workdir"
)

var (
	ErrMissingWorkdir = errors.New("missing workdir from context")
)

func sigCtx() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	go func() {
		<-sigs
		cancel()
	}()
	return ctx, cancel
}

// Execute executes a Build as configured.
// It takes string arguments to make it easy to run with 'go run'.
// Run this with the -h flag to see usage information.
func (b *Build) Execute(args ...string) (err error) {
	if err := b.cyclesCheck(); err != nil {
		return err
	}
	flags := flag.NewFlagSet("build", flag.ContinueOnError)

	var (
		flagHelp         bool
		flagDoBench      bool
		flagSkipTests    bool
		flagSkipInstall  bool
		flagSkipGenerate bool
		flagWorkdir      string
		flagTimeout      time.Duration
	)

	flags.BoolVarP(&flagHelp, "help", "h", false, "Prints this usage information")
	flags.BoolVar(&flagDoBench, "run-benchmark", false, "Runs the benchmark step")
	flags.BoolVar(&flagSkipTests, "skip-test", false, "Skips the test step, but not its dependencies.")
	flags.BoolVar(&flagSkipInstall, "skip-tools", false, "Skips the tools install step, but not its dependencies.")
	flags.BoolVar(&flagSkipGenerate, "skip-generate", false, "Skips the generate step, but not its dependencies.")
	flags.StringVar(&flagWorkdir, "workdir", ".", "Sets the working directory for the build")
	flags.DurationVar(&flagTimeout, "timeout", 0, "Sets a timeout duration for this build run")

	flags.Usage = func() {
		fmt.Printf(`Executes a modmake build

Usage:
    go run BUILD_FILE.go graph
	go run BUILD_FILE.go steps
	go run BUILD_FILE.go [FLAGS] STEP...

There are specialized commands that can be used to introspect the build.
  - graph: Passing this command as the first argument will emit a step dependency graph with descriptions on standard out. This can also be generated with Build.Graph().
  - steps: Prints the list of all steps in this build.

See https://github.com/saylorsolutions/modmake for detailed usage information.

%s

`, flags.FlagUsages())
		b.Graph()
	}
	if err := flags.Parse(args); err != nil {
		return err
	}

	if flagSkipTests {
		b.testStep.Skip()
	}
	if flagSkipInstall {
		b.toolsStep.Skip()
	}
	if flagSkipGenerate {
		b.generateStep.Skip()
	}
	if flagDoBench {
		b.benchStep.UnSkip()
	}
	flagWorkdir = strings.TrimSpace(flagWorkdir)
	if len(flagWorkdir) == 0 {
		flagWorkdir = b.workdir
	}
	flagWorkdir, err = filepath.Abs(flagWorkdir)
	if err != nil {
		return fmt.Errorf("error getting absolute path for workdir: %w", err)
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("error running build: %v", r)
		}
	}()
	ctx, cancel := sigCtx()
	defer cancel()
	ctx = context.WithValue(ctx, CtxWorkdir, flagWorkdir)

	if flagTimeout > 0 {
		var _cancel context.CancelFunc
		ctx, _cancel = context.WithTimeout(ctx, flagTimeout)
		defer _cancel()
	}

	start := time.Now()
	for i, stepName := range flags.Args() {
		switch {
		case i == 0 && stepName == "graph":
			b.Graph()
			return nil
		case i == 0 && stepName == "steps":
			steps := b.Steps()
			for i := 0; i < len(steps); i++ {
				steps[i] = steps[i] + " - " + b.Step(steps[i]).description
			}
			fmt.Println(strings.Join(steps, "\n"))
			return nil
		default:
			if err := b.Step(stepName).Run(ctx); err != nil {
				return fmt.Errorf("error running build: %w", err)
			}
		}
	}

	log.Printf("Ran successfully in %s\n", time.Since(start).Round(time.Millisecond).String())
	return nil
}
