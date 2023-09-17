package modmake

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestZipArchive_Extract(t *testing.T) {
	tmp, err := os.MkdirTemp("", "ZipArchive_Extract-*")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmp))
	}()
	inputPath := filepath.Join(tmp, "input.txt")
	require.NoError(t, os.WriteFile(inputPath, []byte("A test file"), 0644))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	zipPath := filepath.Join(tmp, "test.zip")
	err = Zip(zipPath).AddFileWithPath(inputPath, "input.txt").Create().Run(ctx)
	require.NoError(t, err)
	require.NoError(t, os.Remove(inputPath))
	require.NoError(t, os.WriteFile(inputPath, []byte("A new message"), 0644))
	require.NoError(t, Zip(zipPath).AddFileWithPath(inputPath, "input.txt").Update().Run(ctx))

	assert.NoError(t, Zip(zipPath).Extract(tmp).Run(ctx), "Failed to extract directory")
	fi, err := os.Stat(inputPath)
	assert.NoError(t, err, "Failed to stat the extracted file")
	assert.False(t, fi.IsDir(), "Should be a file, not a directory")
	data, err := os.ReadFile(inputPath)
	assert.NoError(t, err)
	assert.Equal(t, "A new message", string(data))
}
