package modmake

import (
	"context"
	"fmt"
	flag "github.com/spf13/pflag"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"time"
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

// Execute executes a Build as configured, as if it were a CLI application.
// It takes string arguments to allow overriding the default of capturing os.Args.
// Run this with the -h flag to see usage information.
// If an error occurs within Execute, then the error will be logged and [os.Exit] will be called with a non-zero exit code.
//
// Note that the build will attempt to change its working directory to the root of the module, so all filesystem paths should be relative to the root.
// [GoTools.ToModulePath] may be useful to adhere to this constraint.
func (b *Build) Execute(args ...string) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			log.Fatalf("caught panic while running build: %v\n%s", r, string(stack))
		}
	}()
	if err := os.Chdir(Go().ModuleRoot()); err != nil {
		panic("Failed to change working directory to module root: " + err.Error())
	}
	if err := b.cyclesCheck(); err != nil {
		log.Fatalln(err)
	}
	flags := flag.NewFlagSet("build", flag.ContinueOnError)

	var (
		flagHelp         bool
		flagDoBench      bool
		flagSkipTests    bool
		flagSkipInstall  bool
		flagSkipGenerate bool
		flagSkipDeps     bool
		flagTimeout      time.Duration
	)

	flags.BoolVarP(&flagHelp, "help", "h", false, "Prints this usage information")
	flags.BoolVar(&flagDoBench, "run-benchmark", false, "Runs the benchmark step")
	flags.BoolVar(&flagSkipTests, "skip-test", false, "Skips the test step, but not its dependencies.")
	flags.BoolVar(&flagSkipInstall, "skip-tools", false, "Skips the tools install step, but not its dependencies.")
	flags.BoolVar(&flagSkipGenerate, "skip-generate", false, "Skips the generate step, but not its dependencies.")
	flags.BoolVar(&flagSkipDeps, "skip-dependencies", false, "Skips running the named step's dependencies.")
	flags.DurationVar(&flagTimeout, "timeout", 0, "Sets a timeout duration for this build run")

	flags.Usage = func() {
		fmt.Printf(`Executes this modmake build

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
	if len(args) == 0 {
		args = os.Args[1:]
	}
	if err := flags.Parse(args); err != nil {
		flags.Usage()
		log.Fatalln(err)
	}

	if flags.NArg() == 0 || flagHelp {
		flags.Usage()
		return
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

	ctx, cancel := sigCtx()
	defer cancel()

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
			return
		case i == 0 && stepName == "steps":
			steps := b.Steps()
			for i := 0; i < len(steps); i++ {
				steps[i] = steps[i] + " - " + b.Step(steps[i]).description
			}
			fmt.Println(strings.Join(steps, "\n"))
			return
		default:
			step := b.Step(stepName)
			if flagSkipDeps {
				step.SkipDependencies()
			}

			// Make sure that this step is not skipped, since it was called out by name.
			step.UnSkip()
			if err := step.Run(ctx); err != nil {
				log.Fatalf("error running build: %v\n", err)
			}
		}
	}

	log.Printf("Ran successfully in %s\n", time.Since(start).Round(time.Millisecond).String())
}
