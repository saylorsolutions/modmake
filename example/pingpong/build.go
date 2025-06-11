package main

import (
	. "github.com/saylorsolutions/modmake" //nolint:staticcheck
)

func main() {
	// Create a new build model.
	b := NewBuild()

	// Import additional build models and merge them into this one.
	// Each imported build's steps can be referenced individually with their prefix.
	b.Import("client", client())
	b.Import("server", server())

	// Make sure that build executes without any previous output.
	b.Build().DependsOnRunner("clean", "Removes the build directory",
		RemoveDir("build"),
	)

	// The build step will execute the client and server build with a single Modmake invocation.
	b.Build().DependsOn(b.Step("client:build"))
	b.Build().DependsOn(b.Step("server:build"))
	b.Execute()
}

func client() *Build {
	// A module path is relative to the root of the module, and is something the Go toolchain understands.
	// Forward slashes are usable on all OS's.
	mainPath := Go().ToModulePath("./example/pingpong/client/main.go")
	b := NewBuild()
	b.Build().BeforeRun(Mkdir("build", 0755))
	b.Build().Does(
		Go().Build(mainPath).OutputFilename(Path("build/client_test")),
	)
	b.AddStep(NewStep("run", "Runs the client").Does(
		Go().Run(mainPath)),
	)
	return b
}

func server() *Build {
	// A module path is relative to the root of the module, and is something the Go toolchain understands.
	// Forward slashes are usable on all OS's.
	mainPath := Go().ToModulePath("./example/pingpong/server/main.go")
	b := NewBuild()
	b.Build().BeforeRun(Mkdir("build", 0755))
	b.Build().Does(
		Go().Build(mainPath).OutputFilename(Path("build/server_test")),
	)
	b.AddStep(NewStep("run", "Runs the server").Does(
		Go().Run(mainPath)),
	)
	return b
}
