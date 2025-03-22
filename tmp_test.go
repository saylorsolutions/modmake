package modmake

import (
	"context"
	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err)
	assert.False(t, dir.Exists())

	dir = ""
	assert.Panics(t, func() {
		_ = TempDir("TestTempDir-*", func(tmp PathString) Task {
			dir = tmp
			assert.True(t, tmp.Exists(), "Again, tmp should exist at this point")
			panic("panic while producing Task")
		}).Run(context.Background())
	})
	assert.NotEqual(t, "", dir.String())
	assert.False(t, dir.Exists(), "Temp directory should still be removed if the Task producer function panics")
}
