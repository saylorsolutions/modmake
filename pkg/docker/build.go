package docker

import (
	"context"
	"fmt"
	"github.com/saylorsolutions/modmake"
)

func (d *Inst) Build(dockerfilePath modmake.PathString) *Build {
	if len(dockerfilePath) == 0 {
		dockerfilePath = modmake.Path(".", "Dockerfile")
	}
	return &Build{
		inst:           d,
		dockerfilePath: dockerfilePath,
		contextDir:     modmake.Path("."),
		buildArgs:      map[string]any{},
		buildSecrets:   map[string]string{},
	}
}

type Build struct {
	inst           *Inst
	enableDebug    bool
	noCache        bool
	pullRefs       bool
	dockerfilePath modmake.PathString
	contextDir     modmake.PathString
	buildArgs      map[string]any
	writeImageID   modmake.PathString
	labels         []string
	buildSecrets   map[string]string
	imageTag       string
}

func (b *Build) ContextDir(dir modmake.PathString) *Build {
	b.contextDir = dir
	return b
}

func (b *Build) BuildArg(key, val string) *Build {
	b.buildArgs[key] = val
	return b
}

func (b *Build) EnableDebugLogging() *Build {
	b.enableDebug = true
	return b
}

func (b *Build) WriteImageIDTo(idFile modmake.PathString) *Build {
	b.writeImageID = idFile
	return b
}

func (b *Build) Label(imageLabel string) *Build {
	b.labels = append(b.labels, imageLabel)
	return b
}

func (b *Build) NoCache() *Build {
	b.noCache = true
	return b
}

func (b *Build) Pull() *Build {
	b.pullRefs = true
	return b
}

func (b *Build) Secret(id, value string) *Build {
	b.buildSecrets[id] = value
	return b
}

func (b *Build) Tag(imageTag string) *Build {
	b.imageTag = imageTag
	return b
}

func (b *Build) Tagf(tagFormat string, args ...any) *Build {
	b.imageTag = fmt.Sprintf(tagFormat, args...)
	return b
}

func (b *Build) Command() *modmake.Command {
	cmd := modmake.Exec(b.inst.dockerPath.String(), "build").
		Arg("-f", b.dockerfilePath.String()).
		TrailingArg(b.contextDir.String())
	cmd.CaptureStdin()
	for key, val := range b.buildArgs {
		cmd.Arg("--build-arg", fmt.Sprintf("%s=%s", key, val))
	}
	if b.enableDebug {
		cmd.Arg("--debug")
	}
	if len(b.writeImageID) > 0 {
		cmd.Arg("--iidfile", b.writeImageID.String())
	}
	for _, label := range b.labels {
		cmd.Arg("--label", label)
	}
	if b.noCache {
		cmd.Arg("--no-cache")
	}
	if b.pullRefs {
		cmd.Arg("--pull")
	}
	for id, val := range b.buildSecrets {
		cmd.Arg("--secret", fmt.Sprintf("%s=%s", id, val))
	}
	if len(b.imageTag) > 0 {
		cmd.Arg("-t", b.imageTag)
	}
	return cmd
}

func (b *Build) Task() modmake.Task {
	return b.Run
}

func (b *Build) Run(ctx context.Context) error {
	return b.Command().Run(ctx)
}

func (b *Build) String() string {
	return b.Command().String()
}
