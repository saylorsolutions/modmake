package templates

import _ "embed"

var (
	//go:embed intro.md
	intro []byte
)

func IntroSection(params Params) *Section {
	return NewSection("introduction", "Introduction",
		Markdown(intro),
	).AddSubSection("project-commitments", "Project Commitments",
		Intro_ProjectCommitments(params),
	).AddSubSection("getting-started", "Getting Started",
		Intro_GettingStarted(params),
	)
}
