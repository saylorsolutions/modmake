package assert

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSemverVersion(t *testing.T) {
	tests := map[string]string{
		"Plain with leading space":           " \t\r\n1.0.0",
		"Plain with trailing space":          "1.0.0\t\r\n ",
		"Just version number":                "1.0.0",
		"Pre-release version":                "1.0.0-alpha",
		"Numbered pre-release version":       "1.0.0-alpha.1",
		"Pre-release x-y-z":                  "1.0.0-alpha.1-x-y-z",
		"Build info after patch":             "1.0.0+sha1.deadbeef",
		"Build info after pre-release":       "1.0.0-alpha.1+sha1.deadbeef",
		"All the build hyphens":              "1.0.0-alpha.1+-----sha1.dead-----beef-----",
		"Short build info after patch":       "1.0.0+deadbeef",
		"Short build info after pre-release": "1.0.0-alpha.1+deadbeef",
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			assert.NotPanics(t, func() {
				SemverVersion(&tc)
			})
		})
	}
}

func TestSemverVersionNeg(t *testing.T) {
	tests := map[string]string{
		"Missing minor":  "1",
		"Missing patch":  "1.0",
		"Letter version": "a.b.c",
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			assert.Panics(t, func() {
				SemverVersion(&tc)
			})
		})
	}
}

func TestNotEmpty(t *testing.T) {
	tests := map[string]struct {
		input  string
		output string
		panics bool
	}{
		"All spaces": {
			input:  " \t\r\n ",
			panics: true,
		},
		"Leading spaces": {
			input:  "\t\r\n a",
			output: "a",
		},
		"Trailing spaces": {
			input:  "a\t\r\n ",
			output: "a",
		},
		"Surrounding spaces": {
			input:  "\t\r\n a\t\r\n ",
			output: "a",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			if tc.panics {
				assert.Panics(t, func() {
					NotEmpty(&tc.input)
				})
			} else {
				assert.NotPanics(t, func() {
					NotEmpty(&tc.input)
				})
				assert.Equal(t, tc.output, tc.input)
			}
		})
	}
}

func TestNotEmptyNil(t *testing.T) {
	assert.Panics(t, func() {
		NotEmpty(nil)
	})
}
