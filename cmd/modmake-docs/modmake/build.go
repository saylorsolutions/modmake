package main

import (
	. "github.com/saylorsolutions/modmake"
	"github.com/saylorsolutions/modmake/pkg/git"
)

var (
	templVersion = F("v${TEMPL_VERSION:0.3.833}")
)

func main() {
	b := NewBuild()
	b.Tools().DependsOnRunner("install-templ", "", Go().Install("github.com/a-h/templ/cmd/templ@"+templVersion))
	b.Generate().DependsOnRunner("run-templ", "",
		Script(
			Exec("templ", "generate"),
			Go().VetAll(),
		))
	b.Test().Does(Go().TestAll().Silent())

	docs := NewAppBuild("modmake-docs", ".", "0.1.0")
	docs.Build(func(gb *GoBuild) {
		gb.StripDebugSymbols()
	})
	docs.HostVariant().NoPackage()
	b.ImportApp(docs)
	b.Step("modmake-docs:install").
		DependsOn(b.Test()).
		BeforeRun(Print("Executing from commit hash: %s", git.CommitHash()))

	b.AddNewStep("run", "serves the docs locally", Go().Run(".", "serve",
		F("--base-path=${MM_BASE_PATH:/modmake}"),
		F("--latest-go=${MM_LATEST_GO:1.23}"),
		F("--latest-supported=${MM_LATEST_GO_SUPPORTED:1.21}"),
		F("--modmake-version=v${MM_VERSION:0.4.4}"),
	)).DependsOn(b.Generate())

	b.Execute()
}
