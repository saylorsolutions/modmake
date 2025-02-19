Modmake build logic can be composed of many pieces, and there are a few approaches to this.

### Import

The [Import](https://pkg.go.dev/github.com/saylorsolutions/modmake#Build.Import) function is one of the main methods used.

* A build can be divided into multiple *sub-builds* that are then imported together.
* A build must be imported with a name that becomes its prefix.
* The prefix is important to prevent collisions between different builds' step names.

```go
b := NewBuild()

other := NewBuild()
other.Test().Does(Go().TestAll())
b.Import("other", other)

// Now we can reference the other build with the prefix "other:".
b.Step("other:test")

b.Execute() // Because "other:test" is now an established step, it can be invoked with "go run" too.
```

A variant of `Import` is [ImportAndLink](https://pkg.go.dev/github.com/saylorsolutions/modmake#Build.ImportAndLink), which intrinsically links the sub-build steps to that of its parent.

### CallBuild

Another mechanism is [CallBuild](https://pkg.go.dev/github.com/saylorsolutions/modmake#CallBuild), which allows invoking steps in an unrelated build.
This is a very useful mechanism when working with Git submodules that use modmake.

There's an [example](https://pkg.go.dev/github.com/saylorsolutions/modmake#example-CallBuild) for reference.

### CallRemote

This is a more niche method, but it's still nice to have.

`CallRemote` is used to call a Modmake step in a remote build that is in no way associated with your build.

**This is used in the [Modmake build](https://github.com/saylorsolutions/modmake/blob/main/modmake/build.go) to generate this documentation!**

```go
b.Tools().DependsOnRunner("install-modmake-docs", "",
	CallRemote("github.com/saylorsolutions/modmake-docs@latest", "modmake/build.go", "modmake-docs:install"))

/* ... */

b.Generate().DependsOnRunner("gen-docs", "", Exec("modmake-docs", "generate").
	// Exec configuration truncated for brevity...
)
```

### AppBuild

[AppBuild](https://pkg.go.dev/github.com/saylorsolutions/modmake#AppBuild) is a higher level concept that allows for a lot less typing and more convenience.
This is a good fit when an app's build follows a common way of producing distributions.

`AppBuild` is great for any of these cases.

* You want to produce multiple variants of the same application for different OS/arch combinations.
* You want an easy way to customize build/packaging logic when it's needed, and rely on defaults otherwise.
* You don't want to deal with managing build vs. distribution paths yourself.
* You want a generated `install` step for your app to be used with `CallRemote`.

It follows a similar pattern as a normal build, but it's specifically tailored to producing builds of the same application for different OS/arch combinations with similar expectations.
Each combination of OS and arch in an `AppBuild` is called a variant.
They can be individually customized or their build step can be configured at the `AppBuild` level.

```go
a := NewAppBuild("modmake", "cmd/modmake", version).
    Build(func(gb *GoBuild) {
        gb.
            StripDebugSymbols().
            SetVariable("main", "gitHash", git.CommitHash()).
            SetVariable("main", "gitBranch", git.BranchName())
    })

// HostVariant for local testing.
a.HostVariant()

// The same build logic will be applied to all variants.
// The default packaging will be used.
a.Variant("windows", "amd64")
a.Variant("linux", "amd64")
a.Variant("linux", "arm64")
a.Variant("darwin", "amd64")
a.Variant("darwin", "arm64")
b.ImportApp(a)
```

* A new `AppBuild` is created with some basic information: name, path, and version.
* The `AppBuild` above is using a consistent build step configured at the higher level (variants can override this logic).
* Packaging is not being configured above, so the defaults are used.
  * For Windows builds, the default is to package the binary in a zip file.
  * For everything else, the binary is packaged in a tar.gz.
* There's also a `HostVariant` that will match your build machine's OS and arch. This is most useful for local testing.
* An additional step will be generated automatically for each AppBuild: `install`. To reference this step, prefix it with the AppBuild name. `modmake:install` would be used in the code above.
