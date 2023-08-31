package main

import (
	. "github.com/saylorsolutions/modmake"
	"log"
	"os"
)

func main() {
	b := NewBuild()
	client(b)
	server(b)
	if err := b.Execute(os.Args[1:]...); err != nil {
		log.Fatalf("Failed to execute build: %v\n", err)
	}
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
