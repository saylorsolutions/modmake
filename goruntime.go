package modmake

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
			panic("Unable to resolve GOROOT. Is Go installed?")
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

func (g *GoTools) Test(pattern string) Runner {
	return RunnerFunc(func(ctx context.Context) error {
		val, err := getWorkdir(ctx)
		if err != nil {
			return err
		}
		cmd := exec.Command(g.goTool(), "test", "-v", filepath.Join(val, pattern))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	})
}

func (g *GoTools) TestAll() Runner {
	return g.Test("...")
}

func (g *GoTools) Generate(pattern string) Runner {
	return RunnerFunc(func(ctx context.Context) error {
		workdir, err := getWorkdir(ctx)
		if err != nil {
			return err
		}
		cmd := exec.Command(g.goTool(), "generate", filepath.Join(workdir, pattern))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	})
}

func (g *GoTools) GenerateAll() Runner {
	return g.Generate("...")
}

func (g *GoTools) Benchmark(pattern string) Runner {
	return RunnerFunc(func(ctx context.Context) error {
		val, err := getWorkdir(ctx)
		if err != nil {
			return err
		}
		cmd := exec.Command(g.goTool(), "bench", "-v", filepath.Join(val, pattern))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	})
}

func (g *GoTools) BenchmarkAll() Runner {
	return g.Benchmark("...")
}

type GoBuild struct {
	goTool        string
	goos          string
	goarch        string
	changeDir     string
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
}

// NewBuild creates a GoBuild to hold parameters for a new go build run.
func (g *GoTools) NewBuild() *GoBuild {
	return &GoBuild{
		goTool: g.goTool(),
	}
}

// ChangeDir will change the working directory from the default to a new location.
// If an absolute path cannot be derived from newDir, then this function will panic.
func (b *GoBuild) ChangeDir(newDir string) *GoBuild {
	var err error
	newDir, err = filepath.Abs(newDir)
	if err != nil {
		panic("unable to derive absolute path from newDir: " + newDir)
	}
	b.changeDir = newDir
	return b
}

// OutputFilename specifies the name of the built artifact.
func (b *GoBuild) OutputFilename(filename string) *GoBuild {
	b.output = filename
	return b
}

// ForceRebuild will force all source to be recompiled, rather than relying on build cache.
func (b *GoBuild) ForceRebuild() *GoBuild {
	b.forceRebuild = true
	return b
}

// DryRun will print the commands, but not run them.
func (b *GoBuild) DryRun() *GoBuild {
	b.dryRun = true
	return b
}

// MemorySanitizer will enable interoperation with memory sanitizer.
// Not all build targets are supported.
func (b *GoBuild) MemorySanitizer() *GoBuild {
	b.memorySan = true
	return b
}

// AddressSanitizer will enable interoperation with address sanitizer.
// Not all build targets are supported.
func (b *GoBuild) AddressSanitizer() *GoBuild {
	b.addrSan = true
	return b
}

// RaceDetector will enable race detection.
func (b *GoBuild) RaceDetector() *GoBuild {
	b.detectRace = true
	return b
}

// PrintPackages will print the names of built packages.
func (b *GoBuild) PrintPackages() *GoBuild {
	b.printPackages = true
	return b
}

// PrintCommands will print commands as they are executed.
func (b *GoBuild) PrintCommands() *GoBuild {
	b.printCommands = true
	return b
}

// Verbose will print both commands and built packages.
func (b *GoBuild) Verbose() *GoBuild {
	return b.PrintPackages().PrintCommands()
}

// BuildArchive builds listed non-main packages into .a files.
// Packages named main are ignored.
func (b *GoBuild) BuildArchive() *GoBuild {
	b.buildMode = "archive"
	return b
}

// BuildCArchive builds the listed main package, plus all packages it imports, into a C archive file.
// The only callable symbols will be those functions exported using a cgo //export comment.
// Requires exactly one main package to be listed.
func (b *GoBuild) BuildCArchive() *GoBuild {
	b.buildMode = "c-archive"
	return b
}

// BuildCShared builds the listed main package, plus all packages it imports, into a C shared library.
// The only callable symbols will be those functions exported using a cgo //import comment.
// Requires exactly one main package to be listed.
func (b *GoBuild) BuildCShared() *GoBuild {
	b.buildMode = "c-shared"
	return b
}

