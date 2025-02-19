package templates

import "github.com/a-h/templ"

type Content struct {
	Sections []*Section
}

func (c *Content) AddSection(s ...*Section) *Content {
	c.Sections = append(c.Sections, s...)
	return c
}

type Section struct {
	ID, HeaderText string
	Prose          []templ.Component
	SubSections    []*SubSection
}

func NewSection(id, header string, prose ...templ.Component) *Section {
	return &Section{
		ID:         id,
		HeaderText: header,
		Prose:      prose,
	}
}

func (s *Section) AddSubSection(id, header string, prose ...templ.Component) *Section {
	s.SubSections = append(s.SubSections, &SubSection{
		ID:         id,
		HeaderText: header,
		Prose:      prose,
	})
	return s
}

type SubSection struct {
	ID, HeaderText string
	Prose          []templ.Component
}
