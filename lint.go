package modmake

import (
	"context"
	"fmt"
	"regexp"
	"sync"
)

const (
	defaultLintVersion    = "latest"
	linterV2VersionString = "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@%s"
	EnvLinterPath         = "MM_LINTER_PATH"
)

var (
	lintVersionPattern = regexp.MustCompile(`^(latest|v2\.\d+\.\d+)$`)
	linterPath         string
	linterInitOnce     sync.Once
)

func init() {
}

// Linter provides a means to configure the linter in various ways.
// The current version uses golangci-lint, and the [Linter.Arg] method is provided to pass arguments specific to that tool.
type Linter struct {
	targetDirs     []string
	enabledChecks  []string
	disabledChecks []string
	verbose        bool
	otherArgs      []string
}

// Enable marks given check(s) as enabled.
func (lint *Linter) Enable(check string, others ...string) *Linter {
	checks := []string{check}
	if len(others) > 0 {
		checks = append(checks, others...)
	}
	lint.enabledChecks = append(lint.enabledChecks, checks...)
	return lint
}

func (lint *Linter) EnableSecurityScanning() *Linter {
	return lint.Enable("gosec")
}

// Disable marks given check(s) as disabled.
func (lint *Linter) Disable(check string, others ...string) *Linter {
	checks := []string{check}
	if len(others) > 0 {
		checks = append(checks, others...)
	}
	lint.disabledChecks = append(lint.disabledChecks, checks...)
	return lint
}

func (lint *Linter) Verbose() *Linter {
	lint.verbose = true
	return lint
}

// Target will target specific files/directories for linting.
// If no target is specified, then "./..." will be used by default.
func (lint *Linter) Target(target string, otherTargets ...string) *Linter {
	targets := []string{target}
	if len(otherTargets) > 0 {
		targets = append(targets, otherTargets...)
	}
	lint.targetDirs = append(lint.targetDirs, targets...)
	return lint
}

// Arg allows passing unmapped arguments to the golangci-lint invocation.
func (lint *Linter) Arg(args ...string) *Linter {
	lint.otherArgs = append(lint.otherArgs, args...)
	return lint
}

func (lint *Linter) Run(ctx context.Context) error {
	linterInitOnce.Do(func() {
		linterPath = F("${" + EnvLinterPath + "}")
		if len(linterPath) == 0 {
			linterPath = Path(Go().GetEnv("GOBIN"), "golangci-lint").String()
		}
	})

	ctx, log := WithGroup(ctx, "lint")
	var args []string
	for _, enabled := range lint.enabledChecks {
		args = append(args, "-E", enabled)
	}
	for _, disabled := range lint.disabledChecks {
		args = append(args, "-D", disabled)
	}
	if lint.verbose {
		args = append(args, "-v")
	}
	if len(lint.otherArgs) > 0 {
		args = append(args, lint.otherArgs...)
	}
	if len(lint.targetDirs) == 0 {
		lint.targetDirs = []string{"./..."}
	}
	log.Info("Running linter")
	return Exec(linterPath).
		Arg(args...).
		TrailingArg("run").
		TrailingArg(lint.targetDirs...).
		Run(ctx)
}

// LintLatest is the same as [Build.Lint], but will use "latest" as the version.
func (b *Build) LintLatest() *Linter {
	return b.Lint("latest")
}

// Lint will enable code linting support for this module, and returns the Linter for further configuration.
// The version parameter must be either "latest" or a string that can describe a version of a go module.
//
// The default path for invoking the linter is "${GOBIN}/golangci-lint".
// The environment variable MM_LINTER_PATH (see EnvLinterPath) can be used to override the invocation path to the golangci-lint executable, in case GOBIN is undefined or not on the PATH.
func (b *Build) Lint(version string) *Linter {
	lintVersion := defaultLintVersion
	if len(version) > 0 {
		lintVersion = version
	}
	if !lintVersionPattern.MatchString(lintVersion) {
		panic(fmt.Sprintf("invalid linter version %s", lintVersion))
	}
	lintVersion = fmt.Sprintf(linterV2VersionString, lintVersion)
	installLinter := b.AddNewStep("install-linter", "Installs golangci-lint", Go().Install(lintVersion))
	b.Tools().DependsOn(installLinter)
	linter := new(Linter)
	lintStep := b.AddNewStep("lint", "Analyses code for quality issues", linter)
	b.Test().DependsOn(lintStep)
	lintStep.DependsOn(installLinter)
	return linter
}
