package main

import (
	"context"
	"fmt"
	. "github.com/saylorsolutions/modmake" //nolint:staticcheck
	"github.com/saylorsolutions/modmake/pkg/git"
)

const (
	version  = "0.6.0"
	docsPath = "/modmake"
	latestGo = 24
)

func main() {
	Go().PinLatestV1(latestGo)
	b := NewBuild()
	b.LintLatest().
		Target("./...", "./cmd/modmake", "./cmd/modmake-docs").
		EnableSecurityScanning().
		Enable("testifylint", "bidichk", "asasalint", "gocritic", "godox", "unparam")
	b.Generate().DependsOnRunner("tidy", "", tidyModules())
	b.Generate().DependsOnRunner("gen-docs", "",
		Script(
			CallBuild("./cmd/modmake-docs/modmake", "test"),
			Go().Run("./cmd/modmake-docs", "generate").
				Env("MD_BASE_PATH", docsPath).
				Env("MD_LATEST_GO", fmt.Sprintf("1.%d", latestGo)).
				Env("MD_SUPPORTED_GO", fmt.Sprintf("1.%d", latestGo-2)).
				Env("MD_MODMAKE_VERSION", "v"+version).
				Env("MD_GODOC_DIRS", ".,./pkg/git,./pkg/minify").
				Env("MD_GEN_DIR", "./docs"),
		),
	)
	b.Test().AfterRun(git.AssertNoChanges())
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

	b.AddNewStep("publish", "Publishes release tags from main", publishTags()).
		DependsOn(b.Package())

	b.Execute()
}

func tidyModules() Task {
	return Script(
		Go().ModTidy(),
		Go().WorkSync(),
	)
}

func publishTags() Task {
	return Script(
		Task(func(ctx context.Context) error {
			curBranch := git.BranchName()
			if curBranch != "main" {
				return fmt.Errorf("must publish from main, not %s", curBranch)
			}
			return nil
		}),
		git.Exec("tag", "v"+version),
		git.Exec("tag", "cmd/modmake/v"+version),
		git.Exec("push", "--tags"),
	)
}
