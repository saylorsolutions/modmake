package modmake

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestF(t *testing.T) {
	tests := map[string]struct {
		in   string
		out  string
		data EnvMap
	}{
		"Case insensitive reference": {
			in:  "A string with a name of ${name}",
			out: "A string with a name of Bob",
			data: EnvMap{
				"NAME": "Bob",
			},
		},
		"Case insensitive data key": {
			in:  "A string with a name of ${NAME}",
			out: "A string with a name of Bob",
			data: EnvMap{
				"name": "Bob",
			},
		},
		"Nil data": {
			in:   "Nothing here ${here}",
			out:  "Nothing here ",
			data: nil,
		},
		"Variable with no value": {
			in:  "Nothing here ${here}",
			out: "Nothing here ",
			data: EnvMap{
				"here": "",
			},
		},
		"Dollars aren't enough to trigger interpolation": {
			in:  "Here's a $VALUE",
			out: "Here's a $VALUE",
			data: EnvMap{
				"VALUE": "Should not be substituted",
			},
		},
		"Financial figures aren't interpolation": {
			in:  "Here's $10.00",
			out: "Here's $10.00",
			data: EnvMap{
				"10.00": "11.00",
				"10":    "11",
			},
		},
		"Braces aren't enough to trigger interpolation": {
			in:  "Here's a $ {VALUE}",
			out: "Here's a $ {VALUE}",
			data: EnvMap{
				"VALUE": "Should not be substituted",
			},
		},
		"Variable names can have space padding": {
			in:  "A string with a name of ${ \tname \t\n}",
			out: "A string with a name of Bob",
			data: EnvMap{
				"NAME": "Bob",
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			result := F(tc.in, tc.data)
			assert.Equal(t, tc.out, result)
			result = string(FReader(strings.NewReader(tc.in), tc.data))
			assert.Equal(t, tc.out, result)
		})
	}
}

func TestF_DynamicVariables(t *testing.T) {
	const (
		nonExistentKey = "SOME_KEY_THAT_SHOULD_NOT_BE_A_THING"
		value          = "some value"
	)
	oldEnv := Environment()
	assert.NoError(t, os.Setenv(nonExistentKey, value))
	newEnv := Environment()
	assert.Equal(t, "", oldEnv[nonExistentKey])
	assert.NotEqual(t, oldEnv[nonExistentKey], newEnv[nonExistentKey])
	assert.Equal(t, value, newEnv[nonExistentKey])
}
