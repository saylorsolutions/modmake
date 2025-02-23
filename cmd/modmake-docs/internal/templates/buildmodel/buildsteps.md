A `Step` is something that may be invoked with either `go run` or the [modmake CLI](#modmake-cli), but may also
have [dependencies](https://pkg.go.dev/github.com/saylorsolutions/modmake#Step.DependsOn) and
[pre/post](https://pkg.go.dev/github.com/saylorsolutions/modmake#Step.BeforeRun) actions.

Step dependencies are arranged as a directed acyclic graph (a [DAG](https://en.wikipedia.org/wiki/Directed_acyclic_graph)).
If a cycle is detected during invocation — or while running the builtin `graph` step — then the Build will panic to include details of the error.

A step's [BeforeRun](https://pkg.go.dev/github.com/saylorsolutions/modmake#Step.BeforeRun) hooks will run in order _after_ dependency steps have executed in order.

  - Dependencies are good for orchestration and ensuring order of operations.
  - Before/After hooks are good for actions that are <em>intrinsic</em> to a `Step`'s execution.

### Default Steps

Here are the steps added by default to each `NewBuild`.
This is done to ensure a consistent base structure for every build.

  - `tools` - This step is for installing external tools that may be needed for the Build to function as expected.
  - `generate` - This step is for generating code (potentially with newly installed tools) that will be required for `test` and later steps. Depends on `tools`
  - `test` - This step should run unit tests in the project. Depends on `generate`.
  - `benchmark` - This step is skipped by default (it's not very often that these need to be run), but the step is here when required. Depends on `test`.
  - `build` - This step is for building the code 
  - `package` - This step is for packaging executables into an easily distributable/deployable format. 

> **Note:** the default build Steps do nothing by default. 
> A default Step's [Does](https://pkg.go.dev/github.com/saylorsolutions/modmake#Step.Does) or [DependsOn](https://pkg.go.dev/github.com/saylorsolutions/modmake#Step.DependsOn) method must be used to make it perform some operation.

### Utility Steps

  - `graph` - Prints a graph of steps and their dependencies.
  - `steps` - Prints a list of all steps in a build and their descriptions. Very greppable.
