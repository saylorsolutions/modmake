package main

import (
	. "github.com/saylorsolutions/modmake"
	"log"
	"os"
)

func main() {
	b := NewBuild()
	b.Test().Does(Print("Testing..."))
	b.Build().Does(Print("Hello, modmake!"))

	if err := b.Execute(os.Args[1:]...); err != nil {
		log.Fatalf("Failed to execute build: %v\n", err)
	}
}
