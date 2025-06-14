package modmake

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/saylorsolutions/cache"
)

// GoTools provides some utility functions for interacting with the go tool chain.
// To get an instance of GoTools, use the Go function.
// Upon first invocation, Go will cache filesystem and module details for reference later.
//
// To invalidate this cache, use [GoTools.InvalidateCache].
//
// If your build logic needs to cross into another go module, try using CallBuild.
type GoTools struct {
	goName     string
	goRootPath *cache.Value[string]
	goModPath  *cache.Value[PathString]
	moduleName *cache.Value[string]
}

var (
	_goMux      sync.RWMutex
	_goInstance *GoTools
)

// Go will retrieve or initialize an instance of GoTools.
// This indirection is desirable to support caching of tool chain, filesystem, and module details.
// This function is concurrency safe, and may be called by multiple goroutines if desired.
func Go() *GoTools {
	_goMux.RLock()
	inst := _goInstance
	_goMux.RUnlock()
	if inst != nil {
		return inst
	}
	var err error
	inst, err = initGoInst()
	if err != nil {
		panic(err)
	}
	_goInstance = inst
	return inst
}

func goToolsAt(path PathString) *GoTools {
	gt, err := goToolsAtErr(path)
	if err != nil {
		panic(err)
	}
	return gt
}

func goToolsAtErr(path PathString) (*GoTools, error) {
	if path.IsFile() {
		path = path.Dir()
	}
	goModPath, found := scanGoMod(path)
	if !found {
		return nil, fmt.Errorf("unable to locate go.mod from path '%s'", path.String())
	}
	modName, err := moduleNameLookup(goModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse module name from file '%s': %v", goModPath, err)
	}
	return &GoTools{
		goName:     Go().goName,
		goRootPath: Go().goRootPath,
		goModPath: cache.New(func() (PathString, error) {
			return goModPath, nil
		}),
		moduleName: cache.New[string](func() (string, error) {
			return modName, nil
		}),
	}, nil
}

func initGoInst() (*GoTools, error) {
	return initGoInstNamed("go")
}

func initGoInstNamed(goName string) (*GoTools, error) {
	_goMux.Lock()
	defer _goMux.Unlock()
	if _goInstance != nil {
		return _goInstance, nil
	}
	goRootPath := cache.New(func() (string, error) {
		errMsg := fmt.Sprintf("unable to resolve GOROOT, '%s' may not be installed correctly or on the PATH", goName)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var output strings.Builder
		err := Exec(goName, "env", "GOROOT").Silent().Stdout(&output).LogGroup("go env").Run(ctx)
		if err != nil {
			return "", fmt.Errorf("%s: %v", errMsg, err)
		}
		goRootDir := strings.TrimSpace(output.String())
		if len(goRootDir) == 0 {
			// Reasonable fallback for when we can't get GOROOT
			if goName != "go" {
				return "", nil
			}
			return "", errors.New(errMsg)
		}
		return goRootDir, nil
	})
	modPath := cache.New(func() (PathString, error) {
		modPath, err := locateModRoot()
		if err != nil {
			return "", err
		}
		return modPath, nil
	})
	moduleName := cache.New(func() (string, error) {
		path, err := modPath.Get()
		if err != nil {
			return "", err
		}
		return moduleNameLookup(path)
	})
	inst := &GoTools{
		goName:     goName,
		goRootPath: goRootPath,
		goModPath:  modPath,
		moduleName: moduleName,
	}
	_goInstance = inst
	return inst, nil
}

func moduleNameLookup(goModPath PathString) (string, error) {
	moduleName, err := parseModuleName(goModPath)
	if err != nil {
		return "", err
	}
	return moduleName, nil
}

// InvalidateCache will break the instance cache, forcing the next call to Go to scan the filesystem's information again.
func (g *GoTools) InvalidateCache() {
	_goMux.Lock()
	defer _goMux.Unlock()
	_goInstance = nil
}

func (g *GoTools) goTool() string {
	// Override for when we already have an absolute path to the executable.
	if g.goName != "go" {
		return g.goName
	}
	goRootPath, err := g.goRootPath.Get()
	if err != nil {
		panic(fmt.Sprintf("failed to get GOROOT: %v", err))
	}
	return filepath.Join(goRootPath, "bin", "go")
}

