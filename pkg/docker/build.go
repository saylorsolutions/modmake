package docker

import (
	"context"
	"fmt"

	"github.com/saylorsolutions/modmake"
)

func Build(dockerfilePath modmake.PathString) *Builder {
	if len(dockerfilePath) == 0 {
		dockerfilePath = modmake.Path(".", "Dockerfile")
	}
	return &Builder{
		dockerfilePath: dockerfilePath,
		contextDir:     modmake.Path("."),
		buildArgs:      map[string]any{},
		buildSecrets:   map[string]string{},
	}
}

type Builder struct {
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

func (b *Builder) ContextDir(dir modmake.PathString) *Builder {
	b.contextDir = dir
	return b
}

func (b *Builder) BuildArg(key, val string) *Builder {
	b.buildArgs[key] = val
	return b
}

func (b *Builder) EnableDebugLogging() *Builder {
	b.enableDebug = true
	return b
}

func (b *Builder) WriteImageIDTo(idFile modmake.PathString) *Builder {
	b.writeImageID = idFile
	return b
}

func (b *Builder) Label(imageLabel string) *Builder {
	b.labels = append(b.labels, imageLabel)
	return b
}

func (b *Builder) NoCache() *Builder {
	b.noCache = true
	return b
}

func (b *Builder) Pull() *Builder {
	b.pullRefs = true
	return b
}

func (b *Builder) Secret(id, value string) *Builder {
	b.buildSecrets[id] = value
	return b
}

func (b *Builder) Tag(imageTag string) *Builder {
	b.imageTag = imageTag
	return b
}

func (b *Builder) Tagf(tagFormat string, args ...any) *Builder {
	b.imageTag = fmt.Sprintf(tagFormat, args...)
	return b
}

// Command resolves the docker CLI, builds the build Command, and returns it for further customization.
func (b *Builder) Command() *modmake.Command {
	cmd := modmake.Exec(resolveDockerPath().String(), "build").
		Arg("-f", b.dockerfilePath.String()).
		TrailingArg(b.contextDir.String()).
		LogGroup("docker-build")
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

func (b *Builder) Task() modmake.Task {
	return b.Run
}

func (b *Builder) Run(ctx context.Context) error {
	return b.Command().Run(ctx)
}

func (b *Builder) String() string {
	return b.Command().String()
}
