package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&FileWriteTool{})
}

type FileWriteTool struct{}

var _ plugin.ApprovalSummaryCapable = (*FileWriteTool)(nil)
var _ plugin.AutonomyTool = (*FileWriteTool)(nil)

func (t *FileWriteTool) Name() string { return "write_file" }

// AutonomyLevel returns L2 (regular operation - auto execute + notify)
func (t *FileWriteTool) AutonomyLevel() plugin.AutonomyLevel {
	return plugin.AutonomyL2
}

func (t *FileWriteTool) Description() string {
	return "Write content to a file. Overwrites existing files. Uses atomic write mechanism."
}

// ApprovalSummary returns a user-friendly summary for the approval card.
func (t *FileWriteTool) ApprovalSummary(args map[string]interface{}) string {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	displayContent := content
	if len(displayContent) > 100 {
		displayContent = displayContent[:97] + "..."
	}
	return fmt.Sprintf("📁 **写入文件**\n路  径：%s\n预  览：%s", path, displayContent)
}

// ApprovalDetail returns the full file content for the collapsible panel.
func (t *FileWriteTool) ApprovalDetail(args map[string]interface{}) string {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)
	return fmt.Sprintf("**完整内容**:\n```\n%s\n```\n**文件路径**: %s", content, path)
}

func (t *FileWriteTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"path": {
				Type:        "string",
				Description: "Path to the file.",
			},
			"content": {
				Type:        "string",
				Description: "Content to write.",
			},
		},
		Required: []string{"path", "content"},
	}
}

func (t *FileWriteTool) Execute(_ context.Context, args map[string]interface{}) *plugin.ToolResult {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to create directory: %v", err))
	}

	// Atomic write: Write to tmp file then rename
	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to write temporary file: %v", err))
	}

	if err := os.Rename(tmpFile, path); err != nil {
		_ = os.Remove(tmpFile)
		return plugin.ErrorResult(fmt.Sprintf("failed to rename file: %v", err))
	}

	return plugin.NewToolResult(fmt.Sprintf("successfully wrote to %s", path))
}
