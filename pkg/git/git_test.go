package git

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTools_BranchName(t *testing.T) {
	branch := BranchName()
	t.Log("branch:", branch)
	assert.Equal(t, strings.TrimSpace(branch), branch)
	assert.NotEmpty(t, branch)
}

func TestTools_CommitHash(t *testing.T) {
	hash := CommitHash()
	t.Log("hash:", hash)
	assert.Equal(t, strings.TrimSpace(hash), hash)
	assert.NotEmpty(t, hash)
}
