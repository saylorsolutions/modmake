package modmake

import (
	"context"
	"github.com/bitfield/script"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestExecScript(t *testing.T) {
	err := Exec(
		func(_ context.Context) error {
			_, err := script.File("script.go").WriteFile("script.go.copy")
			return err
		},
		func(_ context.Context) error {
			_, err := script.File("script.go.copy").Stdout()
			return err
		},
		func(_ context.Context) error {
			return os.Remove("script.go.copy")
		},
	).Run(context.Background())
	assert.NoError(t, err)
}

func TestIfNotExists(t *testing.T) {
	var executed bool
	err := IfNotExists("script.go", RunnerFunc(func(ctx context.Context) error {
		executed = true
		return nil
	})).Run(context.Background())
	assert.NoError(t, err)
	assert.False(t, executed)
}

func TestIfExists(t *testing.T) {
	var executed bool
	err := IfExists("script.go", RunnerFunc(func(ctx context.Context) error {
		executed = true
		return nil
	})).Run(context.Background())
	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestCopyFile_Relative(t *testing.T) {
	dir, err := os.MkdirTemp("", "testcopyfile-*")
	require.NoError(t, err)
	t.Logf("Using temp dir: %s", dir)
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	fname := filepath.Join(dir, "test.file")
	f, err := os.Create(fname)
	require.NoError(t, err)
	require.NotNil(t, f)
	_, err = f.WriteString("Hello!")
	assert.NoError(t, err)
	assert.NoError(t, f.Close())

	assert.NoError(t, CopyFile("test.file", "test.file2").Run(context.WithValue(context.Background(), CtxWorkdir, dir)))
	assert.NoError(t, script.IfExists(fname+"2").Error())
	assert.NoError(t, script.IfExists(fname).Error())

	data, err := script.File(fname + "2").String()
	assert.NoError(t, err)
	assert.Equal(t, "Hello!", data)
}

func TestCopyFile_Abs(t *testing.T) {
	dir, err := os.MkdirTemp("", "testcopyfile-*")
	require.NoError(t, err)
	t.Logf("Using temp dir: %s", dir)
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	fname := filepath.Join(dir, "test.file")
	f, err := os.Create(fname)
	require.NoError(t, err)
	require.NotNil(t, f)
	_, err = f.WriteString("Hello!")
	assert.NoError(t, err)
	assert.NoError(t, f.Close())

	assert.NoError(t, CopyFile(fname, fname+"2").Run(context.WithValue(context.Background(), CtxWorkdir, dir)))
	assert.NoError(t, script.IfExists(fname+"2").Error())
	assert.NoError(t, script.IfExists(fname).Error())

	data, err := script.File(fname + "2").String()
	assert.NoError(t, err)
	assert.Equal(t, "Hello!", data)
}

func TestMoveFile(t *testing.T) {
	dir, err := os.MkdirTemp("", "testmovefile-*")
	require.NoError(t, err)
	t.Logf("Using temp dir: %s", dir)
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	fname := filepath.Join(dir, "test.file")
	f, err := os.Create(fname)
	require.NoError(t, err)
	require.NotNil(t, f)
	_, err = f.WriteString("Hello!")
	assert.NoError(t, err)
	assert.NoError(t, f.Close())

	assert.NoError(t, MoveFile(fname, fname+"2").Run(context.WithValue(context.Background(), CtxWorkdir, dir)))
	assert.NoError(t, script.IfExists(fname+"2").Error())
	assert.ErrorIs(t, script.IfExists(fname).Error(), os.ErrNotExist)

	data, err := script.File(fname + "2").String()
	assert.NoError(t, err)
	assert.Equal(t, "Hello!", data)
}

func TestGetWorkdir(t *testing.T) {
	workdir, err := getWorkdir(context.Background())
	assert.ErrorIs(t, err, ErrMissingWorkdir)
	assert.Equal(t, "", workdir)
	workdir, err = getWorkdir(context.WithValue(context.Background(), CtxWorkdir, "./test"))
	assert.NoError(t, err)
	assert.Equal(t, "./test", workdir)
}

func TestRelativeToWorkdir(t *testing.T) {
	rel := relativeToWorkdir("./test", "a")
	if runtime.GOOS == "windows" {
		assert.Equal(t, "test\\a", rel)
	} else {
		assert.Equal(t, "test/a", rel)
	}
	abs, err := filepath.Abs(".")
	assert.NoError(t, err)
	assert.Equal(t, abs, relativeToWorkdir("./test", abs))
}
