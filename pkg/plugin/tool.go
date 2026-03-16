// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

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

// SummaryCapableTool is an optional interface for tools that can provide
// a human-readable summary of their intended operation for approval UIs.
type SummaryCapableTool interface {
	Tool
	Summary(args map[string]interface{}) string
}

// ApprovalSummaryCapable is an optional interface for tools that support
// interactive approval cards with user-friendly summaries and optional details.
// This is used for Feishu approval cards with collapsible panels.
type ApprovalSummaryCapable interface {
	Tool
	// ApprovalSummary returns a user-friendly summary for the approval card.
	// This is always shown in the card.
	ApprovalSummary(args map[string]interface{}) string
	// ApprovalDetail returns the full detail for the collapsible panel.
	// Return empty string if no detail panel is needed.
	ApprovalDetail(args map[string]interface{}) string
}

// AutonomyLevel represents the autonomy level for tool execution.
// L1: Auto-execute, only log (safe operations)
// L2: Auto-execute, notify user (regular operations)
// L3: Require explicit approval (sensitive operations)
type AutonomyLevel int

const (
	AutonomyL1 AutonomyLevel = 1
	AutonomyL2 AutonomyLevel = 2
	AutonomyL3 AutonomyLevel = 3
)

// String returns the string representation of autonomy level.
func (a AutonomyLevel) String() string {
	switch a {
	case AutonomyL1:
		return "L1"
	case AutonomyL2:
		return "L2"
	case AutonomyL3:
		return "L3"
	default:
		return "L2"
	}
}

// AutonomyTool is an optional interface for tools that declare their autonomy level.
// This allows tools to define their own permission requirements in code,
// making it harder to tamper with compared to external configuration.
type AutonomyTool interface {
	Tool
	// AutonomyLevel returns the autonomy level for this tool.
	// L1: Safe operations (read, search) - auto execute
	// L2: Regular operations (write, send) - auto execute + notify
	// L3: Sensitive operations (delete, shell) - require approval
	AutonomyLevel() AutonomyLevel
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
