package git

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTools_BranchName(t *testing.T) {
	g := NewTools()
	b := g.BranchName()
	assert.True(t, len(b) > 0)
}

func TestTools_CommitHash(t *testing.T) {
	g := NewTools()
	h := g.CommitHash()
	assert.True(t, len(h) > 0)
}
