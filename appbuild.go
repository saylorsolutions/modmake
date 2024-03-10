package modmake

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/saylorsolutions/modmake/assert"
)

// AppBuildFunc is a function used to customize an AppBuild or AppVariant's build step.
type AppBuildFunc func(gb *GoBuild)

func (bf AppBuildFunc) Then(other AppBuildFunc) AppBuildFunc {
	return func(gb *GoBuild) {
		if bf != nil {
			bf(gb)
		}
		if other != nil {
			other(gb)
		}
	}
}

// AppPackageFunc is a function used to customize an AppVariant's package step.
type AppPackageFunc func(binaryPath, destDir PathString, app, variant, version string) Task

func (pf AppPackageFunc) Then(other AppPackageFunc) AppPackageFunc {
	return func(binaryPath, destDir PathString, app, variant, version string) Task {
		var t Task
		if pf != nil {
			t.Then(pf(binaryPath, destDir, app, variant, version))
		}
		if other != nil {
			t.Then(other(binaryPath, destDir, app, variant, version))
		}
		return t
	}
}

// PackageTar will package the binary into a tar.gz.
// This is the default for non-windows builds.
func PackageTar() AppPackageFunc {
	return func(binaryPath, destDir PathString, app, variant, version string) Task {
		return Tar(destDir.Join(fmt.Sprintf("%s_%s_%s.tar.gz", app, variant, version))).
			AddFileWithPath(binaryPath, binaryPath.Base()).
			Create()
	}
}

// PackageZip will package the binary into a zip.
// This is the default for windows builds.
func PackageZip() AppPackageFunc {
	return func(binaryPath, destDir PathString, app, variant, version string) Task {
		return Zip(destDir.Join(fmt.Sprintf("%s_%s_%s.zip", app, variant, version))).
			AddFileWithPath(binaryPath, binaryPath.Base()).
			Create()
	}
}

// PackageGoInstall will copy the binary to GOPATH/bin.
// This is the default packaging for the AppBuild generated install step.
func PackageGoInstall() AppPackageFunc {
	return func(binaryPath, _ PathString, app, _, version string) Task {
		gopathBin := Path(Go().GetEnv("GOPATH"), "bin")
		return Task(func(ctx context.Context) error {
			return binaryPath.CopyTo(gopathBin.JoinPath(binaryPath.Base()))
		}).Then(Print(warnColor("Ensure that " + gopathBin.String() + " is on your PATH to easily access " + app)))
	}
}

// AppBuild is a somewhat opinionated abstraction over the common pattern of building a static executable, including packaging.
// The build step may be customized as needed, and different OS/Arch variants may be created as needed.
// Each built executable will be output to ${MODROOT}/build/${APP}_${VARIANT_NAME}/${APP}
// Default packaging will write a zip or tar.gz to ${MODROOT}/dist/${APP}/${APP}_${VARIANT_NAME}_${VERSION}.(zip|tar.gz)
// Each variant may override or remove its packaging step.
type AppBuild struct {
	mainPath, version  string
	buildFunc          AppBuildFunc
	variants           []*AppVariant
	appName            string
	installPackageFunc AppPackageFunc
}

// NewAppBuild creates a new AppBuild with the given details.
// Empty values are not allowed and will result in a panic.
// If mainPath is not prefixed with the module name, then it will be added.
func NewAppBuild(appName, mainPath, version string) *AppBuild {
	assert.NotEmpty(&appName)
	assert.NotEmpty(&mainPath)
	assert.SemverVersion(&version)
	if !strings.HasPrefix(mainPath, Go().ModuleName()) {
		mainPath = Go().ToModulePath(mainPath)
	}
	return &AppBuild{
		appName:            appName,
		mainPath:           mainPath,
		version:            version,
		installPackageFunc: PackageGoInstall(),
	}
}

func (a *AppBuild) hasVariant(variant string) bool {
	for _, v := range a.variants {
		if v.variant == variant {
			return true
		}
	}
	return false
}

// Build allows customizing the GoBuild used to build all variants, which may be further customized for a specific AppVariant.
func (a *AppBuild) Build(bf AppBuildFunc) *AppBuild {
	a.buildFunc = bf
	return a
}

// Install allows specifying the AppPackageFunc used for installation.
// This defaults to PackageGoInstall.
func (a *AppBuild) Install(pf AppPackageFunc) *AppBuild {
	if pf == nil {
		return a
	}
	a.installPackageFunc = pf
	return a
}

func (a *AppBuild) buildName(v *AppVariant) string {
	return "build-" + a.appName + "_" + v.variant
}

func (a *AppBuild) packageName(v *AppVariant) string {
	return "package-" + a.appName + "_" + v.variant
}

