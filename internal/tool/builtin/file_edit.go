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
	tool.Register(&FileEditTool{})
}

type FileEditTool struct{}

func (t *FileEditTool) Name() string { return "edit_file" }

func (t *FileEditTool) Description() string {
	return "Edit a file using either string replacement or specific line modification. " +
		"In string mode, 'old_str' must be unique. In line mode, 'line_number' is required."
}

func (t *FileEditTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"path": {
				Type:        "string",
				Description: "Path to the file to edit.",
			},
			"old_str": {
				Type:        "string",
				Description: "Snippet mode: exact literal string to find. Must be unique.",
			},
			"new_str": {
				Type:        "string",
				Description: "Snippet or line mode: new content to insert.",
			},
			"line_number": {
				Type:        "integer",
				Description: "Line mode: 1-based line number to modify.",
			},
			"expected_line_content": {
				Type:        "string",
				Description: "Line mode: optional verification of existing line content before modification.",
			},
		},
		Required: []string{"path", "new_str"},
	}
}

func (t *FileEditTool) Execute(_ context.Context, args map[string]interface{}) *plugin.ToolResult {
	path, _ := args["path"].(string)
	newStr, _ := args["new_str"].(string)
	lineNumFloat, hasLine := args["line_number"].(float64)
	oldStr, hasOldStr := args["old_str"].(string)

	if hasLine {
		return t.editByLine(path, int(lineNumFloat), newStr, args)
	}

	if hasOldStr && oldStr != "" {
		return t.editBySnippet(path, oldStr, newStr)
	}

	return plugin.ErrorResult("either 'line_number' or 'old_str' must be provided")
}

func (t *FileEditTool) editBySnippet(path, oldStr, newStr string) *plugin.ToolResult {
	content, err := os.ReadFile(path)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to read file: %v", err))
	}

	fullText := string(content)
	count := strings.Count(fullText, oldStr)
	if count == 0 {
		return plugin.ErrorResult(fmt.Sprintf("could not find 'old_str' in %s", path))
	}
	if count > 1 {
		return plugin.ErrorResult(fmt.Sprintf("'old_str' appears %d times in %s. Snippet must be unique.", count, path))
	}

	newText := strings.Replace(fullText, oldStr, newStr, 1)
	return t.atomicWrite(path, newText)
}

func (t *FileEditTool) editByLine(path string, lineNum int, newStr string, args map[string]interface{}) *plugin.ToolResult {
	if lineNum < 1 {
		return plugin.ErrorResult("line_number must be >= 1")
	}

	file, err := os.Open(path)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to open file: %v", err))
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if lineNum > len(lines) {
		return plugin.ErrorResult(fmt.Sprintf("line %d is beyond file length (%d lines)", lineNum, len(lines)))
	}

	// Optional verification
	if expected, ok := args["expected_line_content"].(string); ok && expected != "" {
		actual := lines[lineNum-1]
		if !strings.Contains(actual, expected) {
			return plugin.ErrorResult(fmt.Sprintf("verification failed: line %d content mismatch. Expected to contain: %q", lineNum, expected))
		}
	}

	lines[lineNum-1] = newStr
	return t.atomicWrite(path, strings.Join(lines, "\n")+"\n")
}

func (t *FileEditTool) atomicWrite(path, content string) *plugin.ToolResult {
	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to write temporary file: %v", err))
	}

	if err := os.Rename(tmpFile, path); err != nil {
		_ = os.Remove(tmpFile)
		return plugin.ErrorResult(fmt.Sprintf("failed to rename file: %v", err))
	}

	return plugin.NewToolResult(fmt.Sprintf("successfully edited %s", path))
}
