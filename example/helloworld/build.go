package main

import (
	. "github.com/saylorsolutions/modmake" //nolint:staticcheck
)

func main() {
	// Creates a new build model that may then be configured with whatever operations are needed.
	b := NewBuild()
	// Standard build steps have a dedicated Build method for referencing them.
	// This uses the Print task to print a message.
	b.Test().Does(Print("Testing..."))
	// This makes the build step print a different message. Hello!
	b.Build().Does(Print("Hello, modmake!"))
	// This is where the magic happens, and must be called to allow invoking your Modmake build.
	b.Execute()
}
