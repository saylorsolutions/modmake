package modmake

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestGoTools_Test(t *testing.T) {
	abs, err := filepath.Abs("./testingtest")
	assert.NoError(t, err)
	ctx := context.WithValue(context.Background(), CtxWorkdir, abs)
	assert.NoError(t, Go().TestAll().Run(ctx))
}

func TestGoTools_Benchmark(t *testing.T) {
	abs, err := filepath.Abs("./testingtest")
	assert.NoError(t, err)
	ctx := context.WithValue(context.Background(), CtxWorkdir, abs)
	assert.NoError(t, Go().BenchmarkAll().Run(ctx))
}

func TestGoTools_Build_File(t *testing.T) {
	test := Go().Test("testingtest/test_test.go")
	build := Go().NewBuild("main.go").
		ChangeDir("testingbuild").
		OutputFilename("blah.exe").
		Verbose().
		ForceRebuild().
		RaceDetector()
	b := NewBuild()
	b.Generate().Does(Go().GenerateAll())
	b.Test().Does(test)
	b.Build().Does(build)
	b.Build().AfterRun(IfNotExists("testingbuild/blah.exe", RunnerFunc(func(ctx context.Context) error {
		return errors.New("failed to build blah.exe")
	})))
	b.Build().AfterRun(RunnerFunc(func(ctx context.Context) error {
		err := os.Remove("testingbuild/blah.exe")
		return err
	}))
	assert.NoError(t, b.Execute("build", "package"))
}

func TestGoTools_Build_ModulePath(t *testing.T) {
	module := "github.com/saylorsolutions/modmake"
	test := Go().Test(module + "/testingtest")
	build := Go().NewBuild(module+"/testingbuild").
		OutputFilename("testingbuild/blah.exe").
		StripDebugSymbols().
		Tags("blah").
		SetVariable(module+"/main", "TestVar", "blah").
		RaceDetector()
	b := NewBuild()
	b.Generate().Does(Go().GenerateAll())
	b.Test().Does(test)
	b.Build().Does(build)
	b.Build().AfterRun(IfNotExists("testingbuild/blah.exe", RunnerFunc(func(ctx context.Context) error {
		return errors.New("failed to build blah.exe")
	})))
	b.Build().AfterRun(RunnerFunc(func(ctx context.Context) error {
		err := os.Remove("testingbuild/blah.exe")
		return err
	}))
	assert.NoError(t, b.Execute("build", "package"))
}
