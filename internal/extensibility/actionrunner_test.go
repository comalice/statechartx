package extensibility

import (
	"testing"

	"github.com/comalice/statechartx/internal/primitives"
)

func TestDefaultActionRunner_Run_Func(t *testing.T) {
	ctx := primitives.NewContext()
	event := primitives.NewEvent("test", nil)
	called := false
	action := func(c *primitives.Context, e primitives.Event) {
		called = true
	}
	r := &DefaultActionRunner{}
	err := r.Run(ctx, action, event)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("action func not called")
	}
}

func TestDefaultActionRunner_Run_String(t *testing.T) {
	r := &DefaultActionRunner{}
	err := r.Run(primitives.NewContext(), "unknown", primitives.NewEvent("test", nil))
	if err == nil {
		t.Error("expected error for string action")
	}
	expected := `action ID 'unknown' not registered`
	if err.Error() != expected {
		t.Errorf("wrong error: %v", err)
	}
}

func TestDefaultActionRunner_Run_Nil(t *testing.T) {
	r := &DefaultActionRunner{}
	err := r.Run(primitives.NewContext(), nil, primitives.NewEvent("test", nil))
	if err != nil {
		t.Errorf("unexpected error for nil: %v", err)
	}
}

func TestLoggingActionRunner(t *testing.T) {
	ctx := primitives.NewContext()
	event := primitives.NewEvent("test", nil)
	called := false
	action := func(c *primitives.Context, e primitives.Event) {
		called = true
	}
	inner := &DefaultActionRunner{}
	r := NewLoggingActionRunner(inner)
	err := r.Run(ctx, action, event)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("inner action not called")
	}
}
