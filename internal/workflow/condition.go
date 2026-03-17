// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package workflow

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Evaluator evaluates workflow conditions.
type Evaluator struct {
	resolver *Resolver
}

// NewEvaluator creates a new condition evaluator.
func NewEvaluator(resolver *Resolver) *Evaluator {
	return &Evaluator{resolver: resolver}
}

// Evaluate evaluates a condition expression.
// Supported operators:
// - ==, != : equality
// - >, >=, <, <= : comparison
// - &&, || : logical AND/OR
// - ! : logical NOT
// - contains : string/array contains
// - startsWith, endsWith : string prefix/suffix
// - exists : check if variable exists
// Examples:
// - "{{steps.validate.output.valid}} == true"
// - "{{steps.validate.output.count}} > 0"
// - "{{var.priority}} == 'high' && {{steps.validate.output.valid}} == true"
// - "{{steps.validate.output.errors}} contains 'critical'"
// - "!{{steps.validate.output.valid}}"
func (e *Evaluator) Evaluate(condition string) (bool, error) {
	if condition == "" {
		return true, nil // Empty condition is always true
	}

	// Replace all variable expressions with their values
	re := regexp.MustCompile(`\{\{\s*([^}]+)\s*\}\}`)
	expr := re.ReplaceAllStringFunc(condition, func(match string) string {
		varExpr := strings.TrimSpace(match[2 : len(match)-2])
		val, err := e.resolver.resolveExpression(varExpr)
		if err != nil {
			return "null"
		}
		return formatValue(val)
	})

	// Evaluate the expression
	return e.evaluateExpression(expr)
}

// evaluateExpression evaluates a simple expression.
func (e *Evaluator) evaluateExpression(expr string) (bool, error) {
	expr = strings.TrimSpace(expr)

	// Handle parentheses
	for {
		start := strings.Index(expr, "(")
		if start == -1 {
			break
		}
		end := findMatchingParen(expr, start)
		if end == -1 {
			return false, fmt.Errorf("unmatched parenthesis")
		}
		inner := expr[start+1 : end]
		result, err := e.evaluateExpression(inner)
		if err != nil {
			return false, err
		}
		expr = expr[:start] + formatValue(result) + expr[end+1:]
	}

	// Handle logical OR
	if parts := splitByOperator(expr, "||"); len(parts) > 1 {
		for _, part := range parts {
			result, err := e.evaluateExpression(strings.TrimSpace(part))
			if err != nil {
				return false, err
			}
			if result {
				return true, nil
			}
		}
		return false, nil
	}

	// Handle logical AND
	if parts := splitByOperator(expr, "&&"); len(parts) > 1 {
		for _, part := range parts {
			result, err := e.evaluateExpression(strings.TrimSpace(part))
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil
			}
		}
		return true, nil
	}

	// Handle NOT
	if strings.HasPrefix(expr, "!") {
		result, err := e.evaluateExpression(strings.TrimSpace(expr[1:]))
		if err != nil {
			return false, err
		}
		return !result, nil
	}

	// Handle comparison operators
	return e.evaluateComparison(expr)
}

// evaluateComparison evaluates a comparison expression.
func (e *Evaluator) evaluateComparison(expr string) (bool, error) {
	// Try different operators in order of precedence
	operators := []string{
		"==", "!=",
		">=", "<=",
		">", "<",
		"contains",
		"startsWith", "endsWith",
		"exists",
	}

	for _, op := range operators {
		if idx := strings.Index(expr, op); idx != -1 {
			left := strings.TrimSpace(expr[:idx])
			right := strings.TrimSpace(expr[idx+len(op):])

			switch op {
			case "==":
				return e.compareEqual(left, right)
			case "!=":
				result, err := e.compareEqual(left, right)
				if err != nil {
					return false, err
				}
				return !result, nil
			case ">":
				return e.compareGreater(left, right)
			case ">=":
				result, err := e.compareGreater(left, right)
				if err != nil {
					return false, err
				}
				equal, _ := e.compareEqual(left, right)
				return result || equal, nil
			case "<":
				result, err := e.compareGreater(right, left)
				if err != nil {
					return false, err
				}
				return result, nil
			case "<=":
				result, err := e.compareGreater(right, left)
				if err != nil {
					return false, err
				}
				equal, _ := e.compareEqual(left, right)
				return result || equal, nil
			case "contains":
				return e.contains(left, right)
			case "startsWith":
				return e.startsWith(left, right)
			case "endsWith":
				return e.endsWith(left, right)
			case "exists":
				return left != "null" && left != "", nil
			}
		}
	}

	// If no operator found, treat as boolean value
	return parseBool(expr), nil
}

