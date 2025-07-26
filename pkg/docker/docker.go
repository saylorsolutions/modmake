/*
Package docker provides a means to programmatically call the docker CLI for building images and running containers.

By default, the PATH will be consulted to locate the docker CLI, but this can be overridden by setting the MODMAKE_DOCKER_PATH environment variable.
This is done in testing to use 'echo' instead.

Most flags are supported, but some less-common flags are not.
If you require an unsupported flag, then build your base command and call Command to set custom arguments on the Modmake Command that is returned.
*/
package docker

import (
	"github.com/saylorsolutions/modmake"
	"os"
	"os/exec"
)

const (
	EnvDockerPath = "MODMAKE_DOCKER_PATH"
)

type Inst struct {
	dockerPath modmake.PathString
}

func Docker() *Inst {
	dockerOverride, ok := os.LookupEnv(EnvDockerPath)
	if ok && len(dockerOverride) > 0 {
		return &Inst{
			dockerPath: modmake.Path(dockerOverride),
		}
	}
	path, err := exec.LookPath("docker")
	if err != nil {
		panic(err)
	}
	return &Inst{
		dockerPath: modmake.Path(path),
	}
}
