package minify

import mm "github.com/saylorsolutions/modmake"

func (mini *Minifier) MapFile(source mm.PathString) *Minifier {
	mini.tasks = mini.tasks.Then(mini.minAndMapFile(source))
	return mini
}
