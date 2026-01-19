package docker_test

import (
	"fmt"
	"testing"

	"github.com/saylorsolutions/modmake"
	"github.com/saylorsolutions/modmake/pkg/docker"
	"github.com/stretchr/testify/assert"
)

func TestDockerRun_String(t *testing.T) {
	testDocker(t)
	tests := map[string]struct {
		run      *docker.Runner
		expected string
	}{
		"Base": {
			run:      docker.Run("postgres:latest"),
			expected: "echo run postgres:latest",
		},
		"Port binding": {
			run:      docker.Run("postgres:latest").ExposePort(5432, 5432),
			expected: "echo run -p 5432:5432 postgres:latest",
		},
		"Bind mount": {
			run: docker.Run("postgres:latest").
				ExposePort(5432, 5432).
				BindMount("./mount", "/var/lib/postgresql/data"),
			expected: fmt.Sprintf("echo run -v %s:/var/lib/postgresql/data -p 5432:5432 postgres:latest",
				modmake.Path("./mount").Abs()),
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.run.String())
		})
	}
}
