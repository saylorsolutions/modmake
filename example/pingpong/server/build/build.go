package build

import . "github.com/saylorsolutions/modmake"

func Import() *Build {
	mainPath := Go().ToModulePath("./example/pingpong/server/main.go")
	b := NewBuild()
	b.Build().Does(
		Go().Build(mainPath).OutputFilename("server_test"),
	)
	b.AddStep(NewStep("run", "Runs the server").Does(
		Go().Run(mainPath)),
	)
	return b
}
