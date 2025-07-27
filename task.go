package modmake

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
)

// Task is a convenient way to make a function that satisfies the Runner interface, and allows for more flexible invocation options.
type Task func(ctx context.Context) error

// NoOp is a Task placeholder that immediately returns nil.
func NoOp() Task {
	return nil
}

// WithoutErr is a convenience function that allows passing a function that should never return an error and translating it to a Task.
// The returned Task will recover panics by returning them as errors.
func WithoutErr(fn func(context.Context)) Task {
	return func(ctx context.Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%v", r)
			}
		}()
		if fn != nil {
			fn(ctx)
		}
		return nil
	}
}

// WithoutContext is a convenience function that handles the inbound context.Context in cases where it isn't needed.
// If the context is cancelled when this Task executes, then the context's error will be returned.
func WithoutContext(fn func() error) Task {
	return func(ctx context.Context) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return fn()
	}
}

// ContextAware creates a Task that wraps the parameter with context handling logic.
// In the event that the context is done, the context's error is returned.
// This should not be used if custom [context.Context] handling is desired.
func ContextAware(r Runner) Task {
	return func(ctx context.Context) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		return r.Run(ctx)
	}
}

// Plain is a convenience function that translates a no-argument, no-return function into a Task, combining the logic of WithoutContext and WithoutErr.
func Plain(fn func()) Task {
	return func(ctx context.Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%v", r)
			}
		}()
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if fn != nil {
				fn()
			}
			return nil
		}
	}
}

// Error will create a Task returning an error, creating it by passing msg and args to [fmt.Errorf].
func Error(msg string, args ...any) Task {
	return func(ctx context.Context) error {
		return fmt.Errorf(msg, args...)
	}
}

func (t Task) Run(ctx context.Context) error {
	if t != nil {
		return t(ctx)
	}
	return nil
}

// Then returns a Task that runs if this Task executed successfully.
func (t Task) Then(other Runner) Task {
	return func(ctx context.Context) error {
		if err := t.Run(ctx); err != nil {
			return err
		}
		return other.Run(ctx)
	}
}

// Catch runs the catch function if this Task returns an error.
func (t Task) Catch(catch func(error) Task) Task {
	return func(ctx context.Context) error {
		if err := t.Run(ctx); err != nil {
			return catch(err).Run(ctx)
		}
		return nil
	}
}

// Finally can be used to run a function after a [Task] executes, regardless whether it was successful.
//
// The given function will receive the error returned from the base [Task].
// If the given function returns a non-nil error, it will be returned from the produced function.
// Otherwise, the error from the underlying [Task] will be returned.
func (t Task) Finally(finally func(err error) error) Task {
	return func(ctx context.Context) (terr error) {
		defer func() {
			if err := finally(terr); err != nil {
				terr = err
			}
		}()
		return t.Run(ctx)
	}
}

// Debounce will ensure that the returned [Task] can only be executed once per interval.
func (t Task) Debounce(interval time.Duration) Task {
	if interval <= time.Duration(0) {
		panic(fmt.Sprintf("invalid debounce interval: %d", int64(interval)))
	}

	var bouncing atomic.Bool
	return func(ctx context.Context) error {
		ready := bouncing.CompareAndSwap(false, true)
		if !ready {
			return nil
		}
		time.AfterFunc(interval, func() {
			bouncing.CompareAndSwap(true, false)
		})
		return t.Run(ctx)
	}
}

// Once ensures that the returned [Task] can only be executed at most once.
func (t Task) Once() Task {
	hasRun := atomic.Bool{}
	return func(ctx context.Context) error {
		if hasRun.CompareAndSwap(false, true) {
			return t.Run(ctx)
		}
		return nil
	}
}

// LogGroup sets a logging group for the [Task], and attaches the logging context if an error is returned.
// See [WithGroup] and [Logger.WrapErr] for details.
func (t Task) LogGroup(group string) Task {
	return func(ctx context.Context) error {
		ctx, log := WithGroup(ctx, group)
		if err := t.Run(ctx); err != nil {
			return log.WrapErr(err)
		}
		return nil
	}
}
