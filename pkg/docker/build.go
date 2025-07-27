package docker

import (
	"context"
	"fmt"
	"github.com/saylorsolutions/modmake"
)

func Build(dockerfilePath modmake.PathString) *DockerBuild {
	if len(dockerfilePath) == 0 {
		dockerfilePath = modmake.Path(".", "Dockerfile")
	}
	return &DockerBuild{
		dockerfilePath: dockerfilePath,
		contextDir:     modmake.Path("."),
		buildArgs:      map[string]any{},
		buildSecrets:   map[string]string{},
	}
}

type DockerBuild struct {
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

func (b *DockerBuild) ContextDir(dir modmake.PathString) *DockerBuild {
	b.contextDir = dir
	return b
}

func (b *DockerBuild) BuildArg(key, val string) *DockerBuild {
	b.buildArgs[key] = val
	return b
}

func (b *DockerBuild) EnableDebugLogging() *DockerBuild {
	b.enableDebug = true
	return b
}

func (b *DockerBuild) WriteImageIDTo(idFile modmake.PathString) *DockerBuild {
	b.writeImageID = idFile
	return b
}

func (b *DockerBuild) Label(imageLabel string) *DockerBuild {
	b.labels = append(b.labels, imageLabel)
	return b
}

func (b *DockerBuild) NoCache() *DockerBuild {
	b.noCache = true
	return b
}

func (b *DockerBuild) Pull() *DockerBuild {
	b.pullRefs = true
	return b
}

func (b *DockerBuild) Secret(id, value string) *DockerBuild {
	b.buildSecrets[id] = value
	return b
}

func (b *DockerBuild) Tag(imageTag string) *DockerBuild {
	b.imageTag = imageTag
	return b
}

func (b *DockerBuild) Tagf(tagFormat string, args ...any) *DockerBuild {
	b.imageTag = fmt.Sprintf(tagFormat, args...)
	return b
}

// Command resolves the docker CLI, builds the build Command, and returns it for further customization.
func (b *DockerBuild) Command() *modmake.Command {
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

func (b *DockerBuild) Task() modmake.Task {
	return b.Run
}

func (b *DockerBuild) Run(ctx context.Context) error {
	return b.Command().Run(ctx)
}

func (b *DockerBuild) String() string {
	return b.Command().String()
}
