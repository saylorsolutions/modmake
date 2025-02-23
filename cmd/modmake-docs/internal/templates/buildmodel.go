package templates

import _ "embed"

var (
	//go:embed buildmodel/pinlatest.md
	pinLatest []byte
	//go:embed buildmodel/buildsteps.md
	buildSteps []byte
	//go:embed buildmodel/composability.md
	composability []byte
)

func BuildModelSection(params Params) *Section {
	return NewSection("build-model", "Build Model",
		BuildModel(),
	).
		AddSubSection("pin-latest", "PinLatest", Markdown(pinLatest)).
		AddSubSection("steps", "Steps", Markdown(buildSteps)).
		AddSubSection("tasks", "Tasks", BuildModel_Tasks()).
		AddSubSection("invocation", "Invocation", BuildModel_Invocation(params)).
		AddSubSection("composability", "Build Composability", Markdown(composability)).
		AddSubSection("api-patterns", "API Patterns", BuildModel_APIPatterns())
}
