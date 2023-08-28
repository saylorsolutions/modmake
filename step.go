package modmake

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// Runner is any type that may be executed.
type Runner interface {
	// Run should immediately execute in the current goroutine when called to ensure predictable build semantics.
	// Run may initiate other goroutines, but they should complete and be cleaned up before Run returns.
	Run(context.Context) error
}

// RunnerFunc is a convenient way to make a function that satisfies the Runner interface.
type RunnerFunc func(ctx context.Context) error

func (fn RunnerFunc) Run(ctx context.Context) error {
	return fn(ctx)
}

// ContextAware creates a Runner that wraps the parameter with context handling logic.
// In the event that the context is done, the context's error is returned.
// This should not be used if custom [context.Context] handling is desired.
func ContextAware(r Runner) Runner {
	return RunnerFunc(func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return r.Run(ctx)
		}
	})
}

// NoOp is a Runner placeholder that immediately returns nil.
func NoOp() Runner {
	return RunnerFunc(func(ctx context.Context) error {
		return nil
	})
}

// Todo returns a Runner that prints "Not yet defined" with the standard logger.
// This is intended to be used where a Runner should be defined, but hasn't yet, much like [context.TODO].
func Todo() Runner {
	return RunnerFunc(func(ctx context.Context) error {
		log.Println("Not yet defined")
		return nil
	})
}

// Error will create a Runner returning an error, creating with passing msg and args to [fmt.Errorf].
func Error(msg string, args ...any) Runner {
	return RunnerFunc(func(ctx context.Context) error {
		return fmt.Errorf(msg, args...)
	})
}

// RunState indicates the state of a Step.
type RunState int

const (
	StateNotRun     RunState = iota // StateNotRun means that this Step has not yet run.
	StateSuccessful                 // StateSuccessful means that the Step has already run successfully.
	StateFailed                     // StateFailed means that the Step has already run and failed.
)

var reservedStepNames = map[string]bool{
	"tools":     true,
	"generate":  true,
	"test":      true,
	"benchmark": true,
	"build":     true,
	"package":   true,

	// Reserved so they don't conflict with Execute commands.
	"graph": true,
	"steps": true,
}

// Step is a step in a Build.
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
	if ok := reservedStepNames[name]; ok {
		panic(fmt.Sprintf("step name '%s' is reserved by the build system", name))
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

// DependsOn makes this Step depend on the given step.
// The given step must execute successfully for this Step to be executed.
func (s *Step) DependsOn(dependency *Step) *Step {
	if dependency == nil {
		panic("attempt to add nil Step")
	}
	if s.build != nil {
		dependency.setBuild(s.build)
	}
	s.dependencies = append(s.dependencies, dependency)
	return s
}

func (s *Step) DependsOnRunner(name, description string, r Runner) *Step {
	step := NewStep(name, description).Does(r)
	return s.DependsOn(step)
}

func (s *Step) DependsOnFunc(name, description string, fn RunnerFunc) *Step {
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

// DoesFunc specifies the RunnerFunc that should happen as a result of executing this Step.
func (s *Step) DoesFunc(fn RunnerFunc) *Step {
	return s.Does(fn)
}

// BeforeRun adds an operation that will execute before this Step.
// BeforeRun operations will happen after this Step's dependencies.
func (s *Step) BeforeRun(op Runner) *Step {
	if op == nil {
		return s
	}
	s.beforeOp = append(s.beforeOp, op)
	return s
}

// AfterRun adds an operation that will execute after this Step.
// AfterRun operations will happen before any dependent Step.
func (s *Step) AfterRun(op Runner) *Step {
	if op == nil {
		return s
	}
	s.afterOp = append(s.afterOp, op)
	return s
}

func (s *Step) Run(ctx context.Context) error {
	if s.shouldSkipDeps {
		log.Printf("[%s] Skipping dependencies\n", s.name)
	} else {
		for _, d := range s.dependencies {
			if d.state != StateNotRun {
				continue
			}
			if err := d.Run(ctx); err != nil {
				s.state = StateFailed
				return err
			}
		}
	}

	if s.state != StateNotRun {
		return nil
	}
	if s.shouldSkip {
		log.Printf("[%s] Skipping step\n", s.name)
		return nil
	}

	if len(s.beforeOp) > 0 {
		log.Printf("[%s] Runnning before hooks...\n", s.name)
		start := time.Now()
		for _, before := range s.beforeOp {
			if err := before.Run(ctx); err != nil {
				log.Printf("[%s] Error running step: %v\n", s.name, err)
				s.state = StateFailed
				return err
			}
		}
		log.Printf("[%s] Before hooks ran successfully in %s\n", s.name, time.Since(start).Round(time.Millisecond).String())
	}
	s.state = StateSuccessful

	if s.operation != nil {
		log.Printf("[%s] Running step...\n", s.name)
		runStart := time.Now()
		if err := s.operation.Run(ctx); err != nil {
			log.Printf("Error running step '%s': %v\n", s.name, err)
			s.state = StateFailed
			return err
		}
		log.Printf("[%s] Successfully ran step in %s\n", s.name, time.Since(runStart).Round(time.Millisecond).String())
	}

	if len(s.afterOp) > 0 {
		log.Printf("[%s] Runnning after hooks...\n", s.name)
		start := time.Now()
		for _, after := range s.afterOp {
			if err := after.Run(ctx); err != nil {
				log.Printf("[%s] Error running after hooks: %v", s.name, err)
				s.state = StateFailed
				return err
			}
		}
		log.Printf("[%s] Successfully ran after hooks in %s\n", s.name, time.Since(start).Round(time.Millisecond).String())
	}
	return nil
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
