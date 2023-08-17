# ModMake

An attempt at creating a build system for Go using Go code.
Note that this is a work in progress, and probably shouldn't be used for anything serious yet.

## Goals and justification

* Reduce or eliminate the need for Makefiles for non-trivial builds.
  * I like Make, but I end up losing people when I try explaining `PHONY` and why it's a thing.
  * I really just want a stable task runner for my builds that doesn't have any OS/terminal dependencies, with helpful syntax to make it easy to use.
* Reduce or eliminate the need for scripting.
  * Scripting is great when everyone is on the same OS, but that's not always the case.
* As much as possible, make the common paths easy and discoverable.
  * This is super important too, because no one wants to look at a super bloated build script.
  * Only require what is needed, and allow the rest to be added by those that need it.
* Useful and consistent error reporting.
* Common sense base structure that isn't too restrictive.
* Allow adding custom logic where needed, while using the standard structure.
  * I really like this about Gradle, and I'd like to provide a similar user experience.
* Work avoidance for simpler cases.
* Allow creating "plugins" that modify a build in a predictable and reproducible way.
  * What I mean by this is *compile-time* plugins, which means that they can be go-got the same way any other dependency can.
  * Adding a bunch of plugins won't bloat any actual binaries unless they reference a build package for some reason.
* Make builds easily and usefully go-run-able.
  * This is important because it means that no specialized CLI tools are needed to get started, reducing the barrier to entry.
  * The build should still be able to be operated like it *were* a CLI tool, so flags, commands, and usage output are supported.
  * This allows build step documentation to have a real purpose, other than documenting the DSL. (Who looks at that, other than the person that wrote it?)
  * A nice win here is that this system can reuse the features of the go tool, meaning that the build system can benefit from all of the niceness that the Go team already provides and maintains.
