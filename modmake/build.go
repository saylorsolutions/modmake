package main

import (
	. "github.com/saylorsolutions/modmake"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	b := NewBuild()
	b.Test().Does(Go().TestAll())
	b.Build().DependsOnRunner("clean", "Removes previous build output if it exists",
		RemoveDir("build"),
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
	git = NewGitTools()
)

func cliBuild(os string, arch string) *Build {
	buildDirName := filepath.Join("build", os+"_"+arch)
	var target string
	if os == "windows" {
		target = filepath.Join(buildDirName, "modmake.exe")
	} else {
		target = filepath.Join(buildDirName, "modmake")
	}

	b := NewBuild()
	b.Test().Does(Go().TestAll()).Skip()

	b.Build().BeforeRun(MkdirAll(buildDirName, 0755))
	build := Go().Build(Go().ToModulePath("cmd/modmake")).
		OS(os).
		Arch(arch).
		StripDebugSymbols().
		OutputFilename(target).
		SetVariable("main", "gitHash", git.CommitHash()).
		SetVariable("main", "gitBranch", git.BranchName())
	b.Build().AfterRun(IfNotExists(target, Error("Failed to build modmake CLI for %s-%s", os, arch)))

	b.Build().Does(build)
	return b
}
