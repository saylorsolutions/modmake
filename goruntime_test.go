package modmake

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"strings"
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
	b.Build().AfterRun(Task(func(ctx context.Context) error {
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
	b.Build().AfterRun(IfNotExists("testingbuild/blah.exe", Task(func(ctx context.Context) error {
		return errors.New("failed to build blah.exe")
	})))
	b.Build().AfterRun(Task(func(ctx context.Context) error {
		output, err := exec.Command("testingbuild/blah.exe").Output()
		if err != nil {
			return err
		}
		expected := "Testing var: blah\n"
		assert.Equal(t, expected, string(output))
		return nil
	}))
	b.Build().AfterRun(Task(func(ctx context.Context) error {
		err := os.Remove("testingbuild/blah.exe")
		return err
	}))
	b.Execute("build", "package")
}

func TestGoBuild_Run(t *testing.T) {
	err := Go().Run("build.go", "--only", "build").WorkDir("example/helloworld").Run(context.TODO())
	assert.NoError(t, err)
}

func TestScanGoMod(t *testing.T) {
	cwd, err := os.Getwd()
	assert.NoError(t, err)
	_cwd := Path(cwd)

	git := _cwd.Join("package/git")
	root, found := scanGoMod(git)
	assert.True(t, found, "Should have found the root of the module")
	assert.Equal(t, _cwd.Join("go.mod").String(), root.String(), "Current working directory should be the root of the module")

	root, found = scanGoMod(_cwd)
	assert.True(t, found, "Should have found the root of the module")
	assert.Equal(t, _cwd.Join("go.mod").String(), root.String(), "Current working directory should be the root of the module")
}

func TestModDownload(t *testing.T) {
	modInfo, err := Go().modDownload(context.Background(), "github.com/saylorsolutions/modmake@v0.2.2")
	assert.NoError(t, err)
	assert.NotEmpty(t, modInfo.Dir)
	t.Logf("%#v", modInfo)
}

func TestGoTools_GetEnv(t *testing.T) {
	cmd := exec.Command("go", "env", "GOPATH")
	output, err := cmd.Output()
	assert.NoError(t, err)
	want := strings.TrimSpace(string(output))

	var got string
	assert.NotPanics(t, func() {
		got = Go().GetEnv("GOPATH")
	})
	assert.Equal(t, want, got)
}
