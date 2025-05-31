package minify

import (
	mm "github.com/saylorsolutions/modmake"
	"path/filepath"
)

// MapFile will attempt to minify the given file and add embed entries into the configured mapping file.
func (mini *Minifier) MapFile(source mm.PathString) *Minifier {
	mini.tasks = mini.tasks.Then(mini.minAndMapFile(source))
	return mini
}

// MapJSBundle will attempt to bundle one or more JS files into one bundle file, and add embed entries into the configured mapping file.
// Source files should have the ".js" file extension.
func (mini *Minifier) MapJSBundle(bundleName string, sources ...mm.PathString) *Minifier {
	return mini.mapBundle(bundleName, ".js", sources...)
}

// MapCSSBundle will attempt to bundle one or more CSS files into one bundle file, and add embed entries into the configured mapping file.
// Source files should have the ".css" file extension.
func (mini *Minifier) MapCSSBundle(bundleName string, sources ...mm.PathString) *Minifier {
	return mini.mapBundle(bundleName, ".css", sources...)
}

func (mini *Minifier) mapBundle(bundleName string, ext string, sources ...mm.PathString) *Minifier {
	if len(bundleName) == 0 {
		panic("missing bundle name")
	}
	if bext := filepath.Ext(bundleName); bext != ext {
		bundleName += ext
	}
	for _, source := range sources {
		if source.Ext() != ext {
			mini.tasks = mini.tasks.Then(mm.Error("bundle file '%s' doesn't have the expected file extension %s", source, ext))
			return mini
		}
	}
	mini.tasks = mini.tasks.Then(mini.minAndMapBundle(bundleName, sources...))
	return mini
}
