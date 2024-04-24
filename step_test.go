package modmake

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
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

func TestStep_ResetState(t *testing.T) {
	var (
		aExecuted, bExecuted int
		ctx                  = context.Background()
	)
	stepA := NewStep("step-a", "").Does(Plain(func() {
		aExecuted++
	}))
	stepB := NewStep("step-b", "").Does(Plain(func() {
		bExecuted++
	}))
	stepA.DependsOn(stepB)
	assert.NoError(t, stepA.Run(ctx))
	assert.Equal(t, 1, aExecuted)
	assert.Equal(t, 1, bExecuted)
	assert.NoError(t, stepA.Run(ctx))
	assert.Equal(t, 1, aExecuted)
	assert.Equal(t, 1, bExecuted)
	assert.NoError(t, stepB.Run(ctx))
	assert.Equal(t, 1, bExecuted)
	stepA.ResetState()
	assert.NoError(t, stepA.Run(ctx))
	assert.Equal(t, 2, aExecuted)
	assert.Equal(t, 2, bExecuted)
	assert.NoError(t, stepB.Run(ctx))
	assert.Equal(t, 2, bExecuted)
}

func TestStep_Debounce(t *testing.T) {
	var (
		executed int
		stopped  int
		ctx      = context.Background()
	)
	stepA := NewStep("step-a", "").Does(WithoutErr(func(ctx context.Context) {
		select {
		case <-ctx.Done():
			t.Log("stopped")
			stopped++
		case <-time.After(200 * time.Millisecond):
			t.Log("executed")
			executed++
		}
	}))

	debounced := stepA.Debounce(100 * time.Millisecond)
	sync := func() { _ = debounced.Run(ctx) }
	async := func() {
		go func() {
			_ = debounced.Run(ctx)
		}()
	}

	assert.Equal(t, 0, executed)
	assert.Equal(t, 0, stopped)
	async()
	async()
	async()
	time.Sleep(250 * time.Millisecond)
	assert.Equal(t, 1, executed)
	assert.Equal(t, 0, stopped)

	async()
	async()
	async()
	time.Sleep(150 * time.Millisecond)
	sync()
	assert.Equal(t, 2, executed)
	assert.Equal(t, 1, stopped)
}
