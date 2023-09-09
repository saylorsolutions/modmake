package main

import (
	. "github.com/saylorsolutions/modmake"
)

func main() {
	b := NewBuild()
	b.Test().Does(Print("Testing..."))
	b.Build().Does(Print("Hello, modmake!"))
	b.Execute()
}
