package build

import . "github.com/saylorsolutions/modmake"

func Import() *Build {
	b := NewBuild()
	b.Build().Does(
		Go().Build("example/pingpong/server/main.go").OutputFilename("server_test"),
	)
	b.AddStep(NewStep("run", "Runs the server").Does(
		Go().Run("example/pingpong/server/main.go")),
	)
	return b
}
