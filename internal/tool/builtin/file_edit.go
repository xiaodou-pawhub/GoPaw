package builtin

import (
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
	return "Precisely edit a file by replacing a specific string snippet ('old_str') with a new snippet ('new_str'). " +
		"The 'old_str' must be unique in the file to avoid ambiguity. This is preferred over 'write_file' for existing files."
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
				Description: "The exact literal string to find. Must be unique within the file.",
			},
			"new_str": {
				Type:        "string",
				Description: "The literal string to replace it with.",
			},
		},
		Required: []string{"path", "old_str", "new_str"},
	}
}

func (t *FileEditTool) Execute(_ context.Context, args map[string]interface{}) *plugin.ToolResult {
	path, _ := args["path"].(string)
	oldStr, _ := args["old_str"].(string)
	newStr, _ := args["new_str"].(string)

	if oldStr == "" {
		return plugin.ErrorResult("'old_str' cannot be empty")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to read file: %v", err))
	}

	fullText := string(content)

	// Check occurrences
	count := strings.Count(fullText, oldStr)
	if count == 0 {
		return plugin.ErrorResult(fmt.Sprintf("could not find 'old_str' in %s. Ensure the text matches exactly, including whitespace.", path))
	}
	if count > 1 {
		return plugin.ErrorResult(fmt.Sprintf("'old_str' appears %d times in %s. Please provide a more unique snippet to avoid incorrect replacements.", count, path))
	}

	// Perform replacement
	newText := strings.Replace(fullText, oldStr, newStr, 1)

	// Write back atomically (reusing temporary file logic)
	tmpFile := path + ".tmp"
	if err := os.WriteFile(tmpFile, []byte(newText), 0644); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to write edit: %v", err))
	}

	if err := os.Rename(tmpFile, path); err != nil {
		_ = os.Remove(tmpFile)
		return plugin.ErrorResult(fmt.Sprintf("failed to save edit: %v", err))
	}

	return plugin.NewToolResult(fmt.Sprintf("successfully edited %s", path))
}
