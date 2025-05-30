package minify_test

import (
	_ "embed"
	mm "github.com/saylorsolutions/modmake"
	"github.com/saylorsolutions/modmake/pkg/minify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

type workingDir struct {
	tmp        mm.PathString
	jsSource   mm.PathString
	cssSource  mm.PathString
	htmlSource mm.PathString
	svgSource  mm.PathString
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
	jsSource := mm.Path(tmp, "test.js")
	require.NoError(t, mm.Path("test.js").CopyTo(jsSource))
	cssSource := mm.Path(tmp, "test.css")
	require.NoError(t, mm.Path("test.css").CopyTo(cssSource))
	htmlSource := mm.Path(tmp, "index.html")
	require.NoError(t, mm.Path("index.html").CopyTo(htmlSource))
	svgSource := mm.Path(tmp, "test.svg")
	require.NoError(t, mm.Path("test.svg").CopyTo(svgSource))
	return &workingDir{
		tmp:        tmpPath,
		jsSource:   jsSource,
		cssSource:  cssSource,
		htmlSource: htmlSource,
		svgSource:  svgSource,
	}
}

func getFilesWithExt(t *testing.T, dir mm.PathString, ext string) []string {
	t.Helper()
	var files []string
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
		if filepath.Ext(path) == ext {
			files = append(files, path)
		}
		return nil
	})
	require.NoError(t, err)
	return files
}

func TestMinifier_MapFile(t *testing.T) {
	work := setupWorkingDirectory(t)
	mappingFile := work.tmp.Join("assets.go")
	assetDir := work.tmp.Join("content")
	minifier, err := minify.New(mappingFile, "content",
		minify.Version("latest"),
		minify.HashDigits(6),
		minify.ClearBeforeWrite(),
		minify.PackageName("testing"),
	)
	require.NoError(t, err)

	b := mm.NewBuild()
	minifier.Apply(b)
	minifier.
		MapFile(work.jsSource).
		MapFile(work.cssSource).
		MapFile(work.htmlSource).
		MapFile(work.svgSource)
	require.NoError(t, b.ExecuteErr("minify"))

	require.True(t, assetDir.IsDir())
	require.True(t, work.tmp.IsDir())
	jsFiles := getFilesWithExt(t, assetDir, ".js")
	require.Len(t, jsFiles, 1)
	assert.True(t, regexp.MustCompile(`^test-[a-f0-9]{6}\.js$`).MatchString(filepath.Base(jsFiles[0])))
	cssFiles := getFilesWithExt(t, assetDir, ".css")
	require.Len(t, cssFiles, 1)
	assert.True(t, regexp.MustCompile(`^test-[a-f0-9]{6}\.css$`).MatchString(filepath.Base(cssFiles[0])))
	htmlFiles := getFilesWithExt(t, assetDir, ".html")
	require.Len(t, htmlFiles, 1)
	assert.True(t, regexp.MustCompile(`^index-[a-f0-9]{6}\.html$`).MatchString(filepath.Base(htmlFiles[0])))
	svgFiles := getFilesWithExt(t, assetDir, ".svg")
	require.Len(t, svgFiles, 1)
	assert.True(t, regexp.MustCompile(`^test-[a-f0-9]{6}\.svg$`).MatchString(filepath.Base(svgFiles[0])))

	require.True(t, mappingFile.IsFile())
	content, err := mappingFile.ReadFile()
	require.NoError(t, err)
	require.Positive(t, len(content))
	strContent := string(content)
	assert.Contains(t, strContent, "package testing")
	assert.Contains(t, strContent, "//go:embed content/")
	assert.Contains(t, strContent, "Testjs []byte")
	assert.Contains(t, strContent, "TestjsName = \"")
	assert.Contains(t, strContent, "Testcss []byte")
	assert.Contains(t, strContent, "TestcssName = \"")
	assert.Contains(t, strContent, "Indexhtml []byte")
	assert.Contains(t, strContent, "IndexhtmlName = \"")
	assert.Contains(t, strContent, "Testsvg []byte")
	assert.Contains(t, strContent, "TestsvgName = \"")
}
