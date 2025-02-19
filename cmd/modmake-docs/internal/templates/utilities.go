package templates

func UtilitiesSection() *Section {
	return NewSection("utilities", "Utility Functions", Utilities()).
		AddSubSection("task-helpers", "Task Helpers", Utilities_TaskHelpers()).
		AddSubSection("go-tools", "Go Tools", Utilities_GoTools()).
		AddSubSection("compression", "Compression", Utilities_Compression()).
		AddSubSection("git", "Git Functions", Utilities_Git())
}
