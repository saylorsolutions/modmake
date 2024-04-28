package git

import (
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func TestTools_BranchName(t *testing.T) {
	branch := BranchName()
	t.Log("branch:", branch)
	assert.Equal(t, strings.TrimSpace(branch), branch)
	assert.True(t, len(branch) > 0)
}

func TestTools_CommitHash(t *testing.T) {
	hash := CommitHash()
	t.Log("hash:", hash)
	assert.Equal(t, strings.TrimSpace(hash), hash)
	assert.True(t, len(hash) > 0)
}
