package modmake

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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

func (g *GoTools) Test(patterns ...string) Runner {
	return RunnerFunc(func(ctx context.Context) error {
		workdir, err := getWorkdir(ctx)
		if err != nil {
			return err
		}
		modPath, err := getModPath(workdir)
		if err != nil {
			return err
		}
		for i := 0; i < len(patterns); i++ {
			patterns[i] = relativeOrModulePath(modPath, workdir, patterns[i])
		}
		args := append([]string{"test", "-v"}, patterns...)
		cmd := exec.Command(g.goTool(), args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	})
}

func (g *GoTools) TestAll() Runner {
	return g.Test("...")
}

func (g *GoTools) Generate(patterns ...string) Runner {
	return RunnerFunc(func(ctx context.Context) error {
		workdir, err := getWorkdir(ctx)
		if err != nil {
			return err
		}
		modPath, err := getModPath(workdir)
		if err != nil {
			return err
		}
		for i := 0; i < len(patterns); i++ {
			patterns[i] = relativeOrModulePath(modPath, workdir, patterns[i])
		}
		args := append([]string{"generate"}, patterns...)
		cmd := exec.Command(g.goTool(), args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	})
}

func (g *GoTools) GenerateAll() Runner {
	return g.Generate("...")
}

func (g *GoTools) Benchmark(patterns ...string) Runner {
	return RunnerFunc(func(ctx context.Context) error {
		workdir, err := getWorkdir(ctx)
		if err != nil {
			return err
		}
		modPath, err := getModPath(workdir)
		if err != nil {
			return err
		}
		for i := 0; i < len(patterns); i++ {
			patterns[i] = relativeOrModulePath(modPath, workdir, patterns[i])
		}
		args := append([]string{"test", "-bench", "-v"}, patterns...)
		cmd := exec.Command(g.goTool(), args...)
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
	err           error
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
func (g *GoTools) NewBuild(targets ...string) *GoBuild {
	if len(targets) == 0 {
		return &GoBuild{
			err: errors.New("no targets defined"),
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
		goTool:  g.goTool(),
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

// BuildOS sets the target OS for the go build command using the GOOS environment variable.
func (b *GoBuild) BuildOS(os string) *GoBuild {
	if b.err != nil {
		return b
	}
	b.goos = os
	return b
}

// BuildCpuArch will set the CPU architecture for the go build command using the GOARCH environment variable.
func (b *GoBuild) BuildCpuArch(arch string) *GoBuild {
	if b.err != nil {
		return b
	}
	b.goarch = arch
	return b
}

func (b *GoBuild) Run(ctx context.Context) error {
	if b.err != nil {
		return b.err
	}
	args := []string{"build"}

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

	workdir, err := getWorkdir(ctx)
	if err != nil {
		return err
	}
	args = append(args, keySlice(b.targets)...)
	cmd := exec.Command(b.goTool, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	cmd.Dir = workdir

	if len(b.changeDir) > 0 {
		cmd.Dir = b.changeDir
	}

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

var modCache map[string]string

func getModPath(workdir string) (string, error) {
	if modCache == nil {
		modCache = map[string]string{}
	} else {
		if mod, ok := modCache[workdir]; ok {
			return mod, nil
		}
	}
	var prev string
	cur, err := filepath.Abs(workdir)
	if err != nil {
		return "", err
	}
	for cur != prev {
		mod, found := walkForModPath(prev, cur)
		if found {
			modCache[workdir] = mod
			return mod, nil
		}
		prev = cur
		cur = filepath.Dir(cur)
	}
	return "", errors.New("unable to locate module")
}

var (
	modPattern = regexp.MustCompile(`^module\s+(\S+)$`)
	foundMod   = errors.New("module found")
)

func walkForModPath(prev, cur string) (string, bool) {
	var module string
	err := filepath.WalkDir(cur, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			if path == prev {
				return fs.SkipDir
			}
			return nil
		}
		if d.Name() == "go.mod" {
			mod, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func() {
				_ = mod.Close()
			}()
			scanner := bufio.NewScanner(mod)
			defer func() {
				scanner = nil
			}()
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if modPattern.MatchString(line) {
					groups := modPattern.FindStringSubmatch(line)
					module = groups[1]
					return foundMod
				}
			}
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, foundMod) {
			return module, true
		}
	}
	return "", false
}

func relativeOrModulePath(modPath, workdir, file string) string {
	if strings.HasPrefix(file, modPath) {
		return file
	}
	return relativeToWorkdir(workdir, file)
}
