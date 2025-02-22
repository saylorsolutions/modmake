package templates

import (
	"fmt"
	"github.com/a-h/templ"
	"github.com/saylorsolutions/modmake/cmd/modmake-docs/internal/docparser"
)

func UtilitiesSection() *Section {
	return NewSection("utilities", "Utility Functions", Utilities()).
		AddSubSection("task-helpers", "Task Helpers", Utilities_TaskHelpers()).
		AddSubSection("go-tools", "Go Tools", Utilities_GoTools()).
		AddSubSection("compression", "Compression", Utilities_Compression()).
		AddSubSection("git", "Git Functions", Utilities_Git())
}

func PkgDocLink(params Params, pkg *docparser.PackageDocs, linker docparser.Linker) templ.SafeURL {
	return templ.SafeURL(params.Qual(fmt.Sprintf("/godoc/%s#%s", pkg.ImportName, linker.LinkID())))
}

func PkgLink(params Params, pkg *docparser.PackageDocs) templ.SafeURL {
	return templ.SafeURL(params.Qual(fmt.Sprintf("/godoc/%s", pkg.ImportName)))
}
