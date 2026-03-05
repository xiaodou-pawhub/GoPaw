// Package plugin defines the public interfaces that all GoPaw plugins must implement.
package plugin

import (
	"context"
	"fmt"
)

// ToolProperty describes a single parameter property in JSON Schema style.
type ToolProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// ToolParameters is the JSON Schema object describing a tool's accepted parameters.
type ToolParameters struct {
	Type       string                  `json:"type"`
	Properties map[string]ToolProperty `json:"properties"`
	Required   []string                `json:"required"`
}

// ToolResult encapsulates the outcome of a tool execution.
type ToolResult struct {
	// LLMOutput is the raw result data intended for the LLM's context.
	LLMOutput string
	// UserOutput is an optional friendly message to display to the user.
	UserOutput string
	// IsError indicates if the execution failed.
	IsError bool
	// Silent marks the tool execution as background information (not visible to user).
	Silent bool
	// Media contains references to media files (media://uuid) produced by the tool.
	Media []string
}

// Tool is the interface that every callable tool must implement.
type Tool interface {
	Name() string
	Description() string
	Parameters() ToolParameters
	// Execute runs the tool with the given arguments.
	// args values are typed according to the JSON Schema (string, float64, bool …).
	Execute(ctx context.Context, args map[string]interface{}) *ToolResult
}

// ContextualTool is an optional interface for tools that need to know
// the message context (channel, chatID, session, user) they are running in.
type ContextualTool interface {
	Tool
	SetContext(channelID, chatID, sessionID, userID string)
}
// GuardedTool is an optional interface for tools that require manual
// user approval before execution.
type GuardedTool interface {
	Tool
	// RequireApproval returns true if the tool should be gated by a user confirmation.
	RequireApproval(args map[string]interface{}) bool
}

// AsyncCallback is invoked when an AsyncTool completes its work.
type AsyncCallback func(result *ToolResult)

// AsyncTool is an optional interface for tools that perform long-running
// operations and return their results via a callback.
type AsyncTool interface {
	Tool
	SetCallback(cb AsyncCallback)
}

// ── ToolResult Helpers ──────────────────────────────────────────────────────

// NewToolResult creates a standard successful tool result.
func NewToolResult(llmOutput string) *ToolResult {
	return &ToolResult{
		LLMOutput: llmOutput,
	}
}

// UserResult creates a successful result with a specific message for the user.
func UserResult(llmOutput, userOutput string) *ToolResult {
	return &ToolResult{
		LLMOutput:  llmOutput,
		UserOutput: userOutput,
	}
}

// ErrorResult creates a failure result with an explanation for the LLM.
func ErrorResult(err string) *ToolResult {
	return &ToolResult{
		LLMOutput: fmt.Sprintf("Error: %s", err),
		IsError:   true,
	}
}

// SilentResult creates a successful result that won't be shown in the UI.
func SilentResult(llmOutput string) *ToolResult {
	return &ToolResult{
		LLMOutput: llmOutput,
		Silent:    true,
	}
}
