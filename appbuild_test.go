package modmake

import (
	"context"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestNewAppBuild(t *testing.T) {
	a := NewAppBuild("testapp", "cmd/modmake", "1.0.0")
	assert.NotNil(t, a)
	assert.Equal(t, "testapp", a.appName)
	assert.Equal(t, Go().ToModulePath("cmd/modmake"), a.mainPath, "Module path should be prepended to a raw path")
	assert.Equal(t, "1.0.0", a.version)

	v := a.Variant("linux", "amd64").Build(func(gb *GoBuild) {
		gb.OS("whatever").Arch("something")
	})
	assert.NotNil(t, v)
	assert.Equal(t, "linux_amd64", v.variant)
	assert.Equal(t, Path("build/testapp_linux_amd64/testapp"), v.buildOutput)
	gb := a.goBuild(v)
	var (
		foundOS, foundArch bool
	)
	envs := gb.cmd.env
	// Traverse in reverse order, since GOOS and GOARCH are added to the Cmd as environment overrides.
	for i := len(envs) - 1; i >= 0; i-- {
		env := envs[i]
		kv := strings.SplitN(env, "=", 2)
		assert.Len(t, kv, 2)
		k, v := kv[0], kv[1]
		if !foundOS && k == "GOOS" {
			assert.Equal(t, "linux", v)
			// Ignore previously set values.
			foundOS = true
		}
		if !foundArch && k == "GOARCH" {
			assert.Equal(t, "amd64", v)
			// Ignore previously set values.
			foundArch = true
		}
	}
	assert.Equal(t, Path("build/testapp_linux_amd64/testapp"), gb.output)
	assert.Equal(t, Path("dist/testapp"), v.distDir)
}

func TestNewAppBuild_Windows_adds_exe_suffix(t *testing.T) {
	a := NewAppBuild("testapp", "cmd/modmake", "1.0.0")
	v := a.Variant("windows", "amd64")
	assert.Equal(t, Path("build/testapp_windows_amd64/testapp.exe"), v.buildOutput)
}

func TestAppVariant_NoPackage(t *testing.T) {
	a := NewAppBuild("testapp", "cmd/modmake", "1.0.0")
	v := a.Variant("darwin", "arm64").NoPackage()
	task := a.pkgTask(v)
	assert.Nil(t, task, "Calling NoPackage should result in a nil Task")
}

func TestAppVariant_Package(t *testing.T) {
	var packageCalled bool
	a := NewAppBuild("testapp", "cmd/modmake", "1.0.0")
	v := a.Variant("darwin", "arm64").
		NoPackage().
		Package(func(binaryPath, destDir PathString, app, variant, version string) Task {
			return Plain(func() {
				packageCalled = true
			})
		})
	task := a.pkgTask(v)
	assert.NoError(t, task.Run(context.Background()))
	assert.NotNil(t, task, "Calling Package should override the package Task")
	assert.True(t, packageCalled, "The new package task should have been called")
}

func TestAppBuild_NoDupeVariants(t *testing.T) {
	a := NewAppBuild("testapp", "cmd/modmake", "1.0.0")
	assert.NotPanics(t, func() {
		a.NamedVariant("dupe", "darwin", "arm64").NoPackage()
	})
	assert.Panics(t, func() {
		a.NamedVariant("dupe", "linux", "amd64").NoPackage()
	}, "Should panic when a duplicate variant name is added to an AppBuild since it's configuration time")
}

func TestBuild_ImportApp_Default(t *testing.T) {
	b := NewBuild()
	a := NewAppBuild("testapp", "cmd/modmake", "1.0.0")
	b.ImportApp(a)
	_, ok := b.StepOk("testapp:build-testapp_localtest")
	assert.True(t, ok, "A default task should be created when no variants are added")
	assert.Len(t, a.variants, 1)
}

func TestBuild_ImportApp(t *testing.T) {
	b := NewBuild()
	a := NewAppBuild("testapp", "cmd/modmake", "1.0.0")
	win := a.Variant("windows", "amd64")
	mac := a.Variant("darwin", "amd64")
	lin := a.Variant("linux", "arm64")
	b.ImportApp(a)
	_, ok := b.StepOk("testapp:localtest")
	assert.False(t, ok, "A default task should NOT be created when variants are added")
	assert.Len(t, a.variants, 3)

	shouldExist := []string{
		"testapp:build-testapp_windows_amd64",
		"testapp:package-testapp_windows_amd64",
		"testapp:build-testapp_darwin_amd64",
		"testapp:package-testapp_darwin_amd64",
		"testapp:build-testapp_linux_arm64",
		"testapp:package-testapp_linux_arm64",

		// These should be equivalent to the list above
		"testapp:" + a.buildName(win),
		"testapp:" + a.packageName(win),
		"testapp:" + a.buildName(mac),
		"testapp:" + a.packageName(mac),
		"testapp:" + a.buildName(lin),
		"testapp:" + a.packageName(lin),
	}
	for i := 6; i < len(shouldExist); i++ {
		assert.Equal(t, shouldExist[i-6], shouldExist[i])
	}
	for _, step := range shouldExist {
		_, ok := b.StepOk(step)
		assert.Truef(t, ok, "Step %s should exist", step)
	}
}

func TestAppBuildFunc_Then(t *testing.T) {
	var calls []string
	a := AppBuildFunc(func(gb *GoBuild) {
		calls = append(calls, "a")
	})
	b := AppBuildFunc(func(gb *GoBuild) {
		calls = append(calls, "b")
	})
	a.Then(b)(Go().Build(""))
	assert.Len(t, calls, 2)
	assert.Equal(t, "a", calls[0])
	assert.Equal(t, "b", calls[1])
}

func TestAppPackageFunc_Then(t *testing.T) {
	var calls []string
	a := AppPackageFunc(func(_, _ PathString, _, _, _ string) Task {
		return Plain(func() {
			calls = append(calls, "a")
		})
	})
	b := AppPackageFunc(func(_, _ PathString, _, _, _ string) Task {
		return Plain(func() {
			calls = append(calls, "b")
		})
	})
	a.Then(b)(Path(""), Path(""), "", "", "")
}
