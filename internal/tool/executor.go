// Package tool manages the tool lifecycle and execution for the agent.
package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/gopaw/gopaw/pkg/plugin"
	"go.uber.org/zap"
)

const maxToolOutputRunes = 50_000

// ApprovalUI describes the capability to prompt a human for tool execution permission.
type ApprovalUI interface {
	RequestApproval(ctx context.Context, req *ApprovalRequest) error
}

// Executor provides a higher-level API for the agent to call tools.
type Executor struct {
	registry   *Registry
	logger     *zap.Logger
	approvalUI ApprovalUI
}

// NewExecutor creates an Executor backed by the given registry.
func NewExecutor(registry *Registry, logger *zap.Logger) *Executor {
	return &Executor{
		registry: registry,
		logger:   logger.Named("tool_executor"),
	}
}

// SetApprovalUI injects the UI handler for gated tools.
func (e *Executor) SetApprovalUI(ui ApprovalUI) {
	e.approvalUI = ui
}

// Execute parses argsJSON and calls the tool identified by toolName.
func (e *Executor) Execute(ctx context.Context, toolName, argsJSON, channel, session, user string) (string, error) {
	// 1. Parse arguments once in a central place.
	var args map[string]interface{}
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return fmt.Sprintf("Error: failed to parse arguments: %v", err), nil
		}
	}
	if args == nil {
		args = make(map[string]interface{})
	}

	e.logger.Info("tool executing",
		zap.String("tool", toolName),
		zap.String("session", session),
		zap.Any("args", args))

	// 2. Check for manual approval if needed.
	t, ok := e.registry.Get(toolName)
	if ok {
		if gt, ok := t.(plugin.GuardedTool); ok && gt.RequireApproval(args) {
			if e.approvalUI == nil {
				return "Error: this tool requires manual approval but no approval handler is configured", nil
			}

			req := GlobalApprovalStore.CreateRequest(toolName, args, channel, session)
			if err := e.approvalUI.RequestApproval(ctx, req); err != nil {
				return fmt.Sprintf("Error: failed to send approval request: %v", err), nil
			}

			e.logger.Info("tool execution suspended, waiting for approval", zap.String("request_id", req.ID))
			verdict := GlobalApprovalStore.WaitForVerdict(ctx, req, 10*time.Minute)
			
			if verdict != VerdictAllowed {
				return fmt.Sprintf("Error: execution was %s by the user", verdict), nil
			}
			e.logger.Info("tool execution approved", zap.String("request_id", req.ID))
		}
	}

	// 3. Safe execution with panic recovery.
	var result *plugin.ToolResult
	func() {
		defer func() {
			if r := recover(); r != nil {
				e.logger.Error("tool panic recovered",
					zap.String("tool", toolName),
					zap.Any("panic", r),
					zap.String("stack", string(debug.Stack())))
				result = plugin.ErrorResult(fmt.Sprintf("internal tool error: %v", r))
			}
		}()
		result = e.registry.Execute(ctx, toolName, args, channel, session, user)
	}()

	if result.IsError {
		e.logger.Warn("tool execution failed",
			zap.String("tool", toolName),
			zap.String("output", result.LLMOutput))
	} else {
		e.logger.Info("tool execution completed",
			zap.String("tool", toolName),
			zap.Int("output_len", len(result.LLMOutput)))
	}

	// 4. Truncate output if it exceeds limits to prevent context blowing up.
	output := result.LLMOutput
	if len(output) > maxToolOutputRunes {
		output = output[:maxToolOutputRunes] + "\n\n[Output truncated due to length]"
	}

	return output, nil
}
