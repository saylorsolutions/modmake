package modmake

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func ExampleBuild_Graph() {
	b := NewBuild()
	b.Tools().DependsOn(NewStep("print-tools", "").Does(RunnerFunc(func(ctx context.Context) error {
		fmt.Println("Running in tools")
		return nil
	})))
	b.Package().DependsOn(NewStep("print-pkg", "").Does(RunnerFunc(func(ctx context.Context) error {
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
	b.Tools().DependsOnFunc("print-tools", "", func(ctx context.Context) error {
		fmt.Println("Running in tools")
		return nil
	})
	b.Package().DependsOnFunc("print-pkg", "", func(ctx context.Context) error {
		fmt.Println("Running in package")
		return nil
	})
	if err := b.Execute("--skip-tools", "--skip-generate", "package", "print-tools"); err != nil {
		panic(err)
	}
	// Output:
	// Running in tools
	// Running in package
}

func ExampleBuild_Steps() {
	b := NewBuild()
	err := b.Execute("steps")
	if err != nil {
		panic(err)
	}
	// Output:
	// benchmark - Runs benchmarking on the code base
	// build - Builds the code base and outputs an artifact
	// generate - Generates code, possibly using external tools
	// package - Bundles one or more built artifacts into one or more distributable packages
	// test - Runs unit tests on the code base
	// tools - Installs external tools that will be needed later
}

func TestCyclesCheck(t *testing.T) {
	tests := map[string]struct {
		b func() *Build
	}{
		"Self-dependence": {
			b: func() *Build {
				b := NewBuild()
				b.Build().DependsOn(b.Build())
				return b
			},
		},
		"Direct cyclic": {
			b: func() *Build {
				b := NewBuild()
				b.Benchmark().DependsOn(b.Build())
				return b
			},
		},
		"Large cycle": {
			b: func() *Build {
				steps := make([]*Step, 1_000)
				for i := 0; i < 1_000; i++ {
					steps[i] = NewStep(strconv.FormatInt(int64(i), 10), "").Does(NoOp())
					if i > 0 {
						steps[i].DependsOn(steps[i-1])
					}
				}
				b := NewBuild()
				b.Tools().DependsOn(steps[999])
				steps[0].DependsOn(b.Package())
				return b
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			b := tc.b()
			err := b.cyclesCheck()
			t.Log(err)
			assert.ErrorIs(t, err, ErrCycleDetected)
		})
	}
}
