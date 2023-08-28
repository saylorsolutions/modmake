package modmake

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestContextAware(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Intentionally cancelling early to test execution

	fn := ContextAware(Error("should not have executed"))

	for i := 0; i < 100; i++ {
		assert.ErrorIs(t, fn.Run(ctx), context.Canceled)
	}
}

func TestError(t *testing.T) {
	r := IfExists("step.go", Error("File '%s' should not exist", "step.go"))
	err := r.Run(context.Background())
	assert.Equal(t, "File 'step.go' should not exist", err.Error())
}