// compareEqual compares two values for equality.
func (e *Evaluator) compareEqual(left, right string) (bool, error) {
	// Try numeric comparison
	leftNum, leftErr := strconv.ParseFloat(left, 64)
	rightNum, rightErr := strconv.ParseFloat(right, 64)
	if leftErr == nil && rightErr == nil {
		return leftNum == rightNum, nil
	}

	// String comparison (remove quotes)
	leftStr := strings.Trim(left, `"'`)
	rightStr := strings.Trim(right, `"'`)

	return leftStr == rightStr, nil
}

// compareGreater compares if left > right.
func (e *Evaluator) compareGreater(left, right string) (bool, error) {
	// Try numeric comparison
	leftNum, leftErr := strconv.ParseFloat(left, 64)
	rightNum, rightErr := strconv.ParseFloat(right, 64)
	if leftErr == nil && rightErr == nil {
		return leftNum > rightNum, nil
	}

	// String comparison
	return left > right, nil
}

// contains checks if left contains right.
func (e *Evaluator) contains(left, right string) (bool, error) {
	leftStr := strings.Trim(left, `"'`)
	rightStr := strings.Trim(right, `"'`)
	return strings.Contains(leftStr, rightStr), nil
}

// startsWith checks if left starts with right.
func (e *Evaluator) startsWith(left, right string) (bool, error) {
	leftStr := strings.Trim(left, `"'`)
	rightStr := strings.Trim(right, `"'`)
	return strings.HasPrefix(leftStr, rightStr), nil
}

// endsWith checks if left ends with right.
func (e *Evaluator) endsWith(left, right string) (bool, error) {
	leftStr := strings.Trim(left, `"'`)
	rightStr := strings.Trim(right, `"'`)
	return strings.HasSuffix(leftStr, rightStr), nil
}

// findMatchingParen finds the matching closing parenthesis.
func findMatchingParen(expr string, start int) int {
	count := 1
	for i := start + 1; i < len(expr); i++ {
		switch expr[i] {
		case '(':
			count++
		case ')':
			count--
			if count == 0 {
				return i
			}
		}
	}
	return -1
}

// splitByOperator splits expression by operator, respecting parentheses.
func splitByOperator(expr, op string) []string {
	var parts []string
	var start int
	depth := 0

	for i := 0; i <= len(expr)-len(op); i++ {
		switch expr[i] {
		case '(':
			depth++
		case ')':
			depth--
		}

		if depth == 0 && expr[i:i+len(op)] == op {
			parts = append(parts, strings.TrimSpace(expr[start:i]))
			start = i + len(op)
			i += len(op) - 1
		}
	}

	if start < len(expr) {
		parts = append(parts, strings.TrimSpace(expr[start:]))
	}

	return parts
}

// formatValue formats a value for expression evaluation.
func formatValue(val interface{}) string {
	if val == nil {
		return "null"
	}
	switch v := val.(type) {
	case bool:
		if v {
			return "true"
		}
		return "false"
	case string:
		return fmt.Sprintf(`"%s"`, v)
	case int, int64, float64:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf(`"%v"`, v)
	}
}

// parseBool parses a boolean value from string.
func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "1" || s == "yes"
}
