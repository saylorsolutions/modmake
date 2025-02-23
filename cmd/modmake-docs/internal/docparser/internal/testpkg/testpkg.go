// Package testpkg is here just to test go doc introspection and output parsing.
package testpkg

// SomeEnum is a custom type with const enum values
type SomeEnum int

// SomeVar is a test var
var SomeVar string

const (
	SomeConst float64 = 3.14 // SomeConst is a test constant
)
const AnotherConst = "const string"

const (
	A SomeEnum = iota + 1 // A
	B                     // B
	C                     // C
)
const D SomeEnum = 4

// SomeType is a test struct type.
type SomeType struct{}

// DoTheThing does the thing.
func (t *SomeType) DoTheThing() {}

// DoTheThing does the thing.
func DoTheThing() {}

func shouldNotAppearInResults() {}
