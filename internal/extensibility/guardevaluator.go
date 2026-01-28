package extensibility

import (
	"strconv"
	"strings"

	"github.com/comalice/statechartx/internal/primitives"
)

// DefaultGuardEvaluator provides the default implementation of GuardEvaluator.
type DefaultGuardEvaluator struct{}

// Eval evaluates a guard condition.
func (e *DefaultGuardEvaluator) Eval(ctx *primitives.Context, guard primitives.GuardRef, event primitives.Event) bool {
	if guard == nil {
		return true
	}
	switch g := guard.(type) {
	case func(*primitives.Context, primitives.Event) bool:
		return g(ctx, event)
	case string:
		return false // unregistered guards fail closed
	default:
		return false
	}
}

// ExpressionGuardEvaluator evaluates simple string expressions like "temp > 30" or "loggedIn == true".
type ExpressionGuardEvaluator struct{}

// NewExpressionGuardEvaluator creates a new ExpressionGuardEvaluator.
func NewExpressionGuardEvaluator() *ExpressionGuardEvaluator {
	return &ExpressionGuardEvaluator{}
}

// Eval parses and evaluates simple expressions against the context.
func (e *ExpressionGuardEvaluator) Eval(ctx *primitives.Context, guard primitives.GuardRef, event primitives.Event) bool {
	if guard == nil {
		return true
	}
	str, ok := guard.(string)
	if !ok {
		return false
	}
	// Parse "key op value"
	parts := strings.Fields(str)
	if len(parts) != 3 {
		return false
	}
	key, op, valStr := parts[0], parts[1], parts[2]

	v, hasKey := ctx.Get(key)
	if !hasKey {
		return false
	}

	switch op {
	case "==":
		switch valStr {
		case "true":
			return v == true
		case "false":
			return v == false
		case "nil":
			return v == nil
		default:
			fVal, err := strconv.ParseFloat(valStr, 64)
			if err == nil {
				if f, ok := v.(float64); ok {
					return f == fVal
				}
			}
			if s, ok := v.(string); ok {
				return s == valStr
			}
			return false
		}
	case ">":
		fVal, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return false
		}
		if f, ok := v.(float64); ok {
			return f > fVal
		}
		return false
	case "<":
		fVal, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			return false
		}
		if f, ok := v.(float64); ok {
			return f < fVal
		}
		return false
	case "!=":
		// reuse == logic but negate
		return !e.Eval(ctx, primitives.GuardRef(key+" == "+valStr), event)
	default:
		return false
	}
}
