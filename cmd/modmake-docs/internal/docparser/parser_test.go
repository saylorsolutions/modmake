package docparser_test

import (
	"encoding/json"
	"github.com/saylorsolutions/modmake/cmd/modmake-docs/internal/docparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
)

func TestParser_ParsePackageDir(t *testing.T) {
	p := docparser.New()
	docs, err := p.ParsePackageDir(".")
	assert.NoError(t, err)
	data, err := json.MarshalIndent(docs, "", "  ")
	assert.NoError(t, err)
	t.Log("\n" + string(data))
	assert.NotNil(t, docs.Types["Constant"])
	assert.NotNil(t, docs.Types["Function"])
	assert.NotNil(t, docs.Types["Method"])
	assert.NotNil(t, docs.Types["Module"])
	assert.NotNil(t, docs.Types["PackageDocs"])
	assert.NotNil(t, docs.Types["Parser"])
	assert.NotNil(t, docs.Types["Type"])
	assert.NotNil(t, docs.Types["Variable"])

	assert.NotNil(t, docs.Types["Parser"].Methods["ParsePackageDir"])
}

func TestParser_ParsePackageDir_Internal(t *testing.T) {
	p := docparser.New()
	docs, err := p.ParsePackageDir(filepath.Join("internal", "testpkg"))
	require.NoError(t, err)
	require.NotNil(t, docs)
	data, err := json.MarshalIndent(docs, "", "  ")
	assert.NoError(t, err)
	t.Log("\n" + string(data))

	assert.Equal(t, "AnotherConst", docs.Constants[0].ConstantName)
	assert.Equal(t, "SomeConst", docs.Constants[1].ConstantName)
	assert.Equal(t, "SomeConst is a test constant", docs.Constants[1].Docs)
	assert.Equal(t, "SomeVar", docs.Variables[0].VarName)
	assert.Equal(t, "SomeVar is a test var", docs.Variables[0].Docs)
	assert.NotNil(t, docs.Types["SomeEnum"])
	assert.Equal(t, "SomeEnum is a custom type with const enum values", docs.Types["SomeEnum"].Docs)
	assert.NotNil(t, docs.Types["SomeType"])
	assert.Equal(t, "SomeType is a test struct type.", docs.Types["SomeType"].Docs)
	assert.NotNil(t, docs.Types["SomeType"].Methods["DoTheThing"])
	assert.Equal(t, "DoTheThing does the thing.", docs.Types["SomeType"].Methods["DoTheThing"].Docs)
	assert.NotNil(t, docs.Functions["DoTheThing"])
	assert.Equal(t, "DoTheThing does the thing.", docs.Functions["DoTheThing"].Docs)
	assert.Nil(t, docs.Functions["shouldNotAppearInResults"])
}
