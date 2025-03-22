package modmake

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var (
	ErrCycleDetected = errors.New("possible dependency cycle detected")
)

// Build is a collection of related Steps.
// Together, they form the structure and basis for a build pipeline.
type Build struct {
	toolsStep    *Step
	generateStep *Step
	testStep     *Step
	benchStep    *Step
	buildStep    *Step
	packageStep  *Step
	logger       Logger

	workdir   string
	stepNames map[string]*Step
}

var hasTested bool
var testOnce Task = func(ctx context.Context) error {
	if hasTested {
		return nil
	}
	return Script(
		Go().TestAll(),
		Plain(func() {
			hasTested = true
		}),
	).Run(ctx)
}

// NewBuild constructs a new Build with the standard structure.
// This includes the following steps:
//   - tools for installing and verifying required tooling.
//   - generate for code generation.
//   - test for running unit tests.
//   - benchmark for running benchmarks. This step will be skipped in [Build.Execute] by default, unless included with a flag or called for explicitly.
//   - build for building binaries.
//   - package for building distributable artifacts.
func NewBuild() *Build {
	tools := newStep("tools", "Installs external tools that will be needed later")
	generate := newStep("generate", "Generates code, possibly using external tools").DependsOn(tools)
	test := newStep("test", "Runs unit tests on the code base").DependsOn(generate).Does(testOnce)
	bench := newStep("benchmark", "Runs benchmarking on the code base").DependsOn(test).Does(Go().BenchmarkAll())
	build := newStep("build", "Builds the code base and outputs an artifact").DependsOn(bench)
	pkg := newStep("package", "Bundles one or more built artifacts into one or more distributable packages").DependsOn(build)
	b := &Build{
		workdir:      ".",
		toolsStep:    tools,
		generateStep: generate,
		testStep:     test,
		benchStep:    bench.Skip(),
		buildStep:    build,
		packageStep:  pkg,
		stepNames: map[string]*Step{
			"tools":     tools,
			"generate":  generate,
			"test":      test,
			"benchmark": bench,
			"build":     build,
			"package":   pkg,
		},
		logger: &stepLogger{
			name: "root",
		},
	}
	tools.build = b
	generate.build = b
	test.build = b
	bench.build = b
	build.build = b
	pkg.build = b
	return b
}

// CallBuild allows easily referencing and calling another modmake build.
// os.Chdir will be called with the module root before go-running the build file, so the buildFile parameter should be relative to the module root, like any Modmake build.
// This is safe to use with Git submodules because a GoTools instance will be created based on the location of the Modmake build file and the closest module.
//
// CallBuild is preferable over [Build.Import] for building separate go modules.
// If you're building a component of the same go module, then use [Build.Import].
//
//   - buildLocation should be the filesystem path to the build (file or directory) that should be executed. CallBuild will panic if the file doesn't exist.
//   - args are flags and steps that should be executed in the build. If none are passed, then CallBuild will panic.
func CallBuild(buildLocation PathString, args ...string) *Command {
	if !buildLocation.Exists() {
		panic(fmt.Sprintf("Unable to locate build file at '%s'. If this is in a Git submodule, try updating submodules first", buildLocation.String()))
	}
	if len(args) == 0 {
		panic("No build steps specified")
	}
	gt := goToolsAt(buildLocation)
	rel, err := gt.ModuleRoot().Rel(buildLocation)
	if err != nil {
		panic(fmt.Sprintf("Unable to determine relative location to '%s' from module root path: %v", buildLocation, err))
	}
	// Go tools expects slash format
	relSlash := rel.ToSlash()
	if !strings.HasPrefix(relSlash, "./") {
		relSlash = "./" + relSlash
	}
	return gt.Run(relSlash, args...).WorkDir(gt.ModuleRoot())
}

// CallRemote will execute a Modmake build on a remote module.
// The module parameter is the module name and version in the same form as would be used to 'go get' the module.
// The buildPath parameter is the path within the module, relative to the module root, where the Modmake build file is located.
// Finally, the args parameters are all steps and flags that should be used to invoke the build.
//
// NOTE: If the remote build relies on Git information, then this method of calling the remote build will not work.
// Try using [TempDir] with [github.com/saylorsolutions/modmake/pkg/git.CloneAt] to make sure the build can reference those details.
func CallRemote(module string, buildPath PathString, args ...string) Task {
	return func(ctx context.Context) error {
		info, err := Go().modDownload(ctx, module)
		if err != nil {
			return fmt.Errorf("failed to download remote module '%s': %w", module, err)
		}
		path := Path(info.Dir).JoinPath(buildPath)
		if err := chmodWrite(info.Dir); err != nil {
			return err
		}
		if err := CallBuild(path, args...).Run(ctx); err != nil {
			return fmt.Errorf("failed to invoke module '%s' build at '%s' with build args '%s': %w", module, path.String(), strings.Join(args, " "), err)
		}
		return nil
	}
}

func chmodWrite(root string) error {
	return filepath.Walk(root, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return os.Chmod(path, info.Mode()|0200)
	})
}

// Workdir sets the working directory for running build steps.
// Defaults to the current working directory of the calling executable.
func (b *Build) Workdir(workdir string) *Build {
	b.workdir = workdir
	return b
}

// Tools allows easily referencing the tools Step.
func (b *Build) Tools() *Step {
	return b.toolsStep
}

// Generate allows easily referencing the generate Step.
func (b *Build) Generate() *Step {
	return b.generateStep
}

// Test allows easily referencing the test Step.
func (b *Build) Test() *Step {
	return b.testStep
}

