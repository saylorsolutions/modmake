package modmake

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ExampleCommand_Silent() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := Exec("go", "version").Silent().Run(ctx)
	if err != nil {
		panic(err)
	}
	// Output:
}

func TestCommand_Output(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var buf strings.Builder
	err := Exec("go", "version").Output(&buf).Run(ctx)
	t.Logf("Data written to buffer: %s", buf.String())
	require.NoError(t, err)
	assert.Positive(t, buf.Len(), "The Go version should have been written to buffer")
}

func TestCommand_String(t *testing.T) {
	tests := map[string]struct {
		cmd      *Command
		expected string
	}{
		"Single command": {
			cmd:      Exec("command"),
			expected: "command",
		},
		"Args": {
			cmd:      Exec("command", "arg1", "arg2"),
			expected: "command arg1 arg2",
		},
		"Leading and trailing": {
			cmd:      Exec("command").TrailingArg("arg3").LeadingArg("arg1").Arg("arg2"),
			expected: "command arg1 arg2 arg3",
		},
		"Arg with spaces": {
			cmd:      Exec("command", "arg with spaces"),
			expected: "command \"arg with spaces\"",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.cmd.String())
		})
	}
}
