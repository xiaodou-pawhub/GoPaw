// Package tools provides the built-in Tool implementations for GoPaw.
package tools

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&FileReadTool{})
}

const maxFileReadSize = 1 << 20 // 1 MB

// FileReadTool reads a local file and returns its contents as a string.
type FileReadTool struct{}

func (t *FileReadTool) Name() string { return "file_read" }

func (t *FileReadTool) Description() string {
	return "Read the contents of a local file. Returns the text content. Limit: 1MB."
}

func (t *FileReadTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"path": {
				Type:        "string",
				Description: "Absolute or relative path to the file to read.",
			},
		},
		Required: []string{"path"},
	}
}

func (t *FileReadTool) Execute(_ context.Context, args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("file_read: 'path' argument is required")
	}

	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("file_read: open %q: %w", path, err)
	}
	defer f.Close()

	content, err := io.ReadAll(io.LimitReader(f, maxFileReadSize))
	if err != nil {
		return "", fmt.Errorf("file_read: read %q: %w", path, err)
	}
	return string(content), nil
}