// Benchmark allows easily referencing the benchmark Step.
func (b *Build) Benchmark() *Step {
	return b.benchStep
}

// Build allows easily referencing the build Step.
func (b *Build) Build() *Step {
	return b.buildStep
}

// Package allows easily referencing the package step.
func (b *Build) Package() *Step {
	return b.packageStep
}

// AddStep adds an existing custom step to the build.
func (b *Build) AddStep(step *Step) {
	if step == nil {
		panic("nil step")
	}
	name := strings.ToLower(step.name)
	step.name = name
	if _, ok := reservedStepNames[step.name]; ok {
		panic(fmt.Errorf("step name '%s' is reserved", name))
	}
	if _, ok := b.stepNames[step.name]; ok {
		panic(fmt.Errorf("step name '%s' already exists", name))
	}
	b.stepNames[name] = step
	step.build = b
}

// AddNewStep is a shortcut for adding a new step to a build.
func (b *Build) AddNewStep(name, description string, run Runner) *Step {
	step := NewStep(name, description).Does(run)
	b.AddStep(step)
	return step
}

func (b *Build) Step(name string) *Step {
	step, ok := b.stepNames[name]
	if !ok {
		panic(fmt.Sprintf("referencing non-existent step '%s'", name))
	}
	return step
}

func (b *Build) StepOk(name string) (*Step, bool) {
	step, ok := b.stepNames[name]
	return step, ok
}

func (b *Build) Steps() []string {
	steps := keySlice(b.stepNames)
	sort.Strings(steps)
	return steps
}

// Import will import all steps in the given build, with the given prefix applied and a colon separator.
// No dependencies on the imported steps will be applied to the current build, dependencies must be applied on the parent Build after importing.
//
// This is used to integrate build steps of components in the same go module.
// For building a separate go module (like a Git submodule, for example), use CallBuild.
func (b *Build) Import(prefix string, other *Build) {
	for name, step := range other.stepNames {
		step.name = prefix + ":" + name
		b.AddStep(step)
	}
	for k := range standardStepNames {
		b.Step(k).DependsOn(b.Step(prefix + ":" + k))
	}
}

// ImportAndLink will import the build, and make its standard steps a dependency of the parent Build's standard steps.
func (b *Build) ImportAndLink(prefix string, other *Build) {
	b.Import(prefix, other)
	for k := range standardStepNames {
		b.Step(k).DependsOn(b.Step(prefix + ":" + k))
	}
}

func (b *Build) Graph(verbose bool) {
	if err := b.cyclesCheck(); err != nil {
		panic(err)
	}
	visited := map[string]bool{}
	var buf strings.Builder
	buf.WriteString("Printing build graph\n\n")

	if verbose || b.toolsStep.hasOperation() {
		b.graph(b.toolsStep, 0, &buf, visited)
	}
	if verbose || b.generateStep.hasOperation() {
		b.graph(b.generateStep, 0, &buf, visited)
	}
	if verbose || b.testStep.hasOperation() {
		b.graph(b.testStep, 0, &buf, visited)
	}
	if verbose || b.benchStep.hasOperation() {
		b.graph(b.benchStep, 0, &buf, visited)
	}
	if verbose || b.buildStep.hasOperation() {
		b.graph(b.buildStep, 0, &buf, visited)
	}
	if verbose || b.packageStep.hasOperation() {
		b.graph(b.packageStep, 0, &buf, visited)
	}

	for _, stepName := range b.Steps() {
		if _, ok := reservedStepNames[stepName]; ok {
			continue
		}
		step := b.Step(stepName)
		if verbose || step.hasOperation() {
			b.graph(step, 0, &buf, visited)
		}
	}

	buf.WriteString("\n* - duplicate reference\n")
	if !verbose {
		buf.WriteString("Use -v to print defined steps with no operation or dependent operation\n")
	}
	fmt.Print(buf.String())
}

func (b *Build) graph(step *Step, indent int, buf *strings.Builder, visited map[string]bool) {
	indentStr := strings.Repeat(" ", indent)
	if indent > 0 {
		indentStr += "-> "
	}
	var skipStr string
	if step.shouldSkip || step.shouldSkipDeps {
		switch {
		case step.shouldSkip && step.shouldSkipDeps:
			skipStr = " (skip all)"
		case step.shouldSkip:
			skipStr = " (skip step)"
		case step.shouldSkipDeps:
			skipStr = " (skip deps)"
		}
	}

	if visited[step.name] {
		buf.WriteString(fmt.Sprintf("%s%s%s *\n", indentStr, debugColor(step.name), skipStr))
		return
	}

	visited[step.name] = true
	buf.WriteString(fmt.Sprintf("%s%s%s - %s\n", indentStr, debugColor(step.name), skipStr, step.description))

	for _, dep := range step.dependencies {
		if dep.name == step.name {
			continue
		}
		b.graph(dep, indent+2, buf, visited)
	}
}

func (b *Build) cyclesCheck() error {
	for name := range b.stepNames {
		visited := map[string]bool{}
		step := b.Step(name)
		if cycle := b.findCycle(visited, step); cycle != "" {
			return fmt.Errorf("%w: %s", ErrCycleDetected, cycle)
		}
	}
	return nil
}

func (b *Build) findCycle(visited map[string]bool, step *Step) string {
	if visited[step.name] {
		return step.name
	}
	visited[step.name] = true
	for _, dep := range step.dependencies {
		if found := b.findCycle(visited, dep); len(found) > 0 {
			return step.name + " -> " + found
		}
		visited[dep.name] = false
	}
	visited[step.name] = false
	return ""
}
