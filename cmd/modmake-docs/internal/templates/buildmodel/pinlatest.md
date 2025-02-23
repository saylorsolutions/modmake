For consistent builds, it's recommended to use `Go().PinLatestV1`

This call will download and pin the build to the latest v1 patch version of the Go toolchain, given a specific minor version.

This should be used at the top of the `main` function of your build to ensure it takes effect before any steps are executed.

```go
func main() {
    // This will pin to the latest patch version of 1.22, complete with any security patches that have been released.
    Go().PinLatestV1(22)
	
    b := NewBuild()
    // Remaining build logic...
}
```
