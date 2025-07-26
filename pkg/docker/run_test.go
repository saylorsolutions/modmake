package docker_test

import (
	"fmt"
	"github.com/saylorsolutions/modmake"
	"github.com/saylorsolutions/modmake/pkg/docker"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDockerRun_String(t *testing.T) {
	inst := testDocker(t)
	tests := map[string]struct {
		run      *docker.Run
		expected string
	}{
		"Base": {
			run:      inst.Run("postgres:latest"),
			expected: "echo run postgres:latest",
		},
		"Port binding": {
			run:      inst.Run("postgres:latest").ExposePort(5432, 5432),
			expected: "echo run -p 5432:5432 postgres:latest",
		},
		"Bind mount": {
			run: testDocker(t).Run("postgres:latest").
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
