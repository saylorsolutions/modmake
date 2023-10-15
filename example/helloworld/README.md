# Hello World

This is a simple "hello world" build.
It can be used to demonstrate the basic features of a build, and how to get more information about it.

```go
package main

import (
  . "github.com/saylorsolutions/modmake"
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

test - Runs unit tests on the code base
  -> generate - Generates code, possibly using external tools
    -> tools - Installs external tools that will be needed later
benchmark (skip step) - Runs benchmarking on the code base
  -> test *
build - Builds the code base and outputs an artifact
  -> benchmark (skip step) *
package - Bundles one or more built artifacts into one or more distributable packages
  -> build *

* - duplicate reference
Use -v to print defined steps with no operation or dependent operation
```

There are a few things you might notice.
* The `test` step depends on `generate`, so it's listed below `test` with an arrow (`->`) next to it.
* The `generate` step has a dependency on `tools`, indicated by the arrow (`->`) under `generate`.
* The `tools` step has no dependencies and doesn't perform an operation, so it's not listed except as a dependency.
* The `benchmark` step is skipped by default.
    * If we look at the help output, we can see there is the option to enable it, `--run-benchmark`.

If you'd rather see an alphabetical list of steps, then you can run this from the root of the repository.

```shell
go run example/helloworld/build.go steps
```

You should see some output similar to this.

```
benchmark - Runs benchmarking on the code base
build - Builds the code base and outputs an artifact
package - Bundles one or more built artifacts into one or more distributable packages
test - Runs unit tests on the code base
```

> Note that `tools` is not shown, because it doesn't perform an operation.
> Re-run the command with `-v` to see *all* steps.
> `go run example/helloworld/build.go -v steps`

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

> To "unskip" a step, you could use either reference it directly as a step to run, or use `--no-skip step-name`.

Take a look at the [pingpong example](../pingpong/README.md) for a more interesting build, including custom steps and importing multiple builds.
