package modmake

import (
	"context"
	"fmt"
)

// Task is a convenient way to make a function that satisfies the Runner interface, and allows for more flexible invocation options.
type Task func(ctx context.Context) error

// WithoutErr is a convenience function that allows passing a function that should never return an error and translating it to a Task.
// The returned Task will recover panics by returning them as errors.
func WithoutErr(fn func(context.Context)) Task {
	return func(ctx context.Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%v", r)
			}
		}()
		fn(ctx)
		return nil
	}
}

// WithoutContext is a convenience function that handles the inbound context.Context in cases where it isn't needed.
// If the context is cancelled when this Task executes, then the context's error will be returned.
func WithoutContext(fn func() error) Task {
	return func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return fn()
		}
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
			fn()
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
	return t(ctx)
}

// Then returns a Task that runs if this Task executed successfully.
func (t Task) Then(other Runner) Task {
	return func(ctx context.Context) error {
		if err := t(ctx); err != nil {
			return err
		}
		return other.Run(ctx)
	}
}

// Catch runs the catch function if this Task returns an error.
func (t Task) Catch(catch func(error) error) Task {
	return func(ctx context.Context) error {
		if err := t(ctx); err != nil {
			return catch(err)
		}
		return nil
	}
}
