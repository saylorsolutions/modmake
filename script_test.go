package modmake

import (
	"context"
	"fmt"
	"github.com/bitfield/script"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestExecScript(t *testing.T) {
	err := Script(
		Task(func(_ context.Context) error {
			_, err := script.File("script.go").WriteFile("script.go.copy")
			return err
		}),
		Task(func(_ context.Context) error {
			_, err := script.File("script.go.copy").Stdout()
			return err
		}),
		Task(func(_ context.Context) error {
			return os.Remove("script.go.copy")
		}),
	).Run(context.Background())
	assert.NoError(t, err)
}

func TestIfNotExists(t *testing.T) {
	var executed bool
	err := IfNotExists("script.go", Task(func(ctx context.Context) error {
		executed = true
		return nil
	})).Run(context.Background())
	assert.NoError(t, err)
	assert.False(t, executed)
}

func TestIfExists(t *testing.T) {
	var executed bool
	err := IfExists("script.go", Task(func(ctx context.Context) error {
		executed = true
		return nil
	})).Run(context.Background())
	assert.NoError(t, err)
	assert.True(t, executed)
}

func TestCopyFile_Abs(t *testing.T) {
	dir, err := os.MkdirTemp("", "testcopyfile-*")
	require.NoError(t, err)
	t.Logf("Using temp dir: %s", dir)
	defer func() {
		_ = os.RemoveAll(dir)
	}()
	fname := Path(dir, "test.file")
	fname2 := fname + "2"
	f, err := os.Create(fname.String())
	require.NoError(t, err)
	require.NotNil(t, f)
	_, err = f.WriteString("Hello!")
	assert.NoError(t, err)
	assert.NoError(t, f.Close())

	assert.NoError(t, CopyFile(fname, fname2).Run(context.Background()))
	assert.True(t, fname2.Exists())
	assert.True(t, fname.Exists())

	data, err := script.File(fname2.String()).String()
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
	fname := Path(dir, "test.file")
	fname2 := fname + "2"
	f, err := os.Create(fname.String())
	require.NoError(t, err)
	require.NotNil(t, f)
	_, err = f.WriteString("Hello!")
	assert.NoError(t, err)
	assert.NoError(t, f.Close())

	assert.NoError(t, MoveFile(fname, fname2).Run(context.Background()))
	assert.True(t, fname2.Exists())
	assert.False(t, fname.Exists())

	data, err := script.File(fname2.String()).String()
	assert.NoError(t, err)
	assert.Equal(t, "Hello!", data)
}

func ExampleIfError() {
	canError := Error("An error occurred!")
	err := IfError(canError, Print("Error handled")).Run(context.Background())
	if err != nil {
		fmt.Println("Error should not have been returned:", err)
	}
	// Output:
}

func TestRemoveDir(t *testing.T) {
	tmp, err := os.MkdirTemp("", "RemoveDir-*")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmp))
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dir := Path(tmp, "dir")
	err = Script(
		IfExists(dir, Error("Directory '%s' should not already exist", dir)),
		Mkdir(dir, 0755),
		IfNotExists(dir, Error("Directory '%s' should exist", dir)),
		Chdir(Path(tmp), Script(
			RemoveDir("dir"),
			IfExists("dir", Error("RemoveDir should have reported an error")),
		)),
	).Run(ctx)
	assert.NoError(t, err)
}

func TestRemove(t *testing.T) {
	tmp, err := os.MkdirTemp("", "RemoveDir-*")
	require.NoError(t, err)
	defer func() {
		assert.NoError(t, os.RemoveAll(tmp))
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	file := Path(tmp, "file.txt")

	err = Script(
		IfExists(file, Error("File '%s' should not already exist", file)),
		Task(func(ctx context.Context) error {
			return os.WriteFile(file.String(), []byte("Some text"), 0600)
		}),
		IfNotExists(file, Error("File '%s' should exist", file)),
		Chdir(Path(tmp), Script(
			RemoveFile("file.txt"),
			IfExists("file.txt", Error("RemoveFile should have reported an error")),
		)),
	).Run(ctx)
	assert.NoError(t, err)
}
