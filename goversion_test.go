//go:build it

package modmake

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGoTools_PinLatest(t *testing.T) {
	var (
		inst *GoTools
		buf  bytes.Buffer
	)
	Go().InvalidateCache()
	assert.NoError(t, Go().Command("version").Output(&buf).Run(context.Background()))
	assert.NotContains(t, buf.String(), "1.6.4")

	buf.Reset()
	assert.NotPanics(t, func() {
		inst = Go().PinLatestV1(6)
	})
	assert.NoError(t, inst.Command("version").Output(&buf).Run(context.Background()))
	assert.Contains(t, buf.String(), "1.6.4")

	buf.Reset()
	assert.NoError(t, Go().Command("version").Output(&buf).Run(context.Background()))
	assert.Contains(t, buf.String(), "1.6.4")

	buf.Reset()
	Go().InvalidateCache()
	assert.NoError(t, Go().Command("version").Output(&buf).Run(context.Background()))
	assert.NotContains(t, buf.String(), "1.6.4")
}

func TestQueryMatching(t *testing.T) {
	tests := map[string]struct {
		Versions     []goVersionEntry
		MinorSearch  int
		VersionFound string
	}{
		"Version ending in 0": {
			Versions: []goVersionEntry{
				{
					Version: "go1.24.0",
					Stable:  true,
				},
			},
			MinorSearch:  24,
			VersionFound: "go1.24.0",
		},
		"Unstable version": {
			Versions: []goVersionEntry{
				{
					Version: "go1.24.0",
				},
			},
			MinorSearch: 24,
		},
	}

	for name, tc := range tests {
		_goVersionCache = nil
		t.Run(name, func(t *testing.T) {
			found, err := queryLatestPatch(func() ([]goVersionEntry, error) {
				return tc.Versions, nil
			}, 24)
			assert.NoError(t, err)
			assert.Equal(t, tc.VersionFound, found)
		})
	}
}
