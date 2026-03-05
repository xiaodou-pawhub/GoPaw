package builtin

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&FileReadTool{})
}

type FileReadTool struct{}

func (t *FileReadTool) Name() string { return "read_file" }

func (t *FileReadTool) Description() string {
	return "Read the contents of a file. Supports reading specific line ranges with line numbers."
}

func (t *FileReadTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"path": {
				Type:        "string",
				Description: "Path to the file.",
			},
			"start_line": {
				Type:        "integer",
				Description: "Optional 1-based start line (inclusive).",
			},
			"end_line": {
				Type:        "integer",
				Description: "Optional 1-based end line (inclusive).",
			},
		},
		Required: []string{"path"},
	}
}

func (t *FileReadTool) Execute(_ context.Context, args map[string]interface{}) *plugin.ToolResult {
	path, _ := args["path"].(string)
	startLine := 1
	if val, ok := args["start_line"].(float64); ok {
		startLine = int(val)
	}
	endLine := 0
	if val, ok := args["end_line"].(float64); ok {
		endLine = int(val)
	}

	file, err := os.Open(path)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to open file: %v", err))
	}
	defer file.Close()

	var sb strings.Builder
	scanner := bufio.NewScanner(file)
	currentLine := 0
	for scanner.Scan() {
		currentLine++
		if currentLine < startLine {
			continue
		}
		if endLine > 0 && currentLine > endLine {
			break
		}
		sb.WriteString(fmt.Sprintf("%6d | %s\n", currentLine, scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("error reading file: %v", err))
	}

	return plugin.NewToolResult(sb.String())
}
