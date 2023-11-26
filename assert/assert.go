package assert

import (
	"fmt"
	"regexp"
	"strings"
)

// NotEmpty trims a string and panics if its len is 0.
func NotEmpty(s *string) {
	if s == nil {
		panic("nil pointer")
	}
	_s := strings.TrimSpace(*s)
	if len(_s) == 0 {
		panic(fmt.Sprintf("empty string: '%s'", *s))
	}
	*s = _s
}

var semverPattern = regexp.MustCompile(`^(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)

// SemverVersion will trim the string and panic if it doesn't match the spec for a semantic version.
// Uses the suggested regex from https://semver.org/#is-there-a-suggested-regular-expression-regex-to-check-a-semver-string
func SemverVersion(v *string) {
	NotEmpty(v)
	if !semverPattern.MatchString(*v) {
		panic(fmt.Sprintf("version '%s' is not a valid semantic version number", *v))
	}
}
