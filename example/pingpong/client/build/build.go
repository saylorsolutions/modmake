package build

import . "github.com/saylorsolutions/modmake"

func Import() *Build {
	mainPath := Go().ToModulePath("example/pingpong/client/main.go")
	b := NewBuild()
	b.Build().Does(
		Go().Build(mainPath).OutputFilename("client_test"),
	)
	b.AddStep(NewStep("run", "Runs the client").Does(
		Go().Run(mainPath)),
	)
	return b
}
