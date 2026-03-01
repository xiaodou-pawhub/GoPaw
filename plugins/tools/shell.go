// Package tools provides the built-in Tool implementations for GoPaw.
package tools

import (
	"bytes"
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

const defaultShellTimeout = 30 // seconds

// ShellTool executes shell commands and captures combined stdout and stderr.
type ShellTool struct{}

func (t *ShellTool) Name() string { return "shell_execute" }

func (t *ShellTool) Description() string {
	return "Execute a shell command and return its stdout and stderr output. Use with caution."
}

func (t *ShellTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"command": {
				Type:        "string",
				Description: "The shell command to execute.",
			},
			"timeout": {
				Type:        "integer",
				Description: "Timeout in seconds (default: 30, max: 300).",
			},
		},
		Required: []string{"command"},
	}
}

func (t *ShellTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	command, ok := args["command"].(string)
	if !ok || command == "" {
		return "", fmt.Errorf("shell_execute: 'command' argument is required")
	}

	timeoutSec := defaultShellTimeout
	if v, ok := args["timeout"]; ok {
		switch tv := v.(type) {
		case float64:
			timeoutSec = int(tv)
		case int:
			timeoutSec = tv
		}
	}
	if timeoutSec <= 0 || timeoutSec > 300 {
		timeoutSec = defaultShellTimeout
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "sh", "-c", command)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\nSTDERR:\n" + stderr.String()
	}

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return output, fmt.Errorf("shell_execute: command timed out after %ds", timeoutSec)
		}
		// Return the output along with the error so the agent can see what happened.
		return output, fmt.Errorf("shell_execute: command failed: %w", err)
	}

	if output == "" {
		return "(no output)", nil
	}
	return output, nil
}
