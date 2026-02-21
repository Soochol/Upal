package agents

import (
	"fmt"
	"strings"

	"github.com/expr-lang/expr"
	"google.golang.org/adk/session"
)

// evaluateCondition evaluates a condition expression against session state.
// The expression can reference node outputs as variables.
// Example: "sentiment == 'positive'" where sentiment comes from session state.
// Template references like {{node_id}} are resolved before evaluation.
// Returns true if the expression evaluates to a truthy value.
func evaluateCondition(expression string, state session.State) (bool, error) {
	if expression == "" {
		return true, nil
	}

	// Resolve {{key}} templates to actual values first.
	resolved := resolveTemplateFromState(expression, state)

	// Build environment from session state for expr evaluation.
	env := make(map[string]any)
	for k, v := range state.All() {
		if !strings.HasPrefix(k, "__") {
			env[k] = v
		}
	}

	program, err := expr.Compile(resolved, expr.Env(env))
	if err != nil {
		// If compilation fails (e.g. undefined variables), treat as false.
		return false, fmt.Errorf("compile condition %q: %w", expression, err)
	}

	result, err := expr.Run(program, env)
	if err != nil {
		return false, fmt.Errorf("evaluate condition %q: %w", expression, err)
	}

	return isTruthy(result), nil
}

// isTruthy converts a value to a boolean.
func isTruthy(v any) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val != ""
	case int:
		return val != 0
	case int64:
		return val != 0
	case float64:
		return val != 0
	default:
		return true
	}
}
