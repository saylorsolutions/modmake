package modmake

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
	"time"
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
