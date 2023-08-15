package modmake

import (
	"context"
	"github.com/bitfield/script"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestExecScript(t *testing.T) {
	err := Exec(
		func(_ context.Context) error {
			_, err := script.File("script.go").WriteFile("script.go.copy")
			return err
		},
		func(_ context.Context) error {
			_, err := script.File("script.go.copy").Stdout()
			return err
		},
		func(_ context.Context) error {
			return os.Remove("script.go.copy")
		},
	).Run(context.Background())
	assert.NoError(t, err)
}

func TestSkipIfExists(t *testing.T) {
	var executed bool
	err := SkipIfExists("script.go", RunnerFunc(func(ctx context.Context) error {
		executed = true
		return nil
	})).Run(context.Background())
	assert.NoError(t, err)
	assert.False(t, executed)
}
