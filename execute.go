package modmake

import (
	"context"
	"fmt"
	flag "github.com/spf13/pflag"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
func (b *Build) Execute(args []string) (err error) {
	flags := flag.NewFlagSet("build", flag.ContinueOnError)

	var (
		flagHelp         bool
		flagDoBench      bool
		flagSkipTests    bool
		flagSkipInstall  bool
		flagSkipGenerate bool
		flagWorkdir      string
	)

	flags.BoolVarP(&flagHelp, "help", "h", false, "Prints this usage information")
	flags.BoolVar(&flagDoBench, "run-benchmark", false, "Runs the benchmark step")
	flags.BoolVar(&flagSkipTests, "skip-test", false, "Skips the test step, but not its dependencies.")
	flags.BoolVar(&flagSkipInstall, "skip-tools", false, "Skips the tools install step, but not its dependencies.")
	flags.BoolVar(&flagSkipGenerate, "skip-generate", false, "Skips the generate step, but not its dependencies.")
	flags.StringVar(&flagWorkdir, "workdir", ".", "Sets the working directory for the build")

	flags.Usage = func() {
		fmt.Printf(`Executes a modmake build from Go code.

Usage:
    build := modmake.NewBuild()
    // Make modifications to the build as needed.
    args := os.Args()[1:]
    build.Execute(args)

There are specialized commands that can be used to introspect the build.
  - graph: Passing this command as the first argument will emit a step dependency graph with descriptions on standard out. This can also be generated with Build.Graph().

See https://github.com/saylorsolutions/modmake for detailed usage information.

%s`, flags.FlagUsages())
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
	for i, stepName := range flags.Args() {
		if i == 0 && stepName == "graph" {
			b.Graph()
			return nil
		}
		if err := b.Step(stepName).Run(ctx); err != nil {
			return fmt.Errorf("error running build: %w", err)
		}
	}

	return nil
}
