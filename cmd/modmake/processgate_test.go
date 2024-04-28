package main

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestProcessGate_Start(t *testing.T) {
	var (
		executed int
	)
	gate := newProcessGate(context.Background(), 100*time.Millisecond, func(ctx context.Context) error {
		executed++
		return nil
	})
	assert.Equal(t, 0, executed)
	start := func() {
		_ = gate.Start()
	}
	go start()
	go start()
	go start()
	assert.Equal(t, 0, executed)
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 1, executed)
	go start()
	go start()
	go start()
	time.Sleep(200 * time.Millisecond)
	assert.Equal(t, 2, executed)
	assert.NoError(t, gate.Stop())
}

func TestProcessGate_SetExitWait(t *testing.T) {
	var (
		executed int
	)
	gate := newProcessGate(context.Background(), 0, func(ctx context.Context) error {
		executed++
		<-ctx.Done()
		time.Sleep(time.Second)
		return nil
	})
	gate.SetExitWait(100 * time.Millisecond)
	assert.NoError(t, gate.Start())
	assert.NoError(t, gate.Start())
	assert.NoError(t, gate.Start())
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, executed)
	assert.ErrorIs(t, gate.Stop(), ErrLocked)
}

func TestProcessGate_Stop(t *testing.T) {
	var (
		executed int
	)
	gate := newProcessGate(context.Background(), 100*time.Millisecond, func(ctx context.Context) error {
		executed++
		return errors.New("test error")
	})

	assert.NoError(t, gate.Start())
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, executed)
	err := gate.Start()
	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrLocked, "The next call to start should report the error returned from the previous iteration")
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, executed, "The task should not have run with a previous run returning an error")
	assert.ErrorIs(t, gate.Stop(), ErrLocked, "The call to stop should report the gate is locked, due to the previous error")
}
