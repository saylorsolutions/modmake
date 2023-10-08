package main

import (
	. "github.com/saylorsolutions/modmake"
	"github.com/saylorsolutions/modmake/pkg/git"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	b := NewBuild()
	b.Tools().Does(Go().ModTidy())
	b.Test().Does(Go().TestAll())
	b.Benchmark().Does(Go().BenchmarkAll())
	b.Build().DependsOnRunner("clean-build", "Removes previous build output if it exists",
		RemoveDir("build"),
	)
	b.Package().DependsOnRunner("clean-dist", "Removes previous distribution output if it exists",
		RemoveDir("dist"),
	)
	b.Package().AfterRun(RemoveDir("build"))

	b.AddStep(NewStep("clean", "Removes all output directories").
		DependsOn(b.Step("clean-build")).
		DependsOn(b.Step("clean-dist")),
	)

	buildVariants := map[string][]string{
		"windows": {
			"amd64",
		},
		"linux": {
			"amd64", "arm64",
		},
		"darwin": {
			"amd64", "arm64",
		},
	}

	for _os, variants := range buildVariants {
		for _, _arch := range variants {
			variant := _os + "_" + _arch
			b.Import(variant, cliBuild(_os, _arch))
			b.Build().DependsOn(b.Step(variant + ":build"))
			b.Package().DependsOn(b.Step(variant + ":package"))
		}
	}

	sysVariant := runtime.GOOS + "_" + runtime.GOARCH
	executable := filepath.Join("build", sysVariant, "modmake")
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	target := filepath.Join(home, "go", "bin", filepath.Base(executable))
	b.AddStep(NewStep("install", "Installs the modmake CLI to the user's $HOME/go/bin directory.").
		Does(CopyFile(executable, target)).
		DependsOn(b.Step(sysVariant + ":build")),
	)
	b.Execute()
}

var (
	_git = git.NewTools()
)

func cliBuild(os string, arch string) *Build {
	variant := os + "_" + arch
	buildDirName := filepath.Join("build", variant)
	var buildTarget string
	if os == "windows" {
		buildTarget = filepath.Join(buildDirName, "modmake.exe")
	} else {
		buildTarget = filepath.Join(buildDirName, "modmake")
	}

	b := NewBuild()
	b.Test().Does(Go().TestAll()).Skip()

	b.Build().BeforeRun(MkdirAll(buildDirName, 0755))
	build := Go().Build(Go().ToModulePath("cmd/modmake")).
		OS(os).
		Arch(arch).
		StripDebugSymbols().
		OutputFilename(buildTarget).
		SetVariable("main", "gitHash", _git.CommitHash()).
		SetVariable("main", "gitBranch", _git.BranchName())
	b.Build().AfterRun(IfNotExists(buildTarget, Error("Failed to build modmake CLI for %s-%s", os, arch)))
	b.Build().Does(build)

	pkgDirName := filepath.Join("dist")
	pkgPath := filepath.Join(pkgDirName, "modmake-"+variant)
	if os != "windows" {
		pkgPath += ".tar.gz"
		pkg := Tar(pkgPath)
		pkg.AddFileWithPath(buildTarget, "modmake")
		b.Package().Does(pkg.Create())
		b.Package().BeforeRun(MkdirAll(pkgDirName, 0755))
	} else {
		pkgPath += ".zip"
		pkg := Zip(pkgPath)
		pkg.AddFileWithPath(buildTarget, "modmake.exe")
		b.Package().Does(pkg.Create())
		b.Package().BeforeRun(MkdirAll(pkgDirName, 0755))
	}
	return b
}
