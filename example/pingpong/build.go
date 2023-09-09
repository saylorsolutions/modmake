package main

import (
	. "github.com/saylorsolutions/modmake"
	client "github.com/saylorsolutions/modmake/example/pingpong/client/build"
	server "github.com/saylorsolutions/modmake/example/pingpong/server/build"
)

func main() {
	b := NewBuild()
	b.Import("client", client.Import())
	b.Import("server", server.Import())
	b.Build().DependsOn(b.Step("client:build"))
	b.Build().DependsOn(b.Step("server:build"))
	b.Execute()
}
