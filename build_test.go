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
	b.Tools().DependsOn(NewStep("print-tools", "").Does(Task(func(ctx context.Context) error {
		fmt.Println("Running in tools")
		return nil
	})))
	b.Package().DependsOn(NewStep("print-pkg", "").Does(Task(func(ctx context.Context) error {
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
	// print-pkg *
	// print-tools *
	//
	// * - duplicate reference
}

func ExampleBuild_Execute() {
	var (
		ranTools    bool
		ranGenerate bool
	)

	b := NewBuild()
	b.Tools().DependsOnFunc("print-tools", "", func(ctx context.Context) error {
		fmt.Println("Running in tools")
		return nil
	})
	b.Tools().Does(Task(func(ctx context.Context) error {
		ranTools = true
		return nil
	}))
	b.Generate().Does(Task(func(ctx context.Context) error {
		ranGenerate = true
		return nil
	}))
	b.Package().DependsOnFunc("print-pkg", "", func(ctx context.Context) error {
		fmt.Println("Running in package")
		return nil
	})
	b.Execute("--skip", "tools", "--skip", "generate", "package", "print-tools")
	fmt.Println("Ran tools:", ranTools)
	fmt.Println("Ran generate:", ranGenerate)
	// Output:
	// Running in tools
	// Running in package
	// Ran tools: false
	// Ran generate: false
}

func ExampleBuild_Steps() {
	b := NewBuild()
	b.Execute("steps")
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
		b        func() *Build
		noCycles bool
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
		"Dual dependency": {
			b: func() *Build {
				b := NewBuild()
				b.Build().DependsOnRunner("echo", "Prints a message", Print("a message"))
				b.Package().DependsOn(b.Step("echo"))
				return b
			},
			noCycles: true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			b := tc.b()
			err := b.cyclesCheck()
			if tc.noCycles {
				assert.NoError(t, err)
			} else {
				t.Log(err)
				assert.ErrorIs(t, err, ErrCycleDetected)
			}
		})
	}
}

func TestCallBuild(t *testing.T) {
	err := CallBuild("example/helloworld", "build.go", "--only", "build").Run(context.TODO())
	assert.NoError(t, err)
}

func BenchmarkLargeCycle_1000(b *testing.B) {
	build := func() *Build {
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
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = build.cyclesCheck()
	}
}

func BenchmarkLargeCycle_10000(b *testing.B) {
	build := func() *Build {
		steps := make([]*Step, 10_000)
		for i := 0; i < 10_000; i++ {
			steps[i] = NewStep(strconv.FormatInt(int64(i), 10), "").Does(NoOp())
			if i > 0 {
				steps[i].DependsOn(steps[i-1])
			}
		}
		b := NewBuild()
		b.Tools().DependsOn(steps[9999])
		steps[0].DependsOn(b.Package())
		return b
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = build.cyclesCheck()
	}
}

func TestBuild_Import(t *testing.T) {
	b := NewBuild()
	other := NewBuild()
	assert.NotPanics(t, func() {
		other.AddStep(NewStep("print", "Prints a message").Does(Print("Printing!")))
		other.Build().DependsOn(other.Step("print"))
	}, "New step creation should not panic")
	b.Import("other", other)
	_, ok := b.stepNames["other:print"]
	assert.True(t, ok, "Step 'other:print' should have been imported")
	_, ok = b.stepNames["print"]
	assert.False(t, ok, "Other 'print' step should not have been imported")
}
