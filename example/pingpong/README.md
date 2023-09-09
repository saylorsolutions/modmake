# Ping Pong

This example includes two binaries that can be build and run independently.

First, let's get the list of steps.

```shell
go run example/pingpong/build.go steps
```

```
benchmark - Runs benchmarking on the code base
build - Builds the code base and outputs an artifact
client:benchmark - Runs benchmarking on the code base
client:build - Builds the code base and outputs an artifact
client:generate - Generates code, possibly using external tools
client:package - Bundles one or more built artifacts into one or more distributable packages
client:run - Runs the client
client:test - Runs unit tests on the code base
client:tools - Installs external tools that will be needed later
generate - Generates code, possibly using external tools
package - Bundles one or more built artifacts into one or more distributable packages
server:benchmark - Runs benchmarking on the code base
server:build - Builds the code base and outputs an artifact
server:generate - Generates code, possibly using external tools
server:package - Bundles one or more built artifacts into one or more distributable packages
server:run - Runs the server
server:test - Runs unit tests on the code base
server:tools - Installs external tools that will be needed later
test - Runs unit tests on the code base
tools - Installs external tools that will be needed later
```

There are quite a few more steps in this build to accommodate building and running multiple artifacts.

Here's the graph.

```shell
go run example/pingpong/build.go graph
```

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
  -> client:build - Builds the code base and outputs an artifact
    -> client:benchmark (skip step) - Runs benchmarking on the code base
      -> client:test - Runs unit tests on the code base
        -> client:generate - Generates code, possibly using external tools
          -> client:tools - Installs external tools that will be needed later
  -> server:build - Builds the code base and outputs an artifact
    -> server:benchmark (skip step) - Runs benchmarking on the code base
      -> server:test - Runs unit tests on the code base
        -> server:generate - Generates code, possibly using external tools
          -> server:tools - Installs external tools that will be needed later
package - Bundles one or more built artifacts into one or more distributable packages
  -> build *
client:benchmark (skip step) *
client:build *
client:generate *
client:package - Bundles one or more built artifacts into one or more distributable packages
  -> client:build *
client:run - Runs the client
client:test *
client:tools *
server:benchmark (skip step) *
server:build *
server:generate *
server:package - Bundles one or more built artifacts into one or more distributable packages
  -> server:build *
server:run - Runs the server
server:test *
server:tools *

* - duplicate reference
```

A few things to note:
* Steps for both client and server builds are imported into a parent build with prefixes.
* The paths in the client and server builds are relative to the root of the repository.
* Running `build` will build both the client and server. This is done by making `build` depend on both `client:build` and `server:build`.
* Nothing depends on `client:run` and `server:run`. These were added with each `Build`'s `AddStep` method.

## Run the Programs

To see these applications running, we'll need to have two terminals open.

### First terminal

Let's start the server first.

```shell
go run example/pingpong/build.go server:run
```

You should see a message printed showing that the server has started.

### Second terminal

With the server running, the client can be started too.

```shell
go run example/pingpong/build.go client:run
```

You should see a PING and PONG line every second.

```
2023/08/31 00:01:43 [client:run] Running step...
2023/08/31 00:01:44 Starting ping-pong loop
2023/08/31 00:01:44 PING
2023/08/31 00:01:44 PONG
2023/08/31 00:01:45 PING
2023/08/31 00:01:45 PONG
2023/08/31 00:01:46 PING
2023/08/31 00:01:46 PONG
```

Press Ctrl+C in each terminal to send an interrupt signal which stops the client and server.

## Build the Artifacts

Now we can build both of our artifacts with one command.

```shell
go run example/pingpong/modmake/build.go build
```

After this runs, you should see both a client and server executable in the root of the repository.