func (a *AppBuild) goBuild(v *AppVariant) *GoBuild {
	gb := Go().Build(a.mainPath)
	if v.buildFunc != nil {
		v.buildFunc(gb)
	}
	gb.
		OutputFilename(v.buildOutput).
		OS(v.os).
		Arch(v.arch)
	if a.buildFunc != nil {
		a.buildFunc(gb)
	}
	return gb
}

func (a *AppBuild) pkgTask(v *AppVariant) Task {
	if v.packageFunc != nil {
		return v.packageFunc(v.buildOutput, Path("dist", a.appName), a.appName, v.variant, a.version)
	}
	return nil
}

func (a *AppBuild) generateBuild() *Build {
	if len(a.variants) == 0 {
		a.HostVariant()
	}
	b := NewBuild()
	b.Package().DependsOnRunner("create-"+a.appName+"-dist", "Ensures the application dist directory exists",
		RemoveDir(Path("dist", a.appName)),
	)
	for _, v := range a.variants {
		buildStep := NewStep(a.buildName(v), fmt.Sprintf("Builds %s for %s/%s", a.appName, v.os, v.arch))
		buildStep.Does(a.goBuild(v))
		b.AddStep(buildStep)
		b.Build().DependsOnRunner("clean-"+a.buildName(v), "Removes previous build output",
			RemoveDir(v.buildOutput.Dir()).Then(MkdirAll(v.buildOutput.Dir(), 0755)),
		)
		b.Build().DependsOn(buildStep)

		pkgTask := a.pkgTask(v)
		if pkgTask != nil {
			pkgStep := NewStep(a.packageName(v), fmt.Sprintf("Packages %s for %s/%s", a.appName, v.os, v.arch))
			pkgStep.Does(a.pkgTask(v))
			pkgStep.DependsOnRunner("ensure-dist-"+a.packageName(v), "Ensures the application dist directory exists",
				MkdirAll(Path("dist", a.appName), 0755),
			)
			b.AddStep(pkgStep)
			b.Package().DependsOn(pkgStep)
		}
	}
	installVariant := a.NamedVariant("install", runtime.GOOS, runtime.GOARCH).Package(a.installPackageFunc)
	installStep := NewStep("install", "Installs "+a.appName).Does(a.pkgTask(installVariant))
	installStep.BeforeRun(a.goBuild(installVariant))
	b.AddStep(installStep)
	return b
}

// ImportApp imports an AppBuild as a new build, attaching its build and package steps as dependencies of the parent build.
func (b *Build) ImportApp(a *AppBuild) {
	other := a.generateBuild()
	b.Import(a.appName, other)
	b.Build().DependsOn(b.Step(a.appName + ":build"))
	b.Package().DependsOn(b.Step(a.appName + ":package"))
}

// AppVariant is a variant of an AppBuild with an OS/Arch specified.
type AppVariant struct {
	variant, os, arch    string
	buildOutput, distDir PathString
	buildFunc            AppBuildFunc
	packageFunc          AppPackageFunc
}

// HostVariant creates an AppVariant with the current host's GOOS and GOARCH settings.
// Packaging will be disabled by default for this variant.
func (a *AppBuild) HostVariant() *AppVariant {
	return a.NamedVariant("local", runtime.GOOS, runtime.GOARCH).NoPackage()
}

// Variant creates an AppVariant with the given OS and architecture settings.
func (a *AppBuild) Variant(os, arch string) *AppVariant {
	return a.NamedVariant(os+"_"+arch, os, arch)
}

// NamedVariant creates an AppVariant much like [AppBuild.Variant], but with a custom variant name.
func (a *AppBuild) NamedVariant(variant, os, arch string) *AppVariant {
	assert.NotEmpty(&variant)
	assert.NotEmpty(&os)
	assert.NotEmpty(&arch)
	if a.hasVariant(variant) {
		panic("variant " + variant + " already exists")
	}
	exeName := a.appName
	pkgFunc := PackageTar()
	if os == "windows" {
		exeName += ".exe"
		pkgFunc = PackageZip()
	}
	v := &AppVariant{
		variant:     variant,
		os:          os,
		arch:        arch,
		buildOutput: Path("build", fmt.Sprintf("%s_%s", a.appName, variant), exeName),
		distDir:     Path("dist", a.appName),
		packageFunc: pkgFunc,
	}
	a.variants = append(a.variants, v)
	return v
}

// Build sets the AppBuildFunc specific to this variant.
// [AppBuildFunc.Then] may be used to combine multiple build customizations.
func (v *AppVariant) Build(bf AppBuildFunc) *AppVariant {
	v.buildFunc = bf
	return v
}

// Package sets the AppPackageFunc specific to this variant.
// [AppPackageFunc.Then] may be used to combine multiple package steps.
func (v *AppVariant) Package(pf AppPackageFunc) *AppVariant {
	v.packageFunc = pf
	return v
}

// NoPackage will mark this variant as one that doesn't include a packaging step.
func (v *AppVariant) NoPackage() *AppVariant {
	v.packageFunc = nil
	return v
}
