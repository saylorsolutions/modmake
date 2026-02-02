/*
Package docker provides a means to programmatically call the docker CLI for building images and running containers.

By default, the PATH will be consulted to locate the docker CLI, but this can be overridden by setting the MODMAKE_DOCKER_PATH environment variable.
This is done in testing to use 'echo' instead.

Most flags are supported, but some less-common flags are not.
If you require an unsupported flag, then build your base command and call Command to set custom arguments on the Modmake Command that is returned.
*/
package docker

import (
	"context"
	"os"
	"os/exec"

	"github.com/saylorsolutions/modmake"
)

const (
	EnvDockerPath = "MODMAKE_DOCKER_PATH"
)

func resolveDockerPath() modmake.PathString {
	dockerOverride, ok := os.LookupEnv(EnvDockerPath)
	if ok && len(dockerOverride) > 0 {
		return modmake.Path(dockerOverride)
	}
	path, err := exec.LookPath("docker")
	if err != nil {
		panic(err)
	}
	return modmake.Path(path)
}

// Do allows making arbitrary calls to the docker CLI.
func Do(subcommand string, args ...string) modmake.Task {
	return func(ctx context.Context) error {
		return modmake.Exec(append([]string{resolveDockerPath().String(), subcommand}, args...)...).Run(ctx)
	}
}

// Tag creates a Task to tag a Docker image.
func Tag(sourceTag, targetTag string) modmake.Task {
	return Do("tag", sourceTag, targetTag)
}
