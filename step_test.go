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

func TestNoColonInStepName(t *testing.T) {
	assert.Panics(t, func() {
		NewStep("test:step", "This should not be allowed")
	}, "Colon characters should not be allowed in a base step name")
}

func TestStepLifecycle(t *testing.T) {
	step := NewStep("example", "a basis Step to show the lifecycle")
	step.DependsOnRunner("dependency", "", Print("Ran a dependency"))
	step.BeforeRun(Print("Running before hook"))
	step.Does(Print("Running the 'example' step"))
	step.AfterRun(Print("Running after hook"))
	assert.NoError(t, step.Run(context.Background()))
}
