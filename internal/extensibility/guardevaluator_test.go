package extensibility

import (
	"testing"

	"github.com/comalice/statechartx/internal/primitives"
)

func TestDefaultGuardEvaluator_Eval_Func(t *testing.T) {
	ctx := primitives.NewContext()
	event := primitives.NewEvent("test", nil)
	called := false
	guard := func(c *primitives.Context, e primitives.Event) bool {
		called = true
		return true
	}
	e := &DefaultGuardEvaluator{}
	result := e.Eval(ctx, guard, event)
	if !result {
		t.Error("func guard returned false")
	}
	if !called {
		t.Error("guard func not called")
	}
}

func TestDefaultGuardEvaluator_Eval_Nil(t *testing.T) {
	e := &DefaultGuardEvaluator{}
	result := e.Eval(primitives.NewContext(), nil, primitives.NewEvent("test", nil))
	if !result {
		t.Error("nil guard should be true")
	}
}

func TestDefaultGuardEvaluator_Eval_String(t *testing.T) {
	e := &DefaultGuardEvaluator{}
	result := e.Eval(primitives.NewContext(), "unknown", primitives.NewEvent("test", nil))
	if result {
		t.Error("string guard should be false")
	}
}

func TestExpressionGuardEvaluator_EqNumber(t *testing.T) {
	e := NewExpressionGuardEvaluator()
	ctx := primitives.NewContext()
	ctx.Set("temp", 30.0)
	event := primitives.NewEvent("test", nil)
	if !e.Eval(ctx, "temp == 30", event) {
		t.Error("30 == 30")
	}
	if e.Eval(ctx, "temp == 31", event) {
		t.Error("30 != 31")
	}
}

func TestExpressionGuardEvaluator_Gt(t *testing.T) {
	e := NewExpressionGuardEvaluator()
	ctx := primitives.NewContext()
	ctx.Set("temp", 35.0)
	event := primitives.NewEvent("test", nil)
	if !e.Eval(ctx, "temp > 30", event) {
		t.Error("35 > 30")
	}
}

func TestExpressionGuardEvaluator_Bool(t *testing.T) {
	e := NewExpressionGuardEvaluator()
	ctx := primitives.NewContext()
	ctx.Set("loggedIn", true)
	event := primitives.NewEvent("test", nil)
	if !e.Eval(ctx, "loggedIn == true", event) {
		t.Error("loggedIn == true")
	}
}

func TestExpressionGuardEvaluator_Neq(t *testing.T) {
	e := NewExpressionGuardEvaluator()
	ctx := primitives.NewContext()
	ctx.Set("user", "alice")
	event := primitives.NewEvent("test", nil)
	if !e.Eval(ctx, "user != bob", event) {
		t.Error("alice != bob")
	}
	if e.Eval(ctx, "user != alice", event) {
		t.Error("alice == alice")
	}
}

func TestExpressionGuardEvaluator_MissingKey(t *testing.T) {
	e := NewExpressionGuardEvaluator()
	ctx := primitives.NewContext()
	event := primitives.NewEvent("test", nil)
	if e.Eval(ctx, "missing == true", event) {
		t.Error("missing key should false")
	}
}
