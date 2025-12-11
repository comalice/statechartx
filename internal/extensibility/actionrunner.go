package extensibility

import (
	"fmt"
	"log"
	"time"

	"github.com/comalice/statechartx/internal/core"
	"github.com/comalice/statechartx/internal/primitives"
)

// DefaultActionRunner provides the default implementation of ActionRunner.
type DefaultActionRunner struct{}

// Run executes the given action reference.
func (r *DefaultActionRunner) Run(ctx *primitives.Context, action primitives.ActionRef, event primitives.Event) error {
	switch a := action.(type) {
	case nil:
		return nil
	case func(*primitives.Context, primitives.Event):
		a(ctx, event)
		return nil
	case string:
		return fmt.Errorf("action ID '%s' not registered", a)
	default:
		return fmt.Errorf("unknown action type: %T", action)
	}
}

// LoggingActionRunner wraps an ActionRunner and adds logging around execution.
type LoggingActionRunner struct {
	inner core.ActionRunner
}

// NewLoggingActionRunner creates a new LoggingActionRunner wrapping the given inner runner.
func NewLoggingActionRunner(inner core.ActionRunner) *LoggingActionRunner {
	return &LoggingActionRunner{inner: inner}
}

// Run logs before and after delegating to the inner runner.
func (r *LoggingActionRunner) Run(ctx *primitives.Context, action primitives.ActionRef, event primitives.Event) error {
	log.Printf("LOG: Executing action %v for event %q", action, event.Type)
	start := time.Now()
	err := r.inner.Run(ctx, action, event)
	log.Printf("LOG: Action %v completed in %v: %v", action, time.Since(start), err)
	return err
}
