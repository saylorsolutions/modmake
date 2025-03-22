package modmake_test

import (
	"context"
	"errors"
	mm "github.com/saylorsolutions/modmake"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetLogger(t *testing.T) {
	ctx := context.Background()
	log := mm.GetLogger(ctx)
	assert.NotNil(t, log, "Should produce a logger regardless of whether one exists in the context")

	ctx, log = mm.WithLogger(ctx, "name")
	assert.NotNil(t, ctx)
	assert.NotNil(t, log)

	log = mm.GetLogger(ctx)
	assert.NotNil(t, log)
}

func TestWithGroup(t *testing.T) {
	ctx := context.Background()
	ctx, log := mm.WithGroup(ctx, "group")

	scerr := testGetStepContextErr(log, errors.New("test error"))
	assert.Equal(t, "group", scerr.LogName, "WithGroup will set name to group name if not already set")
	assert.Equal(t, "group", scerr.LogGroup, "Group name is set")
	assert.Equal(t, "test error", scerr.Error())

	ctx, _ = mm.WithLogger(context.Background(), "name")
	ctx, log = mm.WithGroup(ctx, "group")

	scerr = testGetStepContextErr(log, errors.New("test error"))
	assert.Equal(t, "name", scerr.LogName, "WithGroup will use the name already set")
	assert.Equal(t, "group", scerr.LogGroup, "Group name is set")
	assert.Equal(t, "test error", scerr.Error())

	ctx, log = mm.WithGroup(ctx, "another-group")
	scerr = testGetStepContextErr(log, errors.New("test error"))
	assert.Equal(t, "name", scerr.LogName, "WithGroup will use the name already set")
	assert.Equal(t, "group/another-group", scerr.LogGroup, "Another group name is set")
	assert.Equal(t, "test error", scerr.Error())
}

func TestStepContextError_Unwrap(t *testing.T) {
	var someError = errors.New("some error")
	_, log := mm.WithLogger(context.Background(), "test")
	err := log.WrapErr(someError)
	assert.True(t, errors.Is(err, someError))
}

func testGetStepContextErr(log mm.Logger, err error) *mm.StepContextError {
	err = log.WrapErr(err)
	var scerr = new(mm.StepContextError)
	errors.As(err, &scerr)
	return scerr
}
