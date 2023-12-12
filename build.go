package modmake

import (
	"errors"
	"fmt"
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

	workdir   string
	stepNames map[string]*Step
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
	test := newStep("test", "Runs unit tests on the code base").DependsOn(generate)
	bench := newStep("benchmark", "Runs benchmarking on the code base").DependsOn(test)
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
// os.Chdir will be called with the module root before go-running the build file, so the buildFile parameter should be relative to the module root.
// This is safe to use with Git submodules because a GoTools instance will be created based on the location of the Modmake build file and the closest module.
//
// CallBuild is preferable over [Build.Import] for building separate go modules.
// If you're building a component of the same go module, then use [Build.Import].
//
//   - buildRef should be the filesystem path to the build file
//   - args are flags and steps that should be executed in the build
func CallBuild(buildFile PathString, args ...string) *Command {
	gt := goToolsAt(buildFile)
	rel, err := gt.ModuleRoot().Rel(buildFile)
	if err != nil {
		panic(fmt.Sprintf("Unable to determine relative location to '%s' from module root path: %v", buildFile, err))
	}
	return gt.Run(rel.String(), args...).WorkDir(gt.ModuleRoot())
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

func (b *Build) AddStep(step *Step) {
	if step == nil {
		panic("nil step")
	}
	name := strings.ToLower(step.name)
	step.name = name
	if reservedStepNames[step.name] {
		panic(fmt.Errorf("step name '%s' is reserved", name))
	}
	if _, ok := b.stepNames[step.name]; ok {
		panic(fmt.Errorf("step name '%s' already exists", name))
	}
	b.stepNames[name] = step
	step.build = b
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
		if reservedStepNames[stepName] {
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
		buf.WriteString(fmt.Sprintf("%s%s%s *\n", indentStr, step.name, skipStr))
		return
	}

	visited[step.name] = true
	buf.WriteString(fmt.Sprintf("%s%s%s - %s\n", indentStr, step.name, skipStr, step.description))

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
