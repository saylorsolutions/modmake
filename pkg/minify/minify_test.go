package minify

import (
	"context"
	_ "embed"
	mm "github.com/saylorsolutions/modmake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

var (
	//go:embed test.js
	testJS []byte
)

type workingDir struct {
	tmp      mm.PathString
	jsSource mm.PathString
}

func setupWorkingDirectory(t *testing.T) *workingDir {
	t.Helper()
	tmp, err := os.MkdirTemp("", "minify_workdir-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.RemoveAll(tmp))
	})
	tmpPath := mm.Path(tmp)
	assert.True(t, tmpPath.IsDir())
	source := mm.Path(tmp, "test.js")
	require.NoError(t, source.WriteFile(testJS, 0600))
	return &workingDir{
		tmp:      tmpPath,
		jsSource: source,
	}
}

func getJSFiles(t *testing.T, dir mm.PathString) []string {
	t.Helper()
	var jsFiles []string
	dirStr := dir.String()
	err := filepath.WalkDir(dirStr, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == dirStr {
			return nil
		}
		if d.IsDir() {
			return filepath.SkipDir
		}
		if filepath.Ext(path) == ".js" {
			jsFiles = append(jsFiles, path)
		}
		return nil
	})
	require.NoError(t, err)
	return jsFiles
}

func TestMinify(t *testing.T) {
	work := setupWorkingDirectory(t)
	mappingFile := work.tmp.Join("mapping")
	conf, err := NewConfig(mappingFile, "assets",
		ClearBeforeWrite(),
		MappingFilePackage("testing"),
	)
	require.NoError(t, err)
	conf.tasks = conf.tasks.Then(conf.invokeMinify(work.jsSource))
	require.NoError(t, conf.Run(context.Background()))
	assert.True(t, work.jsSource.IsFile())
	assert.True(t, work.tmp.IsDir())
	assert.True(t, conf.assetDir.IsDir())
	files := getJSFiles(t, conf.assetDir)
	require.Len(t, files, 1)
	assert.Equal(t, ".js", filepath.Ext(files[0]))
}
