package modmake

import (
	"context"
	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err)
	assert.True(t, buf.Len() > 0, "The Go version should have been written to buffer")
}
