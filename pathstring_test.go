package modmake

import (
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"testing"
)

func TestPath(t *testing.T) {
	tests := map[string]struct {
		path     string
		segments []string
		expected string
	}{
		"Just path": {
			path:     "cmd/runner",
			expected: "cmd/runner",
		},
		"Path and segments": {
			path:     "cmd",
			segments: []string{"runner", "build/dir"},
			expected: "cmd/runner/build/dir",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			var p PathString
			if len(tc.segments) == 0 {
				p = Path(tc.path)
			} else {
				p = Path(tc.path, tc.segments...)
			}
			assert.Equal(t, filepath.FromSlash(tc.expected), string(p))
		})
	}
}

func TestPathString_Join(t *testing.T) {
	tests := map[string]struct {
		base     PathString
		joining  []string
		expected PathString
	}{
		"No segments": {
			base:     Path("a/b"),
			expected: Path("a/b"),
		},
		"One segment": {
			base:     Path("a/b"),
			joining:  []string{"c"},
			expected: Path("a/b/c"),
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			orig := tc.base
			p := tc.base.Join(tc.joining...)
			assert.Equal(t, tc.expected, p)
			assert.Equal(t, orig, tc.base)
		})
	}
}