// GetEnv will call "go env $key", and return the value of the named environment variable.
// If an error occurs, then the call will panic.
//
// If the value of GOBIN is requested, then GoTools.GOBIN will be invoked instead.
func (g *GoTools) GetEnv(key string) string {
	if key == "GOBIN" {
		return g.GOBIN().String()
	}
	return g.getEnv(key)
}

func (g *GoTools) getEnv(key string) string {
	var output bytes.Buffer
	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()
	err := g.Command("env", key).CaptureStdin().Stdout(&output).Run(timeout)
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(output.String())
}

// GOBIN resolves the installation path of tools the same way that 'go install' does for installing them, allowing for more consistent tool invocation behavior.
//
// The order of resolution is as follows:
//
//   - If the GOBIN environment variable is defined, then this path will be returned.
//   - If the GOPATH environment variable is defined, then "$GOPATH/bin" will be returned.
//   - If the user's home directory can be resolved, then the "go/bin" path relative to the user's home directory will be returned.
//   - If none of the above paths can be returned, then the current working directory will be returned.
//
// See 'go help install' for more details.
func (g *GoTools) GOBIN() PathString {
	gobin := g.getEnv("GOBIN")
	if len(gobin) > 0 {
		return Path(gobin)
	}
	// Fall back to $GOPATH/bin
	gopath := g.GetEnv("GOPATH")
	if len(gopath) > 0 {
		return Path(gopath, "bin")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Unable to resolve user's home directory, fall back to current working directory.
		cwd, err := os.Getwd()
		if err != nil {
			return Path(".")
		}
		return Path(cwd)
	}
	return Path(home, "go/bin")
}

// ModuleRoot returns a filesystem path to the root of the current module.
func (g *GoTools) ModuleRoot() PathString {
	return g.goModPath.MustGet().Dir()
}

// ModuleName returns the name of the current module as specified in the go.mod.
func (g *GoTools) ModuleName() string {
	return g.moduleName.MustGet()
}

// ToModulePackage is specifically provided to construct a package reference for [GoBuild.SetVariable] by prepending the module name to the package name, separated by '/'.
// This is not necessary for setting variables in the main package, as 'main' can be used instead.
//
//	// When run in a module named 'example.com/me/myproject',
//	// this will output 'example.com/me/myproject/other'.
//	Go().ToModulePackage("other")
//
// See [GoBuild.SetVariable] for more details.
func (g *GoTools) ToModulePackage(pkg string) string {
	return g.ModuleName() + "/" + pkg
}

// ToModulePath takes a path to a file or directory within the module, relative to the module root, and translates it to a module path.
// If a path to a file is given, then a module path to the file's parent directory is returned.
// If ToModulePath is unable to stat the given path, then this function will panic.
//
// For example, given a module name of 'github.com/example/mymodule', and a relative path of 'app/main.go', the module path 'github.com/example/mymodule/app' is returned.
func (g *GoTools) ToModulePath(dir string) string {
	test := g.ModuleRoot().Join(dir)
	fi, err := test.Stat()
	if err != nil {
		panic(fmt.Errorf("unable to stat path '%s': %w", test, err))
	}
	if !fi.IsDir() {
		dir = filepath.Dir(dir)
	}
	return g.moduleName.MustGet() + "/" + filepath.ToSlash(dir)
}

func (g *GoTools) Command(command string, args ...string) *Command {
	return Exec(g.goTool(), command).Arg(args...).LogGroup("go " + command)
}

type GoClean struct {
	*Command
	buildCache bool
	testCache  bool
	modCache   bool
	fuzzCache  bool
}

func (g *GoTools) Clean() *GoClean {
	return &GoClean{
		Command: g.Command("clean"),
	}
}

func (c *GoClean) BuildCache() *GoClean {
	c.buildCache = true
	return c
}

func (c *GoClean) TestCache() *GoClean {
	c.testCache = true
	return c
}

func (c *GoClean) ModCache() *GoClean {
	c.modCache = true
	return c
}

