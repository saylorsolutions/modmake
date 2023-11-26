package main

import (
	. "github.com/saylorsolutions/modmake"
	"github.com/saylorsolutions/modmake/pkg/git"
)

const (
	version = "0.2.0"
)

var (
	_git = git.NewTools()
)

func main() {
	b := NewBuild()
	b.Tools().Does(Go().ModTidy())
	b.Test().Does(Go().TestAll())
	b.Benchmark().Does(Go().BenchmarkAll())
	b.Build().DependsOnRunner("clean-build", "", RemoveDir("build"))
	b.Package().DependsOnRunner("clean-dist", "", RemoveDir("dist"))

	a := NewAppBuild("modmake", Go().ToModulePath("cmd/modmake"), version).
		Build(func(gb *GoBuild) {
			gb.
				StripDebugSymbols().
				SetVariable("main", "gitHash", _git.CommitHash()).
				SetVariable("main", "gitBranch", _git.BranchName())
		})
	a.Variant("windows", "amd64")
	a.Variant("linux", "amd64")
	a.Variant("linux", "arm64")
	a.Variant("darwin", "amd64")
	a.Variant("darwin", "arm64")
	b.ImportApp(a)

	b.Execute()
}
