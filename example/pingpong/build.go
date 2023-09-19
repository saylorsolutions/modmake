package main

import (
	. "github.com/saylorsolutions/modmake"
	"path/filepath"
)

func main() {
	b := NewBuild()
	b.Import("client", client())
	b.Import("server", server())
	b.Build().DependsOnTask("clean", "Removes the build directory",
		RemoveDir("build"),
	)
	b.Build().DependsOn(b.Step("client:build"))
	b.Build().DependsOn(b.Step("server:build"))
	b.Execute()
}

func client() *Build {
	mainPath := Go().ToModulePath("./example/pingpong/client/main.go")
	b := NewBuild()
	b.Build().BeforeRun(Mkdir("build", 0755))
	b.Build().Does(
		Go().Build(mainPath).OutputFilename(filepath.Join("build", "client_test")),
	)
	b.AddStep(NewStep("run", "Runs the client").Does(
		Go().Run(mainPath)),
	)
	return b
}

func server() *Build {
	mainPath := Go().ToModulePath("./example/pingpong/server/main.go")
	b := NewBuild()
	b.Build().BeforeRun(Mkdir("build", 0755))
	b.Build().Does(
		Go().Build(mainPath).OutputFilename(filepath.Join("build", "server_test")),
	)
	b.AddStep(NewStep("run", "Runs the server").Does(
		Go().Run(mainPath)),
	)
	return b
}
