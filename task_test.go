package modmake

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, task.Run(context.Background()))
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
	require.NoError(t, task.Run(context.Background()))
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

	require.NoError(t, WithoutErr(func(ctx context.Context) {
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
	require.NoError(t, err)

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
	require.NoError(t, task(ctx))
	require.NoError(t, task(ctx))
	require.NoError(t, task(ctx))
	assert.Equal(t, 1, executed)
	require.NoError(t, task(ctx))
	assert.Equal(t, 1, executed)
	time.Sleep(50 * time.Millisecond)
	require.NoError(t, task(ctx))
	assert.Equal(t, 1, executed)
	time.Sleep(100 * time.Millisecond)
	require.NoError(t, task(ctx))
	assert.Equal(t, 2, executed)
}

func TestTask_Finally(t *testing.T) {
	var didFinally bool
	task := Plain(func() {
		assert.False(t, didFinally, "'didFinally' should not have been set yet")
	}).
		Finally(func(_ error) error {
			didFinally = true
			return nil
		}).
		Finally(func(_ error) error {
			assert.True(t, didFinally, "Should have done finally")
			return nil
		})
	require.NoError(t, task.Run(context.Background()))
}
