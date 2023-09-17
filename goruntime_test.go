package modmake

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"testing"
)

func TestGoTools_Test(t *testing.T) {
	assert.NoError(t, Go().TestAll().WorkDir("./testingtest").Run(context.Background()))
}

func TestGoTools_Benchmark(t *testing.T) {
	assert.NoError(t, Go().BenchmarkAll().WorkDir("./testingtest").Run(context.Background()))
}

func TestGoTools_Build_File(t *testing.T) {
	test := Go().Test("testingtest/test_test.go")
	build := Go().Build("main.go").
		ChangeDir("testingbuild").
		OutputFilename("blah.exe").
		ForceRebuild().
		RaceDetector()

	b := NewBuild()
	b.Generate().Does(Go().GenerateAll())
	b.Test().Does(test)
	b.Build().Does(build)
	b.Build().AfterRun(IfNotExists("testingbuild/blah.exe", Error("failed to build blah.exe")))
	b.Build().AfterRun(RunFunc(func(ctx context.Context) error {
		err := os.Remove("testingbuild/blah.exe")
		return err
	}))
	b.Execute("package")
}

func TestGoTools_Build_ModulePath(t *testing.T) {
	test := Go().Test(Go().ToModulePath("testingtest"))
	build := Go().Build(Go().ToModulePath("testingbuild")).
		OutputFilename("testingbuild/blah.exe").
		StripDebugSymbols().
		Tags("blah").
		SetVariable("main", "testVar", "blah").
		Verbose()
	b := NewBuild()
	b.Generate().Does(Go().GenerateAll())
	b.Test().Does(test)
	b.Build().Does(build)
	b.Build().AfterRun(IfNotExists("testingbuild/blah.exe", RunFunc(func(ctx context.Context) error {
		return errors.New("failed to build blah.exe")
	})))
	b.Build().AfterRun(RunFunc(func(ctx context.Context) error {
		output, err := exec.Command("testingbuild/blah.exe").Output()
		if err != nil {
			return err
		}
		expected := "Testing var: blah\n"
		assert.Equal(t, expected, string(output))
		return nil
	}))
	b.Build().AfterRun(RunFunc(func(ctx context.Context) error {
		err := os.Remove("testingbuild/blah.exe")
		return err
	}))
	b.Execute("build", "package")
}

func TestGoBuild_Run(t *testing.T) {
	err := Go().Run("build.go", "--only", "build").WorkDir("example/helloworld").Run(context.TODO())
	assert.NoError(t, err)
}
