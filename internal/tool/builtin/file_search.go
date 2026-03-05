package builtin

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&FileSearchTool{})
}

type FileSearchTool struct{}

func (t *FileSearchTool) Name() string { return "file_search" }

func (t *FileSearchTool) Description() string {
	return "Search for files by name (glob) or content (grep) within the workspace."
}

func (t *FileSearchTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"pattern": {
				Type:        "string",
				Description: "The search pattern (regex for grep, glob for file names).",
			},
			"type": {
				Type:        "string",
				Description: "Search type: 'grep' (content) or 'glob' (file name).",
				Enum:        []string{"grep", "glob"},
			},
			"path": {
				Type:        "string",
				Description: "Root path to search from (default is current directory).",
			},
		},
		Required: []string{"pattern", "type"},
	}
}

func (t *FileSearchTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	pattern, _ := args["pattern"].(string)
	searchType, _ := args["type"].(string)
	rootPath := "."
	if val, ok := args["path"].(string); ok && val != "" {
		rootPath = val
	}

	var cmd *exec.Cmd
	if searchType == "grep" {
		// Use grep -rn to find content with line numbers
		cmd = exec.CommandContext(ctx, "grep", "-rn", "--color=never", pattern, rootPath)
	} else {
		// Use find for globbing
		cmd = exec.CommandContext(ctx, "find", rootPath, "-name", pattern)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) == 0 {
			return plugin.NewToolResult("No matches found.")
		}
		return plugin.ErrorResult(fmt.Sprintf("search failed: %v\nOutput: %s", err, string(output)))
	}

	return plugin.NewToolResult(string(output))
}
