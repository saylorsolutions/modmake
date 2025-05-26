package modmake

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTempDir(t *testing.T) {
	var dir PathString
	err := TempDir("TestTempDir-*", func(tmp PathString) Task {
		assert.True(t, tmp.Exists(), "temp dir should be created before the produced task is run")
		return Plain(func() {
			dir = tmp
			assert.True(t, tmp.Exists())
		})
	}).Run(context.Background())
	require.NoError(t, err)
	assert.False(t, dir.Exists())

	dir = ""
	assert.Panics(t, func() {
		_ = TempDir("TestTempDir-*", func(tmp PathString) Task {
			dir = tmp
			assert.True(t, tmp.Exists(), "Again, tmp should exist at this point")
			panic("panic while producing Task")
		}).Run(context.Background())
	})
	assert.NotEmpty(t, dir.String())
	assert.False(t, dir.Exists(), "Temp directory should still be removed if the Task producer function panics")
}
