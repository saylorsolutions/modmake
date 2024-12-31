package main

import (
	"context"
	"fmt"
	. "github.com/saylorsolutions/modmake"
	"github.com/saylorsolutions/modmake/pkg/git"
)

const (
	version  = "0.4.4"
	docsPath = "/modmake"
	latestGo = 23
)

func main() {
	Go().PinLatestV1(latestGo)
	b := NewBuild()
	b.Tools().DependsOnRunner("install-modmake-docs", "",
		TempDir("modmake-docs-*", func(tmp PathString) Task {
			return Script(
				git.CloneAt("https://github.com/saylorsolutions/modmake-docs.git", tmp),
				Task(func(ctx context.Context) error {
					return CallBuild(tmp.Join("modmake"), "modmake-docs:install").Run(ctx)
				}),
			)
		}),
	)
	b.Generate().DependsOnRunner("mod-tidy", "", Go().ModTidy())
	b.Generate().DependsOnRunner("gen-docs", "",
		Exec("modmake-docs", "generate").
			Arg("--base-path="+docsPath).
			Arg(fmt.Sprintf("--latest-go=1.%d", latestGo)).
			Arg(fmt.Sprintf("--latest-supported=1.%d", latestGo-2)).
			Arg("--modmake-version=v"+version).
			WorkDir("./docs"),
	)
	b.Benchmark().Does(Go().BenchmarkAll())
	b.Build().DependsOnRunner("clean-build", "", RemoveDir("build"))
	b.Package().DependsOnRunner("clean-dist", "", RemoveDir("dist"))

	a := NewAppBuild("modmake", "cmd/modmake", version).
		Build(func(gb *GoBuild) {
			gb.
				StripDebugSymbols().
				SetVariable("main", "gitHash", git.CommitHash()).
				SetVariable("main", "gitBranch", git.BranchName()).
				SetVariable("main", "runtimeVersion", version)
		})
	a.HostVariant()
	a.Variant("windows", "amd64")
	a.Variant("linux", "amd64")
	a.Variant("linux", "arm64")
	a.Variant("darwin", "amd64")
	a.Variant("darwin", "arm64")
	b.ImportApp(a)

	b.Execute()
}
