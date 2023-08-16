package modmake

import (
	"context"
	"fmt"
)

func ExampleBuild_Graph() {
	b := NewBuild()
	b.Tools().DependsOn(NewStep("print-tools", "").Runs(RunnerFunc(func(ctx context.Context) error {
		fmt.Println("Running in tools")
		return nil
	})))
	b.Package().DependsOn(NewStep("print-pkg", "").Runs(RunnerFunc(func(ctx context.Context) error {
		fmt.Println("Running in package")
		return nil
	})))
	b.Graph()

	// Output:
	// Printing build graph
	//
	// tools - Installs external tools that will be needed later
	//   -> print-tools - No description
	// generate - Generates code, possibly using external tools
	//   -> tools *
	// test - Runs unit tests on the code base
	//   -> generate *
	// benchmark (skip step) - Runs benchmarking on the code base
	//   -> test *
	// build - Builds the code base and outputs an artifact
	//   -> benchmark (skip step) *
	// package - Bundles one or more built artifacts into one or more distributable packages
	//   -> build *
	//   -> print-pkg - No description
	//
	// * - duplicate reference
}

func ExampleBuild_Execute() {
	b := NewBuild()
	b.Tools().DependsOn(NewStep("print-tools", "").Runs(RunnerFunc(func(ctx context.Context) error {
		fmt.Println("Running in tools")
		return nil
	})))
	b.Package().DependsOn(NewStep("print-pkg", "").Runs(RunnerFunc(func(ctx context.Context) error {
		fmt.Println("Running in package")
		return nil
	})))
	if err := b.Execute([]string{"--skip-tools", "--skip-generate", "package", "print-tools", "print-pkg"}); err != nil {
		panic(err)
	}
	// Output:
	// Running in tools
	// Running in package
}

func ExampleBuild_Steps() {
	b := NewBuild()
	err := b.Execute([]string{"steps"})
	if err != nil {
		panic(err)
	}
	// Output:
	// benchmark
	// build
	// generate
	// package
	// test
	// tools
}
