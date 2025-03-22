package modmake

import (
	"context"
	"errors"
	"fmt"
	"github.com/fatih/color"
	"io"
	"log"
	"os"
	"strings"
)

func init() {
	log.SetOutput(os.Stderr)
}

var (
	_stepDebugLog bool
	errColor      = color.New(color.FgRed, color.Bold).SprintFunc()
	okColor       = color.New(color.FgGreen, color.Bold).SprintFunc()
	warnColor     = color.New(color.FgYellow, color.Bold).SprintFunc()
	debugColor    = color.New(color.FgCyan).SprintFunc()
)

// SetLogOutput defines where logs should be written.
// By default, logs are written to stderr.
func SetLogOutput(w io.Writer) {
	log.SetOutput(w)
}

type Logger interface {
	// Info writes an informational log message to output.
	Info(msg string, args ...any)
	// Warn writes a warning message to output, and should be used sparingly.
	Warn(msg string, args ...any)
	// Error writes an error message to output, and should indicate a failure to perform a requested action.
	Error(msg string, args ...any)
	// Debug writes an informational log message to output that is only useful for debugging build logic.
	Debug(msg string, args ...any)
	// WrapErr wraps the given error with log context.
	// This is only done if the error is not already a wrapped error.
	WrapErr(err error) error
}

type stepLogger struct {
	name   string
	groups []string
}

func (l *stepLogger) Info(msg string, args ...any) {
	msg = strings.TrimSuffix(msg, "\n")
	log.Printf("[%s%s] %s\n", okColor(l.name), okColor(l.groupOutput()), fmt.Sprintf(msg, args...))
}

func (l *stepLogger) Warn(msg string, args ...any) {
	msg = strings.TrimSuffix(msg, "\n")
	log.Printf("[%s%s] %s %s\n", warnColor(l.name), warnColor(l.groupOutput()), warnColor("WARN"), fmt.Sprintf(msg, args...))
}

func (l *stepLogger) Error(msg string, args ...any) {
	msg = strings.TrimSuffix(msg, "\n")
	log.Printf("[%s%s] %s %s\n", errColor(l.name), errColor(l.groupOutput()), errColor("ERROR"), fmt.Sprintf(msg, args...))
}

func (l *stepLogger) Debug(msg string, args ...any) {
	msg = strings.TrimSuffix(msg, "\n")
	if _stepDebugLog {
		log.Printf("[%s%s] %s %s\n", debugColor(l.name), debugColor(l.groupOutput()), debugColor("DEBUG"), fmt.Sprintf(msg, args...))
	}
}

func (l *stepLogger) WrapErr(err error) error {
	if err == nil {
		return nil
	}
	var ctxErr = new(StepContextError)
	if errors.As(err, &ctxErr) {
		return err
	}
	return &StepContextError{
		inner:    err,
		LogName:  l.name,
		LogGroup: strings.Join(l.groups, "/"),
	}
}

func (l *stepLogger) groupOutput() string {
	if len(l.groups) == 0 {
		return ""
	}
	return fmt.Sprintf(" (%s)", strings.Join(l.groups, "/"))
}

type loggerKeyType string

const (
	loggerKey = loggerKeyType("logger")
)

// WithLogger creates a new [Logger], sets the given name, and sets the Logger value in the given context.
func WithLogger(ctx context.Context, name string) (context.Context, Logger) {
	logger := &stepLogger{
		name: name,
	}
	return context.WithValue(ctx, loggerKey, logger), logger
}

// WithGroup appends a group to the context's [Logger] as a way to indicate a hierarchy of logging contexts.
func WithGroup(ctx context.Context, group string) (context.Context, Logger) {
	logger, ok := ctx.Value(loggerKey).(*stepLogger)
	if !ok {
		logger = &stepLogger{name: group, groups: []string{group}}
		ctx = context.WithValue(ctx, loggerKey, logger)
		return ctx, logger
	}
	newLogger := &stepLogger{
		name:   logger.name,
		groups: append(logger.groups, group),
	}
	ctx = context.WithValue(ctx, loggerKey, newLogger)
	return ctx, newLogger
}

// GetLogger gets the [Logger] from the given context.
// Returns false if there is no [Logger] available.
func GetLogger(ctx context.Context) Logger {
	val, ok := ctx.Value(loggerKey).(Logger)
	if !ok {
		return new(stepLogger)
	}
	return val
}

// SetStepDebug sets the global policy on debug step logs.
func SetStepDebug(printDebug bool) {
	_stepDebugLog = printDebug
}

type StepContextError struct {
	inner    error
	LogName  string
	LogGroup string
}

func (err *StepContextError) Error() string {
	return err.inner.Error()
}

func (err *StepContextError) Unwrap() error {
	return err.inner
}
