package build

import . "github.com/saylorsolutions/modmake"

func Import() *Build {
	b := NewBuild()
	b.Build().Does(
		Go().Build("example/pingpong/client/main.go").OutputFilename("client_test"),
	)
	b.AddStep(NewStep("run", "Runs the client").Does(
		Go().Run("example/pingpong/client/main.go")),
	)
	return b
}
