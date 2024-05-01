package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/saylorsolutions/modmake"
	"log"
	"os/exec"
	"sync/atomic"
	"time"
)

type gateState = int32

const (
	ready           gateState = iota // ready indicates that the gate is able to start immediately.
	starting                         // started is a transition state between ready and started.
	settingDebounce                  // settingDebounce is used to avoid concurrent access to debounce variables.
	started                          // started indicates that the gate is running.
	stopping                         // stopping is a transition state between started and stopped.
	stopped                          // stopped is a preparation state before Stop puts the gate back into ready.
	locked                           // locked indicates that an unrecoverable error was reported from the task.
)

var (
	ErrLocked = errors.New("process gate is locked")
)

type processGate struct {
	base             context.Context
	errCh            chan error
	task             modmake.Task
	state            atomic.Int32
	debounceInterval time.Duration
	cancel           context.CancelFunc
	subCtx           context.Context
	debounceAt       atomic.Int64
	errHandler       func(err error) error
	waitExit         atomic.Int64
}

func newProcessGate(ctx context.Context, debounceInterval time.Duration, task modmake.Task) *processGate {
	return &processGate{
		base:             ctx,
		errCh:            make(chan error, 1),
		cancel:           func() {},
		debounceInterval: debounceInterval,
		task:             task,
		errHandler: func(err error) error {
			exitErr := &exec.ExitError{}
			if errors.As(err, &exitErr) {
				log.Println("Process exited with status:", exitErr.ExitCode())
				return nil
			}
			return err
		},
	}
}

func (g *processGate) SetExitWait(wait time.Duration) {
	g.waitExit.Store(int64(wait))
}

func (g *processGate) Start() error {
	if g.state.Load() == locked {
		return ErrLocked
	}
	if g.getDebounce().After(time.Now()) {
		return nil
	}
	if err := g.Stop(); err != nil {
		return err
	}
	if !g.state.CompareAndSwap(ready, starting) {
		return nil
	}

	g.subCtx, g.cancel = context.WithCancel(g.base)
	go func() {
		g.setDebounce()
		log.Println(">>>>> Starting task")
		g.errCh <- g.task(g.subCtx)
		log.Println(">>>>> Task stopped")
	}()
	return nil
}

func (g *processGate) getDebounce() time.Time {
	return time.UnixMilli(g.debounceAt.Load())
}

func (g *processGate) setDebounce() {
	if !g.state.CompareAndSwap(starting, settingDebounce) {
		return
	}
	old := g.debounceAt.Load()
	_ = g.debounceAt.CompareAndSwap(old, time.Now().Add(g.debounceInterval).UnixMilli())
	g.state.Store(started)
}

func (g *processGate) Stop() error {
	if g.state.Load() == locked {
		return ErrLocked
	}
	if g.state.CompareAndSwap(started, stopping) {
		if err := g.stop(); err != nil {
			g.state.Store(locked)
			return err
		}
		if !g.state.CompareAndSwap(stopped, ready) {
			// Something else already started again.
			return nil
		}
	}
	return nil
}

func (g *processGate) stop() error {
	if !g.state.CompareAndSwap(stopping, stopped) {
		return nil
	}
	var waitCh <-chan time.Time
	wait := g.waitExit.Load()
	if wait > 0 {
		waitCh = time.After(time.Duration(wait))
	}
	g.cancel()
	select {
	case <-g.base.Done():
		return g.base.Err()
	case err, more := <-g.errCh:
		if !more {
			return context.Canceled
		}
		if err != nil {
			if g.errHandler != nil {
				return g.errHandler(err)
			}
			return err
		}
		return nil
	case <-waitCh:
		g.state.Store(locked)
		return fmt.Errorf("%w: exit wait time '%s' elapsed", ErrLocked, time.Duration(wait).String())
	}
}
