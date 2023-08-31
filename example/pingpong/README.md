# Ping Pong

This example includes two binaries that can be build and run independently.

First, let's get the list of steps.

```
benchmark - Runs benchmarking on the code base
build - Builds the code base and outputs an artifact
build-client - Builds the client binary
build-server - Builds the server binary
generate - Generates code, possibly using external tools
package - Bundles one or more built artifacts into one or more distributable packages
run-client - Runs the client
run-server - Runs the server
test - Runs unit tests on the code base
tools - Installs external tools that will be needed later
```

There are a few more steps in this build to accommodate building and running multiple artifacts.

Here's the graph.

```
Printing build graph

tools - Installs external tools that will be needed later
generate - Generates code, possibly using external tools
  -> tools *
test - Runs unit tests on the code base
  -> generate *
benchmark (skip step) - Runs benchmarking on the code base
  -> test *
build - Builds the code base and outputs an artifact
  -> benchmark (skip step) *
  -> build-client - Builds the client binary
  -> build-server - Builds the server binary
package - Bundles one or more built artifacts into one or more distributable packages
  -> build *
build-client *
build-server *
run-client - Runs the client
run-server - Runs the server

* - duplicate reference
```

A few things to note:
* Running `build` will build both the client and server. This is done by making `build` depend on both `build-client` and `build-server`.
* Nothing depends on `run-client` and `run-server`. These were added with the `Build`'s `AddStep` method.

## Run the Programs

To see these applications running, we'll need to have two terminals open.

**First terminal**
Let's start the server first.

```shell
go run example/pingpong/modmake/build.go run-server
```

You should see a message printed showing that the server has started.

**Second terminal**
Now for the client.

```shell
go run example/pingpong/modmake/build.go run-client
```

You should see a PING and PONG line every second.

```
2023/08/31 00:01:43 [run-client] Running step...
2023/08/31 00:01:44 Starting ping-pong loop
2023/08/31 00:01:44 PING
2023/08/31 00:01:44 PONG
2023/08/31 00:01:45 PING
2023/08/31 00:01:45 PONG
2023/08/31 00:01:46 PING
2023/08/31 00:01:46 PONG
```

Press Ctrl+C to send an interrupt to the client and server to stop them.

## Build the Artifacts

Now we can build both of our artifacts with one command.

```shell
go run example/pingpong/modmake/build.go build
```

After this runs, you should see both a client and server executable in the root of the repository.
