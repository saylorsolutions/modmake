package modmake

import (
	"context"
	"errors"
	"fmt"
	"github.com/fatih/color"
	flag "github.com/spf13/pflag"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"time"
)

var (
	errColor   = color.New(color.FgRed, color.Bold).SprintFunc()
	okColor    = color.New(color.FgGreen, color.Bold).SprintFunc()
	warnColor  = color.New(color.FgYellow, color.Bold).SprintFunc()
	debugColor = color.New(color.FgCyan).SprintFunc()
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
	if err := b.ExecuteErr(args...); err != nil {
		log.Fatalf("%s: %v\n", errColor("Error executing build"), err.Error())
	}
}

// ExecuteErr executes a Build as configured, as if it were a CLI application, and returns an error if anything goes wrong.
// It takes string arguments to allow overriding the default of capturing os.Args.
// Run this with the -h flag to see usage information.
// If an error occurs within Execute, then the error will be logged and [os.Exit] will be called with a non-zero exit code.
//
// Note that the build will attempt to change its working directory to the root of the module, so all filesystem paths should be relative to the root.
// [GoTools.ToModulePath] may be useful to adhere to this constraint.
func (b *Build) ExecuteErr(args ...string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			stack := debug.Stack()
			err = fmt.Errorf("caught panic while running build: %v\n%s", r, string(stack))
		}
	}()
	if err := Go().ModuleRoot().Chdir(); err != nil {
		return errors.New("failed to change working directory to module root: " + err.Error())
	}
	if err := b.cyclesCheck(); err != nil {
		return err
	}
	flags := flag.NewFlagSet("build", flag.ContinueOnError)

	var (
		flagHelp     bool
		flagSkip     []string
		flagNoSkip   []string
		flagSkipDeps bool
		flagTimeout  time.Duration
		flagDryRun   bool
		flagDebugLog bool
		flagVerbose  bool
		flagNoColor  bool
	)

	flags.BoolVarP(&flagHelp, "help", "h", false, "Prints this usage information.")
	flags.StringArrayVar(&flagSkip, "skip", nil, "Skips one or more named steps.")
	flags.StringArrayVar(&flagNoSkip, "no-skip", nil, "Specifies that the one or more steps should not be skipped, if they need to run. Note that specifically referencing a step will always run it, even if it's skipped by default.")
	flags.BoolVar(&flagSkipDeps, "only", false, "Skips running the named step's dependencies, only runs the step itself.")
	flags.DurationVar(&flagTimeout, "timeout", 0, "Sets a timeout duration for this build run.")
	flags.BoolVar(&flagDryRun, "dry-run", false, "Runs the build's steps in dry run mode. No actual operations will be executed, but logs will still be printed.")
	flags.BoolVar(&flagDebugLog, "debug", false, "Specifies that debug step logs should be emitted.")
	flags.BoolVarP(&flagVerbose, "verbose", "v", false, "Used with 'steps' or 'graph' to output all steps, including those that do nothing.")
	flags.BoolVar(&flagNoColor, "no-color", false, "Used to disable colorized output.")

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
	}
	if len(args) == 0 {
		args = os.Args[1:]
	}
	if err := flags.Parse(args); err != nil {
		flags.Usage()
		return err
	}
	if flagNoColor {
		color.NoColor = true
	}

	if flags.NArg() == 0 || flagHelp {
		flags.Usage()
		return
	}

	if flagDebugLog {
		_stepDebugLog = true
	}

	ctx, cancel := sigCtx()
	defer cancel()

	if flagTimeout > 0 {
		var _cancel context.CancelFunc
		ctx, _cancel = context.WithTimeout(ctx, flagTimeout)
		defer _cancel()
	}

	for _, skip := range flagSkip {
		step, ok := b.StepOk(skip)
		if !ok {
			log.Printf("%s: User asked that step '%s' be skipped, but it doesn't exist in this model\n", warnColor("WARN"), skip)
			continue
		}
		step.Skip()
	}
	for _, noskip := range flagNoSkip {
		step, ok := b.StepOk(noskip)
		if !ok {
			log.Printf("%s: User asked that step '%s' not be skipped, but it doesn't exist in this model\n", warnColor("WARN"), noskip)
			continue
		}
		step.UnSkip()
	}

	start := time.Now()
	if flagDryRun {
		log.Printf("Running build in %s mode, steps will not run.\n", okColor("DRY RUN"))
	}
	for i, stepName := range flags.Args() {
		switch {
		case i == 0 && stepName == "graph":
			b.Graph(flagVerbose)
			return
		case i == 0 && stepName == "steps":
			var buf strings.Builder
			steps := b.Steps()
			for i := 0; i < len(steps); i++ {
				step := b.Step(steps[i])
				if flagVerbose || step.hasOperation() {
					buf.WriteString(fmt.Sprintf("%s - %s\n", steps[i], step.description))
				}
			}
			fmt.Println(buf.String())
			return
		default:
			step, ok := b.StepOk(stepName)
			if !ok {
				return fmt.Errorf("build step '%s' does not exist", errColor(stepName))
			}
			if flagSkipDeps {
				step.SkipDependencies()
			}

			// Make sure that this step is not skipped, since it was called out by name.
			step.UnSkip()
			run := step.Run
			if flagDryRun {
				run = step.DryRun
			}
			if err := run(ctx); err != nil {
				return fmt.Errorf("error running build: %v", err)
			}
		}
	}

	log.Printf(okColor(fmt.Sprintf("Ran successfully in %s\n", time.Since(start).Round(time.Millisecond).String())))
	return nil
}
