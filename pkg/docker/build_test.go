package docker_test

import (
	"github.com/saylorsolutions/modmake/pkg/docker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func testDocker(t *testing.T) *docker.Inst {
	require.NoError(t, os.Setenv(docker.EnvDockerPath, "echo"))
	t.Cleanup(func() {
		assert.NoError(t, os.Unsetenv(docker.EnvDockerPath))
	})
	return docker.Docker()
}

func TestDockerBuild_String(t *testing.T) {
	d := testDocker(t)
	tests := map[string]struct {
		build    *docker.Build
		expected string
	}{
		"Basic build": {
			build:    d.Build(""),
			expected: "echo build -f Dockerfile .",
		},
		"Tagged": {
			build:    d.Build("").Tag("my-image:0.1.0"),
			expected: "echo build -f Dockerfile -t my-image:0.1.0 .",
		},
		"Formatted tag": {
			build:    d.Build("").Tagf("my-image:%s", "0.1.0"),
			expected: "echo build -f Dockerfile -t my-image:0.1.0 .",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.build.String())
		})
	}
}