// BuildShared will combine all the listed non-main packages into a single shared library that will be used when building with the -linkshared option.
// Packages named main are ignored.
func (b *GoBuild) BuildShared() *GoBuild {
	b.buildMode = "shared"
	return b
}

// BuildExe will build the listed main packages and everything they import into executables.
// Packages not named main are ignored.
func (b *GoBuild) BuildExe() *GoBuild {
	b.buildMode = "exe"
	return b
}

// BuildPie will build the listed main packages and everything they import into position independent executables (PIE).
// Packages not named main are ignored.
func (b *GoBuild) BuildPie() *GoBuild {
	b.buildMode = "pie"
	return b
}

// BuildPlugin builds the listed main packages, plus all packages that they import, into a Go plugin.
// Packages not named main are ignored.
// Note that this is not supported on all build targets, and as far as I know, there are still issues today with plugins.
// Use at your own risk.
func (b *GoBuild) BuildPlugin() *GoBuild {
	b.buildMode = "plugin"
	return b
}

// GoCompileFlags sets arguments to pass to each go tool compile invocation.
func (b *GoBuild) GoCompileFlags(flags ...string) *GoBuild {
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
	return b.LinkerFlags("-s", "-w")
}

func (b *GoBuild) SetVariable(pkg, varName, value string) *GoBuild {
	return b.LinkerFlags(fmt.Sprintf("-X %s.%s=%s", pkg, varName, value))
}

// LinkerFlags sets linker flags (ldflags) values for this build.
func (b *GoBuild) LinkerFlags(flags ...string) *GoBuild {
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
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if len(tag) > 0 {
			b.tags[tag] = true
		}
	}
	return b
}

// BuildOS sets the target OS for the go build command using the GOOS environment variable.
func (b *GoBuild) BuildOS(os string) *GoBuild {
	b.goos = os
	return b
}

// BuildCpuArch will set the CPU architecture for the go build command using the GOARCH environment variable.
func (b *GoBuild) BuildCpuArch(arch string) *GoBuild {
	b.goarch = arch
	return b
}

// Targets sets the build targets for this build execution.
func (b *GoBuild) Targets(targets ...string) *GoBuild {
	for _, target := range targets {
		target = strings.TrimSpace(target)
		if len(target) > 0 {
			b.targets[target] = true
		}
	}
	return b
}

func (b *GoBuild) Run(ctx context.Context) error {
	var args []string

	if len(b.changeDir) > 0 {
		args = append(args, "-C", b.changeDir)
	} else {
		val, err := getWorkdir(ctx)
		if err != nil {
			return err
		}
		args = append(args, "-C", val)
	}
	if len(b.output) > 0 {
		args = append(args, "-o", b.output)
	}
	if b.forceRebuild {
		args = append(args, "-a")
	}
	if b.dryRun {
		args = append(args, "-n")
	}
	if b.detectRace {
		args = append(args, "-race")
	}
	if b.memorySan {
		args = append(args, "-msan")
	}
	if b.addrSan {
		args = append(args, "-asan")
	}
	if b.printPackages {
		args = append(args, "-v")
	}
	if b.printCommands {
		args = append(args, "-x")
	}
	if len(b.buildMode) > 0 {
		args = append(args, "-buildmode="+b.buildMode)
	}
	if len(b.gcFlags) > 0 {
		args = append(args, fmt.Sprintf("-gcflags=%s", strings.Join(keySlice(b.gcFlags), " ")))
	}
	if len(b.ldFlags) > 0 {
		args = append(args, fmt.Sprintf("-ldflags=%s", strings.Join(keySlice(b.ldFlags), " ")))
	}
	if len(b.tags) > 0 {
		args = append(args, "-tags", strings.Join(keySlice(b.tags), ","))
	}
	args = append(args, keySlice(b.targets)...)

	cmd := exec.Command(b.goTool, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if len(b.goos) > 0 {
		cmd.Env = append(cmd.Env, "GOOS="+b.goos)
	}
	if len(b.goarch) > 0 {
		cmd.Env = append(cmd.Env, "GOARCH="+b.goarch)
	}

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func keySlice[T any](set map[string]T) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	return keys
}