func (c *GoClean) FuzzCache() *GoClean {
	c.fuzzCache = true
	return c
}

func (c *GoClean) Run(ctx context.Context) error {
	c.OptArg(c.buildCache, "-cache")
	c.OptArg(c.testCache, "-testcache")
	c.OptArg(c.modCache, "-modcache")
	c.OptArg(c.fuzzCache, "-fuzzcache")
	if err := c.Command.Run(ctx); err != nil {
		return err
	}
	return nil
}

func (c *GoClean) Task() Task {
	return c.Run
}

func (g *GoTools) Install(pkg string) *Command {
	return g.Command("install", pkg)
}

func (g *GoTools) Get(pkg string) *Command {
	return g.Command("get").Arg(pkg)
}

func (g *GoTools) GetUpdated(pkg string) *Command {
	return g.Command("get", "-u").Arg(pkg)
}

// ModTidy will download missing module cache information, and tidy up dependency details.
func (g *GoTools) ModTidy() *Command {
	return g.Command("mod", "tidy")
}

// WorkSync will synchronize dependencies between modules in a Go workspace project.
func (g *GoTools) WorkSync() *Command {
	return g.Command("work", "sync")
}

// moduleInfo is a struct used to capture the JSON output of a module download.
type moduleInfo struct {
	Path     string // module path
	Query    string // version query corresponding to this version
	Version  string // module version
	Error    string // error loading module
	Info     string // absolute path to cached .info file
	GoMod    string // absolute path to cached .mod file
	Zip      string // absolute path to cached .zip file
	Dir      string // absolute path to cached source root directory
	Sum      string // checksum for path, version (as in go.sum)
	GoModSum string // checksum for go.mod (as in go.sum)
	Origin   any    // provenance of module
	Reuse    bool   // reuse of old module info is safe
}

func (g *GoTools) modDownload(ctx context.Context, module string) (moduleInfo, error) {
	var (
		output bytes.Buffer
		mod    moduleInfo
	)
	if err := g.Command("mod", "download", "-json", module).CaptureStdin().Stdout(&output).Run(ctx); err != nil {
		return moduleInfo{}, fmt.Errorf("failed to download module: %w", err)
	}
	if err := json.NewDecoder(&output).Decode(&mod); err != nil {
		return moduleInfo{}, fmt.Errorf("failed to decode module info: %w", err)
	}
	return mod, nil
}

func (g *GoTools) Test(patterns ...string) *Command {
	return g.Command("test", "-v").Arg(patterns...)
}

func (g *GoTools) TestAll() *Command {
	return g.Test("./...")
}

func (g *GoTools) Generate(patterns ...string) *Command {
	return g.Command("generate").TrailingArg(patterns...)
}

func (g *GoTools) GenerateAll() *Command {
	return g.Generate("./...")
}

func (g *GoTools) Benchmark(pattern string) *Command {
	return g.Command("test", "-bench="+pattern, "-run=^$", "-v", "./...")
}

func (g *GoTools) BenchmarkAll() *Command {
	return g.Benchmark(".")
}

func (g *GoTools) Vet(patterns ...string) *Command {
	return g.Command("vet").TrailingArg(patterns...)
}

func (g *GoTools) VetAll() *Command {
	return g.Vet("./...")
}

func (g *GoTools) Format(patterns ...string) *Command {
	return g.Command("fmt", patterns...)
}

func (g *GoTools) FormatAll() *Command {
	return g.Format("./...")
}

type GoBuild struct {
	err           error
	cmd           *Command
	output        PathString
	forceRebuild  bool
	dryRun        bool
	detectRace    bool
	memorySan     bool
	addrSan       bool
	printPackages bool
	printCommands bool
	stripDebug    bool
	trimPath      bool
	buildMode     string
	gcFlags       map[string]bool
	ldFlags       map[string]bool
	tags          map[string]bool
	targets       map[string]bool
	private       []string
}

