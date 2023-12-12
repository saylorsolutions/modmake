package modmake

import (
	"fmt"
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
)

// SetLogOutput defines where logs should be written.
// By default, logs are written to stderr.
func SetLogOutput(w io.Writer) {
	log.SetOutput(w)
}

// SetStepDebug sets the global policy on debug step logs.
func SetStepDebug(printDebug bool) {
	_stepDebugLog = printDebug
}

// Info emits informational log messages that are generally useful for user awareness.
func (s *Step) Info(msg string, args ...any) {
	msg = strings.TrimSuffix(msg, "\n")
	log.Printf("[%s] %s\n", okColor(s.name), fmt.Sprintf(msg, args...))
}

// Warn emits warning log messages that indicate something might not be right.
func (s *Step) Warn(msg string, args ...any) {
	msg = strings.TrimSuffix(msg, "\n")
	log.Printf("[%s] %s %s\n", warnColor(s.name), warnColor("WARN"), fmt.Sprintf(msg, args...))
}

// Error emits log messages about errors while performing a step.
func (s *Step) Error(msg string, args ...any) {
	msg = strings.TrimSuffix(msg, "\n")
	log.Printf("[%s] %s %s\n", errColor(s.name), errColor("ERROR"), fmt.Sprintf(msg, args...))
}

// Debug emits log messages that are really only useful for digging deep into how a build step is executing.
func (s *Step) Debug(msg string, args ...any) {
	msg = strings.TrimSuffix(msg, "\n")
	if _stepDebugLog {
		log.Printf("[%s] %s %s\n", debugColor(s.name), debugColor("DEBUG"), fmt.Sprintf(msg, args...))
	}
}
