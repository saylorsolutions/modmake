# Modmake Build System

This is an attempt at creating a build system for Go using Go code and its toolchain.

For an in-depth description of how Modmake works, check out the documentation [here](https://saylorsolutions.github.io/modmake).

To report an issue or idea for improvement, click [here](https://github.com/saylorsolutions/modmake/issues/new/choose)

## Project Goals

There are a few goals I'm trying to accomplish with this system that may also explain why I didn't want to use existing systems that attempt to solve this problem.
If these goals aren't compatible with yours then Modmake may not be for you, and that's okay.

* Eliminate the need for Makefiles for non-trivial builds.
* Eliminate the need for OS-specific scripting.
* Make the common paths easy and discoverable.
* Informative and consistent error reporting.
* Common sense, consistent base structure that isn't too restrictive.
* Allow adding custom logic where needed.
* Avoid doing work that isn't needed.
* Easy to construct build logic that is invoked with `go run`.

## Benefits

Besides accomplishing the goals set out above, there are some key benefits with this approach.

* Modmake provides a DSL-like API that is simple to use and type-safe. All the power of Go code, but without the verbosity.
* Low barrier of entry with a consistent starting point.
* Go code works on many OS/architecture combinations, and Modmake inherits that ability.
* Modmake includes a lot of common-use functionality with more to come:
  * Using the Go toolchain, resolved from `GOROOT`.
  * File system operations like creating, copying, moving, deleting files and directories with [PathString](https://github.com/saylorsolutions/modmake/blob/main/pathstring.go).
  * Compressing and packaging with zip/tar.
  * `go install`ing and executing external tools.
  * Downloading files over HTTP.
  * Git operations like getting the current branch and commit hash.
  * Orchestrating build operations in terms of [build steps](https://saylorsolutions.github.io/modmake/#build-model_steps) and their dependencies.
