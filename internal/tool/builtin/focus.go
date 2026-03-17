// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package builtin

import (
	"context"
	"fmt"

	"github.com/gopaw/gopaw/internal/focus"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&FocusUpdateTool{})
}

// FocusUpdateTool allows the agent to update focus task status.
type FocusUpdateTool struct {
	focusMgr *focus.Manager
}

// SetFocusManager injects the focus manager.
func (t *FocusUpdateTool) SetFocusManager(mgr *focus.Manager) {
	t.focusMgr = mgr
}

var _ plugin.AutonomyTool = (*FocusUpdateTool)(nil)
var _ plugin.ApprovalSummaryCapable = (*FocusUpdateTool)(nil)
var _ plugin.Tool = (*FocusUpdateTool)(nil)

// Name returns the tool name.
func (t *FocusUpdateTool) Name() string { return "update_focus" }

// AutonomyLevel returns L2 (regular operation - auto execute + notify).
func (t *FocusUpdateTool) AutonomyLevel() plugin.AutonomyLevel {
	return plugin.AutonomyL2
}

// Description returns the tool description.
func (t *FocusUpdateTool) Description() string {
	return "Update the status of a focus task. Use this to mark tasks as completed or in progress. " +
		"Status symbols: '*' or 'completed' for done, '/' or 'in_progress' for doing, ' ' or 'pending' for todo."
}

// Parameters defines the tool parameters.
func (t *FocusUpdateTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"task": {
				Type:        "string",
				Description: "The exact title of the task to update",
			},
			"status": {
				Type:        "string",
				Description: "New status: '*' or 'completed' for done, '/' or 'in_progress' for doing, ' ' or 'pending' for waiting",
				Enum:        []string{"*", "/", " ", "completed", "in_progress", "pending"},
			},
		},
		Required: []string{"task", "status"},
	}
}

// ApprovalSummary returns a user-friendly summary.
func (t *FocusUpdateTool) ApprovalSummary(args map[string]interface{}) string {
	task, _ := args["task"].(string)
	status, _ := args["status"].(string)

	statusText := "pending"
	switch status {
	case "*", "completed":
		statusText = "completed"
	case "/", "in_progress":
		statusText = "in progress"
	}

	return fmt.Sprintf("Update task '%s' to %s", task, statusText)
}

// ApprovalDetail returns detailed information for approval dialog.
func (t *FocusUpdateTool) ApprovalDetail(args map[string]interface{}) string {
	return ""
}

// Execute updates the focus task status.
func (t *FocusUpdateTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	if t.focusMgr == nil {
		return plugin.ErrorResult("focus manager not initialized")
	}

	taskName, ok := args["task"].(string)
	if !ok || taskName == "" {
		return plugin.ErrorResult("task name is required")
	}

	statusStr, ok := args["status"].(string)
	if !ok {
		return plugin.ErrorResult("status is required")
	}

	status := focus.ParseStatus(statusStr)

	if err := t.focusMgr.UpdateTask(taskName, status); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to update task: %v", err))
	}

	statusSymbol := status.Symbol()
	return plugin.NewToolResult(fmt.Sprintf("Task updated: %s %s", statusSymbol, taskName))
}