// Build creates a GoBuild to hold parameters for a new go build run.
func (g *GoTools) Build(targets ...string) *GoBuild {
	if len(targets) == 0 {
		return &GoBuild{
			cmd: &Command{err: errors.New("no targets defined")},
		}
	}
	_targets := map[string]bool{}
	for _, target := range targets {
		target = strings.TrimSpace(target)
		if len(target) > 0 {
			_targets[target] = true
		}
	}
	return &GoBuild{
		cmd:     g.Command("build"),
		targets: _targets,
		gcFlags: map[string]bool{},
		ldFlags: map[string]bool{},
		tags:    map[string]bool{},
	}
}

// ChangeDir will change the working directory from the default to a new location.
// If an absolute path cannot be derived from newDir, then this function will panic.
func (b *GoBuild) ChangeDir(newDir PathString) *GoBuild {
	if b.err != nil {
		return b
	}
	b.cmd.WorkDir(newDir)
	return b
}

// OutputFilename specifies the name of the built artifact.
func (b *GoBuild) OutputFilename(filename PathString) *GoBuild {
	if b.err != nil {
		return b
	}
	b.output = filename
	return b
}

// ForceRebuild will force all source to be recompiled, rather than relying on build cache.
func (b *GoBuild) ForceRebuild() *GoBuild {
	if b.err != nil {
		return b
	}
	b.forceRebuild = true
	return b
}

// DryRun will print the commands, but not run them.
func (b *GoBuild) DryRun() *GoBuild {
	if b.err != nil {
		return b
	}
	b.dryRun = true
	return b
}

// MemorySanitizer will enable interoperation with memory sanitizer.
// Not all build targets are supported.
func (b *GoBuild) MemorySanitizer() *GoBuild {
	if b.err != nil {
		return b
	}
	b.memorySan = true
	return b
}

// AddressSanitizer will enable interoperation with address sanitizer.
// Not all build targets are supported.
func (b *GoBuild) AddressSanitizer() *GoBuild {
	if b.err != nil {
		return b
	}
	b.addrSan = true
	return b
}

// RaceDetector will enable race detection.
func (b *GoBuild) RaceDetector() *GoBuild {
	if b.err != nil {
		return b
	}
	b.detectRace = true
	return b
}

// PrintPackages will print the names of built packages.
func (b *GoBuild) PrintPackages() *GoBuild {
	if b.err != nil {
		return b
	}
	b.printPackages = true
	return b
}

// PrintCommands will print commands as they are executed.
func (b *GoBuild) PrintCommands() *GoBuild {
	if b.err != nil {
		return b
	}
	b.printCommands = true
	return b
}

// Verbose will print both commands and built packages.
func (b *GoBuild) Verbose() *GoBuild {
	if b.err != nil {
		return b
	}
	return b.PrintPackages().PrintCommands()
}

// BuildArchive builds listed non-main packages into .a files.
// Packages named main are ignored.
func (b *GoBuild) BuildArchive() *GoBuild {
	if b.err != nil {
		return b
	}
	b.buildMode = "archive"
	return b
}

// BuildCArchive builds the listed main package, plus all packages it imports, into a C archive file.
// The only callable symbols will be those functions exported using a cgo //export comment.
// Requires exactly one main package to be listed.
func (b *GoBuild) BuildCArchive() *GoBuild {
	if b.err != nil {
		return b
	}
	b.buildMode = "c-archive"
	return b
}

// BuildCShared builds the listed main package, plus all packages it imports, into a C shared library.
// The only callable symbols will be those functions exported using a cgo //import comment.
// Requires exactly one main package to be listed.
func (b *GoBuild) BuildCShared() *GoBuild {
	if b.err != nil {
		return b
	}
	b.buildMode = "c-shared"
	return b
}

// BuildShared will combine all the listed non-main packages into a single shared library that will be used when building with the -linkshared option.
// Packages named main are ignored.
func (b *GoBuild) BuildShared() *GoBuild {
	if b.err != nil {
		return b
	}
	b.buildMode = "shared"
	return b
}

// BuildExe will build the listed main packages and everything they import into executables.
// Packages not named main are ignored.
func (b *GoBuild) BuildExe() *GoBuild {
	if b.err != nil {
		return b
	}
	b.buildMode = "exe"
	return b
}

