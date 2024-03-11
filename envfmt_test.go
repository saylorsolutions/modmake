package modmake

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		"Variables can have default values": {
			in:   "A string with a name of ${ name : John }",
			out:  "A string with a name of John",
			data: nil,
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
	require.Equal(t, "", oldEnv[nonExistentKey])
	assert.NoError(t, os.Setenv(nonExistentKey, value))
	newEnv := Environment()
	assert.Equal(t, "", oldEnv[nonExistentKey])
	assert.NotEqual(t, oldEnv[nonExistentKey], newEnv[nonExistentKey])
	assert.Equal(t, value, newEnv[nonExistentKey])
}

func ExampleF() {
	fmt.Println(F("My string has a variable reference ${BUILD_NUM}?", EnvMap{
		"BUILD_NUM": "1",
	}))
	fmt.Println(F("My string has a variable reference ${:$}{BUILD_NUM}?"))
	fmt.Println(F("My string that references build ${BUILD_NUM}."))
	fmt.Println(F("My string that references build ${BUILD_NUM:0}."))

	// Output:
	// My string has a variable reference 1?
	// My string has a variable reference ${BUILD_NUM}?
	// My string that references build .
	// My string that references build 0.
}
