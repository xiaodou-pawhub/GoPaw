package builtin

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&ShellTool{})
}

type ShellTool struct{}

var _ plugin.ApprovalSummaryCapable = (*ShellTool)(nil)
var _ plugin.AutonomyTool = (*ShellTool)(nil)

func (t *ShellTool) Name() string { return "shell" }

// AutonomyLevel returns L3 (sensitive operation - require approval)
func (t *ShellTool) AutonomyLevel() plugin.AutonomyLevel {
	return plugin.AutonomyL3
}

func (t *ShellTool) Description() string {
	return "Execute a shell command and return its output. Use for system inspection, running scripts, or one-off commands."
}

// ApprovalSummary returns a user-friendly summary for the approval card.
func (t *ShellTool) ApprovalSummary(args map[string]interface{}) string {
	command, _ := args["command"].(string)
	timeoutSec, ok := args["timeout_sec"].(float64)
	if !ok {
		timeoutSec = 60
	}
	displayCmd := command
	if len(displayCmd) > 80 {
		displayCmd = displayCmd[:77] + "..."
	}
	return fmt.Sprintf("💻 **执行 Shell 命令**\n命  令：%s\n超  时：%.0f 秒", displayCmd, timeoutSec)
}

// ApprovalDetail returns the full command detail for the collapsible panel.
func (t *ShellTool) ApprovalDetail(args map[string]interface{}) string {
	command, _ := args["command"].(string)
	timeoutSec, ok := args["timeout_sec"].(float64)
	if !ok {
		timeoutSec = 60
	}
	return fmt.Sprintf("**完整命令**:\n```\n%s\n```\n**超时时间**: %.0f 秒", command, timeoutSec)
}

func (t *ShellTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"command": {
				Type:        "string",
				Description: "The full shell command to execute.",
			},
			"timeout_sec": {
				Type:        "integer",
				Description: "Optional timeout in seconds (default 60).",
			},
		},
		Required: []string{"command"},
	}
}

func (t *ShellTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	cmdStr, _ := args["command"].(string)
	timeoutSec := 60
	if val, ok := args["timeout_sec"].(float64); ok {
		timeoutSec = int(val)
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "sh", "-c", cmdStr)
	output, err := cmd.CombinedOutput()

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return plugin.ErrorResult(fmt.Sprintf("command timed out after %ds", timeoutSec))
		}
		return plugin.ErrorResult(fmt.Sprintf("command failed: %v\nOutput: %s", err, string(output)))
	}

	return plugin.NewToolResult(string(output))
}