// BuildPie will build the listed main packages and everything they import into position independent executables (PIE).
// Packages not named main are ignored.
func (b *GoBuild) BuildPie() *GoBuild {
	if b.err != nil {
		return b
	}
	b.buildMode = "pie"
	return b
}

// BuildPlugin builds the listed main packages, plus all packages that they import, into a Go plugin.
// Packages not named main are ignored.
// Note that this is not supported on all build targets, and as far as I know, there are still issues today with plugins.
// Use at your own risk.
func (b *GoBuild) BuildPlugin() *GoBuild {
	if b.err != nil {
		return b
	}
	b.buildMode = "plugin"
	return b
}

// GoCompileFlags sets arguments to pass to each go tool compile invocation.
func (b *GoBuild) GoCompileFlags(flags ...string) *GoBuild {
	if b.err != nil {
		return b
	}
	for _, flag := range flags {
		flag = strings.TrimSpace(flag)
		if len(flag) > 0 {
			b.gcFlags[flag] = true
		}
	}
	return b
}

// StripDebugSymbols will remove debugging information from the built artifact, reducing file size.
// Assumes TrimPath as well.
func (b *GoBuild) StripDebugSymbols() *GoBuild {
	if b.err != nil {
		return b
	}
	b.stripDebug = true
	b.trimPath = true
	return b
}

// TrimPath will remove build host filesystem path prefix information from the binary.
func (b *GoBuild) TrimPath() *GoBuild {
	if b.err != nil {
		return b
	}
	b.trimPath = true
	return b
}

// SetVariable sets an ldflag to set a variable at build time.
// The pkg parameter should be main, or the fully-qualified package name.
// The variable referenced doesn't have to be exposed (starting with a capital letter).
//
// # Examples
//
// It's a little counter-intuitive how this works, but it's based on how the go tools themselves work.
//
// # Main package
//
// Given a module named 'example.com/me/myproject', and a package directory named 'build' with go files in package 'main', the pkg parameter should just be 'main'.
//
// # Non-main package
//
// Given a module named 'example.com/me/myproject', and a package directory named 'build' with go files in package 'other', the pkg parameter should be 'example.com/me/myproject/build'.
//
// [GoTools.ToModulePackage] is provided as a convenience to make it easier to create these reference strings.
//
// See [this article] for more examples.
//
// [this article]: https://programmingpercy.tech/blog/modify-variables-during-build/
func (b *GoBuild) SetVariable(pkg, varName, value string) *GoBuild {
	if b.err != nil {
		return b
	}
	return b.LinkerFlags(fmt.Sprintf("-X '%s.%s=%s'", pkg, varName, value))
}

// LinkerFlags sets linker flags (ldflags) values for this build.
func (b *GoBuild) LinkerFlags(flags ...string) *GoBuild {
	if b.err != nil {
		return b
	}
	for _, flag := range flags {
		flag = strings.TrimSpace(flag)
		if len(flag) > 0 {
			b.ldFlags[flag] = true
		}
	}
	return b
}

// Tags sets build tags to be activated.
func (b *GoBuild) Tags(tags ...string) *GoBuild {
	if b.err != nil {
		return b
	}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if len(tag) > 0 {
			b.tags[tag] = true
		}
	}
	return b
}

// OS sets the target OS for the go build command using the GOOS environment variable.
func (b *GoBuild) OS(os string) *GoBuild {
	if b.err != nil {
		return b
	}
	b.cmd.Env("GOOS", os)
	return b
}

// Arch will set the CPU architecture for the go build command using the GOARCH environment variable.
func (b *GoBuild) Arch(arch string) *GoBuild {
	if b.err != nil {
		return b
	}
	b.cmd.Env("GOARCH", arch)
	return b
}

// Private specifies private hosts that should not go through proxy.golang.org for resolution.
func (b *GoBuild) Private(privateHosts ...string) *GoBuild {
	if b.err != nil {
		return b
	}
	for _, host := range privateHosts {
		host = strings.TrimSpace(host)
		if len(host) == 0 {
			continue
		}
		b.private = append(b.private, host)
	}
	return b
}

