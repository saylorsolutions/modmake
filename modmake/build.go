package main

import (
	. "github.com/saylorsolutions/modmake"
	"path/filepath"
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

	for os, variants := range buildVariants {
		for _, arch := range variants {
			variant := os + "_" + arch
			b.Import(variant, cliBuild(os, arch))
			b.Build().DependsOn(b.Step(variant + ":build"))
		}
	}

	b.Execute()
}

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
		OutputFilename(target)
	b.Build().AfterRun(IfNotExists(target, Error("Failed to build modmake CLI for %s-%s", os, arch)))

	b.Build().Does(build)
	return b
}
