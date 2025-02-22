package docparser

import "sort"

type Linker interface {
	LinkID() string
}

type Module struct {
	Packages map[string]*PackageDocs
	Docs     string
	parser   *Parser
}

func NewModule() *Module {
	return &Module{
		Packages: map[string]*PackageDocs{},
		parser:   New(),
	}
}

func (m *Module) ParsePackageDir(pkgDir string) error {
	docs, err := m.parser.ParsePackageDir(pkgDir)
	if err != nil {
		return err
	}
	m.Packages[docs.ImportName] = docs
	return nil
}

func (m *Module) SortedPackageDocs() []*PackageDocs {
	return sortedMap(m.Packages, func(docs *PackageDocs) string {
		return docs.ImportName
	})
}

// PackageDocs collects the public symbols of a Go package into one struct.
type PackageDocs struct {
	PackageName string
	ImportName  string
	Docs        string
	Constants   []*Constant
	Variables   []*Variable
	Functions   map[string]*Function
	Types       map[string]*Type
}

func (d *PackageDocs) SortedConstants() []*Constant {
	if len(d.Constants) == 0 {
		return nil
	}
	return sortedSlice(d.Constants, func(val *Constant) string {
		return val.ConstantName
	})
}

func (d *PackageDocs) SortedVariables() []*Variable {
	if len(d.Variables) == 0 {
		return nil
	}
	return sortedSlice(d.Variables, func(val *Variable) string {
		return val.VarName
	})
}

func (d *PackageDocs) SortedFunctions() []*Function {
	if len(d.Functions) == 0 {
		return nil
	}
	return sortedMap(d.Functions, func(function *Function) string {
		return function.FunctionName
	})
}

func (d *PackageDocs) SortedTypes() []*Type {
	if len(d.Types) == 0 {
		return nil
	}
	return sortedMap(d.Types, func(t *Type) string {
		return t.TypeName
	})
}

type Constant struct {
	ConstantName string
	Declaration  string
	Docs         string
}

func (c *Constant) LinkID() string {
	return "const_" + c.ConstantName
}

type Variable struct {
	VarName     string
	Declaration string
	Docs        string
}

func (v *Variable) LinkID() string {
	return "var_" + v.VarName
}

type Function struct {
	FunctionName string
	Signature    string
	Docs         string
}

func (f *Function) LinkID() string {
	return "func_" + f.FunctionName
}

type Type struct {
	TypeName    string
	Declaration string
	Docs        string
	ConstVals   string
	Methods     map[string]*Method
}

func (t *Type) SortedMethods() []*Method {
	if len(t.Methods) == 0 {
		return nil
	}
	return sortedMap(t.Methods, func(method *Method) string {
		return method.MethodName
	})
}

func (t *Type) LinkID() string {
	return t.TypeName
}

type Method struct {
	MethodName string
	TypeName   string
	Signature  string
	Docs       string
}

func (m *Method) LinkID() string {
	return m.TypeName + "_" + m.MethodName
}

type nameSorter[T any] struct {
	accessor func(element T) string
	vals     []T
}

func (s nameSorter[T]) Len() int {
	return len(s.vals)
}

func (s nameSorter[T]) Less(i, j int) bool {
	return s.accessor(s.vals[i]) < s.accessor(s.vals[j])
}

func (s nameSorter[T]) Swap(i, j int) {
	s.vals[i], s.vals[j] = s.vals[j], s.vals[i]
}

func sortedSlice[T any](vals []T, accessor func(val T) string) []T {
	sort.Sort(&nameSorter[T]{
		vals:     vals,
		accessor: accessor,
	})
	return vals
}

func sortedMap[T any](vals map[string]T, accessor func(T) string) []T {
	slice := make([]T, len(vals))
	i := 0
	for _, val := range vals {
		slice[i] = val
		i++
	}
	return sortedSlice(slice, accessor)
}