// Adapting *Command stuff to *GoBuild
// Specifically leaving out Arg, since I should be already enumerating the most relevant/common args for build.
// If more control is needed, then Go().Command would be more useful.

func (b *GoBuild) Env(key, value string) *GoBuild {
	if b.err != nil {
		return b
	}
	b.cmd.Env(key, value)
	return b
}

func (b *GoBuild) CgoEnabled(enabled bool) *GoBuild {
	if b.err != nil {
		return b
	}
	val := "0"
	if enabled {
		val = "1"
	}
	b.cmd.Env("CGO_ENABLED", val)
	return b
}

func (b *GoBuild) Workdir(workdir PathString) *GoBuild {
	if b.err != nil {
		return b
	}
	b.cmd.WorkDir(workdir)
	return b
}

func (b *GoBuild) Silent() *GoBuild {
	if b.err != nil {
		return b
	}
	b.cmd.Silent()
	return b
}

func (b *GoBuild) Run(ctx context.Context) error {
	ctx, log := WithGroup(ctx, "go build")
	if b.err != nil {
		return log.WrapErr(b.err)
	}

	b.cmd.OptArg(len(b.output) > 0, "-o", b.output.String())
	b.cmd.OptArg(b.forceRebuild, "-a")
	b.cmd.OptArg(b.dryRun, "-n")
	b.cmd.OptArg(b.detectRace, "-race")
	b.cmd.OptArg(b.memorySan, "-msan")
	b.cmd.OptArg(b.addrSan, "-asan")
	b.cmd.OptArg(b.printPackages, "-v")
	b.cmd.OptArg(b.printCommands, "-x")
	if len(b.buildMode) > 0 {
		b.cmd.Arg("-buildmode=" + b.buildMode)
	}
	if len(b.gcFlags) > 0 {
		b.cmd.Arg(fmt.Sprintf("-gcflags=%s", strings.Join(keySlice(b.gcFlags), " ")))
	}
	b.cmd.OptArg(b.trimPath, "-trimpath")
	if b.stripDebug {
		b.LinkerFlags("-s", "-w")
	}
	if len(b.ldFlags) > 0 {
		b.cmd.Arg(fmt.Sprintf("-ldflags=%s", strings.Join(keySlice(b.ldFlags), " ")))
	}
	if len(b.tags) > 0 {
		b.cmd.Arg("-tags", strings.Join(keySlice(b.tags), ","))
	}
	if len(b.private) > 0 {
		b.cmd.Env("GOPRIVATE", strings.Join(b.private, ","))
	}

	b.cmd.Arg(keySlice(b.targets)...)
	return b.cmd.Run(ctx)
}

func (b *GoBuild) Task() Task {
	return b.Run
}

func (g *GoTools) Run(target string, args ...string) *Command {
	return g.Command("run", target).Arg(args...).CaptureStdin()
}

func keySlice[T any](set map[string]T) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	return keys
}

var (
	modNamePattern = regexp.MustCompile(`^\s*module\s+(\S+)$`)
)

func locateModRoot(base ...PathString) (PathString, error) {
	var (
		dir PathString
	)
	if len(base) > 0 {
		dir = base[0]
	} else {
		_dir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("unable to locate working directory: %v", err)
		}
		dir = Path(_dir)
	}
	modPath, found := scanGoMod(dir)
	if !found {
		return "", fmt.Errorf("unable to locate go.mod in '%s' or any parent directory", dir)
	}
	return modPath, nil
}

func scanGoMod(root PathString) (PathString, bool) {
	if root.IsFile() {
		root = root.Dir()
	}
	for root != root.Dir() {
		found := root.Join("go.mod")
		if found.IsFile() {
			return found, true
		}
		root = root.Dir()
	}

	// Need to check root too in case it's a relative path.
	found := root.Join("go.mod")
	if found.IsFile() {
		return found, true
	}
	return "", false
}

func parseModuleName(goModPath PathString) (string, error) {
	f, err := goModPath.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open '%s': %w", goModPath, err)
	}
	defer func() {
		_ = f.Close()
	}()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if modNamePattern.MatchString(line) {
			groups := modNamePattern.FindStringSubmatch(line)
			return groups[1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("not a valid go.mod file")
}
