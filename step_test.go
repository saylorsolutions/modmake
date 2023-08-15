package modmake

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestContextAware(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Intentionally cancelling early to test execution
	runs := 0

	var fn Runner = RunnerFunc(func(ctx context.Context) error {
		runs++
		return errors.New("should not have executed")
	})
	fn = ContextAware(fn)

	for i := 0; i < 100; i++ {
		assert.ErrorIs(t, fn.Run(ctx), context.Canceled)
	}
	assert.Equal(t, 0, runs)
}
