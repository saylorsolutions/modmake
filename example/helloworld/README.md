# Hello World

This is a simple "hello world" build.
It can be used to demonstrate the basic features of a build, and how to get more information about it.

```go
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
```

We can get help information by passing `-h` or `--help`.

```
Executes this modmake build

Usage:
	go run BUILD_FILE.go graph
	go run BUILD_FILE.go steps
	go run BUILD_FILE.go [FLAGS] STEP...

There are specialized commands that can be used to introspect the build.
  - graph: Passing this command as the first argument will emit a step dependency graph with descriptions on standard out. This can also be generated with Build.Graph().
  - steps: Prints the list of all steps in this build.

See https://github.com/saylorsolutions/modmake for detailed usage information.

  -h, --help                Prints this usage information
      --run-benchmark       Runs the benchmark step
      --skip-dependencies   Skips running the named step's dependencies.
      --skip-generate       Skips the generate step, but not its dependencies.
      --skip-test           Skips the test step, but not its dependencies.
      --skip-tools          Skips the tools install step, but not its dependencies.
      --timeout duration    Sets a timeout duration for this build run
      --workdir string      Sets the working directory for the build (default ".")


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
package - Bundles one or more built artifacts into one or more distributable packages
  -> build *

* - duplicate reference
```

This includes a graph of the build's steps.
Notice that there are some steps added by default.
This is to provide a consistent structure to builds with a reasonable starting dependency graph that can be extended as needed.

To just get the build graph, run this from the root of the repository.

```shell
go run example/helloworld/build.go graph
```

You should see something like this.

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
package - Bundles one or more built artifacts into one or more distributable packages
  -> build *

* - duplicate reference
```

There are a few things you might notice.
* The `tools` step has no dependencies, so it's listed by itself.
* The `generate` step has a dependency on `tools`, indicated by the arrow (`->`) under `generate`.
    * The description for `tools` isn't printed again, because we've already encountered it before, indicated by the asterisk (`*`).
* The `benchmark` step is skipped by default.
    * If we look at the help output, we can see there is the option to enable it, `--run-benchmark`.

If you'd rather see an alphabetical list of steps, then you can run this.

```shell
go run example/helloworld/build.go steps
```

You should see some output similar to this.

```
benchmark - Runs benchmarking on the code base
build - Builds the code base and outputs an artifact
generate - Generates code, possibly using external tools
package - Bundles one or more built artifacts into one or more distributable packages
test - Runs unit tests on the code base
tools - Installs external tools that will be needed later
```

Let's run the `build` step to see this stuff work.

```shell
go run example/helloworld/build.go build
```

```
2023/08/30 23:06:37 [test] Running step...
2023/08/30 23:06:37 Testing...
2023/08/30 23:06:37 [test] Successfully ran step in 0s
2023/08/30 23:06:37 [benchmark] Skipping step
2023/08/30 23:06:37 [build] Running step...
2023/08/30 23:06:37 Hello, modmake!
2023/08/30 23:06:37 [build] Successfully ran step in 1ms
2023/08/30 23:06:37 Ran successfully in 1ms
```

If you see "Hello, modmake!", then you've run the build step.
Notice that the `benchmark` step is letting us know that it is skipped, and the `test` step ran without it being referenced in the command.
This is because `test` is a transitive dependency of `build` through `benchmark`.
When a step is skipped, its dependencies are not.

Take a look at the [pingpong example](../pingpong/README.md) for a more interesting build, including custom steps and multiple applications.
