package modmake

import (
	"context"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
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
		func(err error) error {
			log.Println(err)
			handled = true
			return nil
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
