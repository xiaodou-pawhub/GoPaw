// Package tool manages the registration and lookup of Tool plugins.
package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
)

// Executor wraps a Registry and provides a single Execute entry point used by the agent.
type Executor struct {
	registry *Registry
	logger   *zap.Logger
}

// NewExecutor creates an Executor backed by the given registry.
func NewExecutor(registry *Registry, logger *zap.Logger) *Executor {
	return &Executor{registry: registry, logger: logger}
}

// Execute finds the named tool and runs it with the provided JSON-encoded arguments.
// argsJSON should be the raw JSON string from the LLM function-call arguments field.
func (e *Executor) Execute(ctx context.Context, toolName, argsJSON string) (string, error) {
	t, err := e.registry.Get(toolName)
	if err != nil {
		return "", fmt.Errorf("executor: tool not found: %w", err)
	}

	var args map[string]interface{}
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("executor: parse args for %q: %w", toolName, err)
		}
	}
	if args == nil {
		args = make(map[string]interface{})
	}

	e.logger.Info("tool executing", zap.String("tool", toolName), zap.Any("args", args))

	result, err := t.Execute(ctx, args)
	if err != nil {
		e.logger.Warn("tool execution failed",
			zap.String("tool", toolName), zap.Error(err))
		return "", fmt.Errorf("executor: %s: %w", toolName, err)
	}
	return result, nil
}
