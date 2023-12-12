package main

import (
	. "github.com/saylorsolutions/modmake"
)

func main() {
	b := NewBuild()
	b.Build().Does(Print("I'm a simulated submodule!"))
	b.Execute()
}
