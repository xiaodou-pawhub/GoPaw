// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package workflow

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Resolver resolves variables in workflow expressions.
type Resolver struct {
	variables map[string]interface{}
	steps     map[string]*StepOutput
	trigger   map[string]interface{}
}

// NewResolver creates a new variable resolver.
func NewResolver(variables, trigger map[string]interface{}, steps map[string]*StepOutput) *Resolver {
	return &Resolver{
		variables: variables,
		steps:     steps,
		trigger:   trigger,
	}
}

// Resolve resolves all variables in the given value.
// Supports:
// - {{var.name}} - workflow variables
// - {{steps.step_id.output.field}} - step outputs
// - {{trigger.field}} - trigger data
// - {{env.NAME}} - environment variables
// - {{now}} - current timestamp
// - {{uuid}} - unique identifier
// - {{date}} - current date
// - {{datetime}} - current datetime
func (r *Resolver) Resolve(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return r.resolveString(v)
	case map[string]interface{}:
		return r.resolveMap(v)
	case []interface{}:
		return r.resolveSlice(v)
	default:
		return value, nil
	}
}

// resolveString resolves variables in a string.
func (r *Resolver) resolveString(s string) (interface{}, error) {
	// Check if the entire string is a variable expression
	if strings.HasPrefix(s, "{{") && strings.HasSuffix(s, "}}") {
		expr := strings.TrimSpace(s[2 : len(s)-2])
		return r.resolveExpression(expr)
	}

	// Replace all variable expressions in the string
	re := regexp.MustCompile(`\{\{\s*([^}]+)\s*\}\}`)
	result := re.ReplaceAllStringFunc(s, func(match string) string {
		expr := strings.TrimSpace(match[2 : len(match)-2])
		val, err := r.resolveExpression(expr)
		if err != nil {
			return match // Keep original if resolution fails
		}
		return fmt.Sprintf("%v", val)
	})

	return result, nil
}

// resolveMap resolves variables in a map.
func (r *Resolver) resolveMap(m map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	for k, v := range m {
		resolved, err := r.Resolve(v)
		if err != nil {
			return nil, err
		}
		result[k] = resolved
	}
	return result, nil
}

// resolveSlice resolves variables in a slice.
func (r *Resolver) resolveSlice(s []interface{}) ([]interface{}, error) {
	result := make([]interface{}, len(s))
	for i, v := range s {
		resolved, err := r.Resolve(v)
		if err != nil {
			return nil, err
		}
		result[i] = resolved
	}
	return result, nil
}

// resolveExpression resolves a single expression.
func (r *Resolver) resolveExpression(expr string) (interface{}, error) {
	parts := strings.Split(expr, ".")
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty expression")
	}

	switch parts[0] {
	case "var":
		return r.resolveVariable(parts[1:])
	case "steps":
		return r.resolveStepOutput(parts[1:])
	case "trigger":
		return r.resolveTrigger(parts[1:])
	case "env":
		return r.resolveEnv(parts[1:])
	case "now":
		return time.Now().Unix(), nil
	case "uuid":
		return generateUUID(), nil
	case "date":
		return time.Now().Format("2006-01-02"), nil
	case "datetime":
		return time.Now().Format("2006-01-02 15:04:05"), nil
	default:
		return nil, fmt.Errorf("unknown namespace: %s", parts[0])
	}
}

// resolveVariable resolves a variable reference.
func (r *Resolver) resolveVariable(parts []string) (interface{}, error) {
	if len(parts) == 0 {
		return nil, fmt.Errorf("variable name required")
	}

	val, ok := r.variables[parts[0]]
	if !ok {
		return nil, fmt.Errorf("variable not found: %s", parts[0])
	}

	// Navigate nested fields
	return r.navigateValue(val, parts[1:])
}

// resolveStepOutput resolves a step output reference.
// Format: steps.step_id.output.field
func (r *Resolver) resolveStepOutput(parts []string) (interface{}, error) {
	if len(parts) < 2 {
		return nil, fmt.Errorf("step output reference must have at least 2 parts: step_id and output")
	}

	stepID := parts[0]
	if parts[1] != "output" {
		return nil, fmt.Errorf("expected 'output' after step_id, got: %s", parts[1])
	}

	step, ok := r.steps[stepID]
	if !ok {
		return nil, fmt.Errorf("step not found: %s", stepID)
	}

	if step.Status != StepStatusCompleted {
		return nil, fmt.Errorf("step %s is not completed", stepID)
	}

	// Get the output field
	if len(parts) == 2 {
		return step.Output, nil
	}

	fieldName := parts[2]
	val, ok := step.Output[fieldName]
	if !ok {
		return nil, fmt.Errorf("output field not found: %s", fieldName)
	}

	// Navigate nested fields
	return r.navigateValue(val, parts[3:])
}

// resolveTrigger resolves a trigger data reference.
func (r *Resolver) resolveTrigger(parts []string) (interface{}, error) {
	if len(parts) == 0 {
		return r.trigger, nil
	}

	val, ok := r.trigger[parts[0]]
	if !ok {
		return nil, fmt.Errorf("trigger field not found: %s", parts[0])
	}

	return r.navigateValue(val, parts[1:])
}

// resolveEnv resolves an environment variable.
func (r *Resolver) resolveEnv(parts []string) (interface{}, error) {
	if len(parts) == 0 {
		return nil, fmt.Errorf("environment variable name required")
	}

	val := os.Getenv(parts[0])
	if val == "" {
		return nil, fmt.Errorf("environment variable not set: %s", parts[0])
	}

	return val, nil
}

// navigateValue navigates through nested map/slice structures.
func (r *Resolver) navigateValue(val interface{}, path []string) (interface{}, error) {
	for _, part := range path {
		switch v := val.(type) {
		case map[string]interface{}:
			var ok bool
			val, ok = v[part]
			if !ok {
				return nil, fmt.Errorf("field not found: %s", part)
			}
		case []interface{}:
			// Try to parse as index
			idx, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("expected array index, got: %s", part)
			}
			if idx < 0 || idx >= len(v) {
				return nil, fmt.Errorf("array index out of bounds: %d", idx)
			}
			val = v[idx]
		default:
			return nil, fmt.Errorf("cannot navigate into %T", val)
		}
	}
	return val, nil
}

// generateUUID generates a unique identifier.
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
