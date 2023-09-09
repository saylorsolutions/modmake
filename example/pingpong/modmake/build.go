package main

import (
	. "github.com/saylorsolutions/modmake"
)

func main() {
	b := NewBuild()
	client(b)
	server(b)
	b.Execute()
}

func client(b *Build) {
	build := NewStep("build-client", "Builds the client binary").
		Does(Go().Build("example/pingpong/client/main.go").OutputFilename("client"))

	run := NewStep("run-client", "Runs the client")
	run.Does(Go().Run("example/pingpong/client/main.go"))

	b.Build().DependsOn(build)
	b.AddStep(run)
}

func server(b *Build) {
	build := NewStep("build-server", "Builds the server binary").
		Does(Go().Build("example/pingpong/server/main.go").OutputFilename("server"))

	run := NewStep("run-server", "Runs the server")
	run.Does(Go().Run("example/pingpong/server/main.go"))

	b.Build().DependsOn(build)
	b.AddStep(run)
}
