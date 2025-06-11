package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestParseSeparate(t *testing.T) {
	flags := setupFlags()
	err := flags.Parse([]string{"-e", "VAR=VAL", "-e", "OTHER=VAL", "--", "--skip-dependencies", "client:build"})
	require.NoError(t, err)

	assert.False(t, flags.help)
	assert.Equal(t, []string{"VAR=VAL", "OTHER=VAL"}, flags.envVars)

	args := flags.Args()
	assert.Len(t, args, 2)
	assert.Equal(t, "--skip-dependencies", flags.Arg(0))
	assert.Equal(t, "client:build", flags.Arg(1))
}
