package modmake

import (
	"fmt"
	"sort"
	"strings"
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
	return b.toolsStep
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

func (b *Build) Graph() {
	visited := map[string]bool{}
	var buf strings.Builder
	buf.WriteString("Printing build graph\n\n")

	b.graph(b.toolsStep, 0, &buf, visited)
	b.graph(b.generateStep, 0, &buf, visited)
	b.graph(b.testStep, 0, &buf, visited)
	b.graph(b.benchStep, 0, &buf, visited)
	b.graph(b.buildStep, 0, &buf, visited)
	b.graph(b.packageStep, 0, &buf, visited)

	buf.WriteString("\n* - duplicate reference\n")
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
