package modmake

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Runner is any type that may be executed with a [context.Context], and could return an error.
type Runner interface {
	// Run should immediately execute in the current goroutine when called to ensure predictable build semantics.
	// Run may initiate other goroutines, but they should complete and be cleaned up before Run returns.
	Run(context.Context) error
}

// RunState indicates the state of a Step.
type RunState int

const (
	StateNotRun     RunState = iota // StateNotRun means that this Step has not yet run.
	StateSuccessful                 // StateSuccessful means that the Step has already run successfully.
	StateFailed                     // StateFailed means that the Step has already run and failed.
)

var standardStepNames = map[string]struct{}{
	"tools":     {},
	"generate":  {},
	"test":      {},
	"benchmark": {},
	"build":     {},
	"package":   {},
}

var reservedStepNames = map[string]struct{}{}

func init() {
	for k, v := range standardStepNames {
		reservedStepNames[k] = v
	}

	// Reserved so they don't conflict with Execute commands.
	reservedStepNames["graph"] = struct{}{}
	reservedStepNames["steps"] = struct{}{}
}

// Step is a step in a Build, a consistent fixture that may be invoked.
// A Step may depend on others to set up pre-conditions that must be done before this Step executes.
// Additionally, a Step may have actions that take place before and/or after this Step runs.
//
// Operations on a Step will change the underlying logic.
// The current Step will be returned as a convenience to allow chaining multiple mutations.
type Step struct {
	name           string
	description    string
	dependencies   []*Step
	beforeOp       []Runner
	operation      Runner
	afterOp        []Runner
	state          RunState
	shouldSkip     bool
	shouldSkipDeps bool
	build          *Build
	dryRun         bool
}

func newStep(name, description string) *Step {
	description = strings.TrimSpace(description)
	if len(description) == 0 {
		description = "No description"
	}
	return &Step{
		name:        name,
		description: description,
	}
}

// NewStep creates a new Step with the given name and description.
// If no description is given (indicated by an empty string), then the default description "No description" will be assigned.
// By default, [Step.Run] will do nothing, have no dependencies, and have no before/after hooks.
func NewStep(name, description string) *Step {
	name = strings.ToLower(name)
	if _, ok := reservedStepNames[name]; ok {
		panic(fmt.Sprintf("step name '%s' is reserved by the build system", name))
	}
	if strings.Contains(name, ":") {
		panic(fmt.Sprintf("step name '%s' includes the character ':' reserved for import separators", name))
	}
	return newStep(name, description)
}

func (s *Step) setBuild(build *Build) {
	if step, ok := build.stepNames[s.name]; ok {
		if s != step {
			panic(fmt.Sprintf("duplicate step name '%s'", s.name))
		}
		return
	}
	s.build = build
	build.stepNames[s.name] = s
	for _, dep := range s.dependencies {
		dep.setBuild(build)
	}
}

// DependsOn makes this Step depend on the given Step.
// The given step must execute successfully for this Step to be executed.
func (s *Step) DependsOn(dependency *Step) *Step {
	if dependency == nil {
		panic("attempt to add nil Step")
	}
	if s.build != nil {
		dependency.setBuild(s.build)
	} else if dependency.build != nil {
		s.build = dependency.build
	}
	s.dependencies = append(s.dependencies, dependency)
	return s
}

// DependsOnRunner wraps the given Runner in a Step using the name and description parameters, and calls DependsOn with it.
func (s *Step) DependsOnRunner(name, description string, fn Runner) *Step {
	step := NewStep(name, description).Does(fn)
	return s.DependsOn(step)
}

// Does specifies the operation that should happen as a result of executing this Step.
func (s *Step) Does(operation Runner) *Step {
	if operation == nil {
		return s
	}
	s.operation = operation
	return s
}

func (s *Step) hasOperation() bool {
	if s.operation != nil {
		return true
	}
	if len(s.beforeOp) > 0 {
		return true
	}
	if len(s.afterOp) > 0 {
		return true
	}
	return s.hasDepOperation()
}

func (s *Step) hasDepOperation() bool {
	for _, dep := range s.dependencies {
		if dep.hasOperation() {
			return true
		}
	}
	return false
}

// BeforeRun adds an operation that will execute before this Step's operation.
// BeforeRun operations will execute after this Step's dependencies, but before the operation performed during this Step.
func (s *Step) BeforeRun(op Runner) *Step {
	if op == nil {
		return s
	}
	s.beforeOp = append(s.beforeOp, op)
	return s
}

