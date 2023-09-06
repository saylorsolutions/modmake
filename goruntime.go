package modmake

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type GoTools struct {
	rootDir string
}

var instance *GoTools

func Go() *GoTools {
	if instance == nil {
		rootDir, ok := os.LookupEnv("GOROOT")
		if !ok {
			panic("Unable to resolve environment variable GOROOT. Is Go installed correctly?")
		}
		instance = &GoTools{
			rootDir: rootDir,
		}
	}
	return instance
}

func (g *GoTools) goTool() string {
	return filepath.Join(g.rootDir, "bin", "go")
}

func (g *GoTools) Command(command string, args ...string) *Command {
	return Exec(g.goTool(), command).Arg(args...)
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
		Command: Exec(g.goTool(), "clean"),
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
	return c.Command.Run(ctx)
}

func (g *GoTools) Install(pkg string) *Command {
	return Exec(g.goTool(), "install", pkg)
}

func (g *GoTools) Get(pkg string) *Command {
	return Exec(g.goTool(), "get").Arg(pkg)
}

func (g *GoTools) GetUpdated(pkg string) *Command {
	return Exec(g.goTool(), "get", "-u").Arg(pkg)
}

func (g *GoTools) ModTidy() *Command {
	return Exec(g.goTool(), "mod", "tidy")
}

func (g *GoTools) Test(pattern string) *Command {
	return Exec(g.goTool(), "test", "-bench=^$", "-run="+pattern, "-v")
}

func (g *GoTools) TestAll() *Command {
	return g.Test(".")
}

func (g *GoTools) Generate(patterns ...string) *Command {
	return Exec(g.goTool(), "generate").Arg(patterns...)
}

func (g *GoTools) GenerateAll() *Command {
	return g.Generate("./...")
}

func (g *GoTools) Benchmark(pattern string) *Command {
	return Exec(g.goTool(), "test", "-bench="+pattern, "-run=^$", "-v")
}

func (g *GoTools) BenchmarkAll() *Command {
	return g.Benchmark(".")
}

type GoBuild struct {
	err           error
	cmd           *Command
	output        string
	forceRebuild  bool
	dryRun        bool
	detectRace    bool
	memorySan     bool
	addrSan       bool
	printPackages bool
	printCommands bool
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
		cmd:     Exec(g.goTool(), "build"),
		targets: _targets,
		gcFlags: map[string]bool{},
		ldFlags: map[string]bool{},
		tags:    map[string]bool{},
	}
}

// ChangeDir will change the working directory from the default to a new location.
// If an absolute path cannot be derived from newDir, then this function will panic.
func (b *GoBuild) ChangeDir(newDir string) *GoBuild {
	if b.err != nil {
		return b
	}
	b.cmd.WorkDir(newDir)
	return b
}

// OutputFilename specifies the name of the built artifact.
func (b *GoBuild) OutputFilename(filename string) *GoBuild {
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

// StripDebugSymbols will remove debugging information from the built artifact, improving file size.
func (b *GoBuild) StripDebugSymbols() *GoBuild {
	if b.err != nil {
		return b
	}
	return b.LinkerFlags("-s", "-w")
}

// SetVariable sets an ldflag to set a variable at build time.
func (b *GoBuild) SetVariable(pkg, varName, value string) *GoBuild {
	if b.err != nil {
		return b
	}
	return b.LinkerFlags(fmt.Sprintf("-X %s.%s=%s", pkg, varName, value))
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

func (b *GoBuild) Workdir(workdir string) *GoBuild {
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
	if b.err != nil {
		return b.err
	}

	b.cmd.OptArg(len(b.output) > 0, "-o", b.output)
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

func (g *GoTools) Run(target string, args ...string) *Command {
	return Exec(g.goTool(), "run").Arg(target).Arg(args...).CaptureStdin()
}

func keySlice[T any](set map[string]T) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	return keys
}
