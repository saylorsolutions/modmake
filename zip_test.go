package modmake

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestZipArchive_Extract(t *testing.T) {
	tmp, err := os.MkdirTemp("", "ZipArchive_Extract-*")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmp))
	}()
	inputPath := Path(tmp, "input.txt")
	require.NoError(t, os.WriteFile(inputPath.String(), []byte("A test file"), 0600))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	zipPath := Path(tmp, "test.zip")
	err = Zip(zipPath).AddFileWithPath(inputPath, "input.txt").Create().Run(ctx)
	require.NoError(t, err)
	require.NoError(t, os.Remove(inputPath.String()))
	require.NoError(t, os.WriteFile(inputPath.String(), []byte("A new message"), 0600))
	require.NoError(t, Zip(zipPath).AddFileWithPath(inputPath, "input.txt").Update().Run(ctx))

	require.NoError(t, Zip(zipPath).Extract(Path(tmp)).Run(ctx), "Failed to extract directory")
	fi, err := os.Stat(inputPath.String())
	require.NoError(t, err, "Failed to stat the extracted file")
	assert.False(t, fi.IsDir(), "Should be a file, not a directory")
	data, err := os.ReadFile(inputPath.String())
	require.NoError(t, err)
	assert.Equal(t, "A new message", string(data))
}