// AfterRun adds an operation that will execute after this Step's operation.
// AfterRun operations will execute before any Step that depends on this Step.
func (s *Step) AfterRun(op Runner) *Step {
	if op == nil {
		return s
	}
	s.afterOp = append(s.afterOp, op)
	return s
}

func (s *Step) Run(ctx context.Context) error {
	ctx, log := WithLogger(ctx, s.name)
	if s.shouldSkipDeps {
		log.Warn("Skipping dependencies")
	} else {
		for _, d := range s.dependencies {
			if d.state != StateNotRun {
				continue
			}
			run := d.Run
			if s.dryRun {
				run = d.DryRun
			}
			if err := run(ctx); err != nil {
				s.state = StateFailed
				return log.WrapErr(err)
			}
		}
	}

	if s.state != StateNotRun {
		return nil
	}
	if s.shouldSkip {
		log.Warn("Skipping step")
		return nil
	}

	if len(s.beforeOp) > 0 {
		log.Info("Running before hooks...")
		start := time.Now()
		if s.dryRun {
			log.Debug("Would have run %d before operations", len(s.beforeOp))
		} else {
			ctx, _ := WithGroup(ctx, "before hooks")
			for i, before := range s.beforeOp {
				ctx, log := WithGroup(ctx, fmt.Sprintf("%d", i))
				if err := before.Run(ctx); err != nil {
					log.Error("Error running step: %v", err)
					s.state = StateFailed
					return log.WrapErr(err)
				}
			}
		}
		log.Info("Before hooks ran successfully in %s", time.Since(start).Round(time.Millisecond).String())
	}
	s.state = StateSuccessful

	if s.operation != nil {
		log.Info("Running step...")
		runStart := time.Now()
		if !s.dryRun {
			if err := s.operation.Run(ctx); err != nil {
				log.Error("Error running step: %v", err)
				s.state = StateFailed
				return log.WrapErr(err)
			}
		}
		log.Info("%s step in %s", okColor("Successfully ran"), time.Since(runStart).Round(time.Millisecond).String())
	}

	if len(s.afterOp) > 0 {
		log.Info("Running after hooks...")
		start := time.Now()
		if s.dryRun {
			log.Debug("Would have run %d after hooks", len(s.afterOp))
		} else {
			ctx, _ := WithGroup(ctx, "after hooks")
			for i, after := range s.afterOp {
				ctx, log := WithGroup(ctx, fmt.Sprintf("%d", i))
				if err := after.Run(ctx); err != nil {
					log.Error("Error running after hooks: %v", err)
					s.state = StateFailed
					return log.WrapErr(err)
				}
			}
		}
		log.Info("Successfully ran after hooks in %s", time.Since(start).Round(time.Millisecond).String())
	}
	return nil
}

func (s *Step) DryRun(ctx context.Context) error {
	s.dryRun = true
	return s.Run(ctx)
}

// Skip will skip execution of this step, including its before/after hooks.
// Dependencies will still be executed unless SkipDependencies is executed.
func (s *Step) Skip() *Step {
	s.shouldSkip = true
	return s
}

// UnSkip is the opposite of Skip, and is useful in the case where a Step is skipped by default.
func (s *Step) UnSkip() *Step {
	s.shouldSkip = false
	return s
}

// SkipDependencies will prevent running dependency Steps for this Step.
func (s *Step) SkipDependencies() *Step {
	s.shouldSkipDeps = true
	return s
}

// ResetState resets the state of a Step such that it can be run again.
// This includes all of its dependencies.
func (s *Step) ResetState() *Step {
	for _, dep := range s.dependencies {
		dep.ResetState()
	}
	s.state = StateNotRun
	return s
}

// Debounce
// Deprecated: use a Task if debounce or multiple executions are needed.
func (s *Step) Debounce(interval time.Duration) Task {
	if s == nil {
		panic("nil step")
	}
	if interval <= time.Duration(0) {
		panic(fmt.Sprintf("invalid debounce interval: %d", int64(interval)))
	}
	var (
		reset = func() {}
	)
	return Task(func(base context.Context) error {
		reset()
		ctx, cancel := context.WithCancel(base)
		reset = sync.OnceFunc(func() {
			cancel()
			s.ResetState()
		})
		defer reset()
		return s.Run(ctx)
	}).Debounce(interval)
}
