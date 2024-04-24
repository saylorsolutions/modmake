package modmake

import (
	"context"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

func TestTask_Then(t *testing.T) {
	var (
		a bool
		b bool
	)
	task := Plain(func() {
		a = true
	}).Then(
		Plain(func() {
			b = true
		}),
	)
	assert.NoError(t, task.Run(context.Background()))
	assert.True(t, a)
	assert.True(t, b)
}

func TestTask_Catch(t *testing.T) {
	var (
		handled    bool
		postAction bool
	)
	task := Error("An error occurred!").Catch(
		func(err error) Task {
			log.Println(err)
			handled = true
			return NoOp()
		},
	).Then(
		Plain(func() {
			postAction = true
		}),
	)
	assert.NoError(t, task.Run(context.Background()))
	assert.True(t, handled)
	assert.True(t, postAction)
}

func TestWithoutErr(t *testing.T) {
	var (
		called int
	)
	err := WithoutErr(func(ctx context.Context) {
		called++
		panic("error")
	})(context.Background())
	assert.Equal(t, "error", err.Error())
	assert.Equal(t, 1, called)

	assert.NoError(t, WithoutErr(func(ctx context.Context) {
		called++
	})(context.Background()))
	assert.Equal(t, 2, called)
}

func TestWithoutContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	fn := WithoutContext(func() error {
		return nil
	})
	err := fn.Run(ctx)
	assert.NoError(t, err)

	cancel()
	err = fn.Run(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestTask_Debounce(t *testing.T) {
	var (
		ctx      = context.Background()
		executed int
	)
	task := Plain(func() { executed++ }).Debounce(100 * time.Millisecond)
	assert.NoError(t, task(ctx))
	assert.NoError(t, task(ctx))
	assert.NoError(t, task(ctx))
	assert.Equal(t, 1, executed)
	assert.NoError(t, task(ctx))
	assert.Equal(t, 1, executed)
	time.Sleep(50 * time.Millisecond)
	assert.NoError(t, task(ctx))
	assert.Equal(t, 1, executed)
	time.Sleep(100 * time.Millisecond)
	assert.NoError(t, task(ctx))
	assert.Equal(t, 2, executed)
}
