package templates

func ModmakeCLISection(params Params) *Section {
	return NewSection("modmake-cli", "Modmake CLI", ModmakeCLI(params)).
		AddSubSection("resolution", "Finding Your Build", ModmakeCLI_Resolution()).
		AddSubSection("invocation", "CLI Invocation", ModmakeCLI_Invocation()).
		AddSubSection("file-watching", "File Watching", ModmakeCLI_Watch()).
		AddSubSection("watch-recipes", "Helpful Recipes", ModmakeCLI_WatchRecipes())
}
