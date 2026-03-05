package builtin

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&ShellTool{})
}

type ShellTool struct{}

var _ plugin.GuardedTool = (*ShellTool)(nil)

func (t *ShellTool) Name() string { return "shell" }

func (t *ShellTool) Description() string {
	return "Execute a shell command and return its output. Use for system inspection, running scripts, or one-off commands. Dangerous commands require manual approval."
}

// RequireApproval checks if the command is dangerous.
func (t *ShellTool) RequireApproval(args map[string]interface{}) bool {
	cmdStr, _ := args["command"].(string)
	cmdStr = strings.ToLower(cmdStr)

	// List of high-risk commands or patterns
	dangerousKeywords := []string{
		"rm ", ">", "kill ", "shutdown", "reboot", "format ", "mkfs", "dd ",
	}

	for _, k := range dangerousKeywords {
		if strings.Contains(cmdStr, k) {
			return true
		}
	}
	return false
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
