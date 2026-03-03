// Package tools provides the built-in Tool implementations for GoPaw.
package tools

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

// FileWriteTool writes content to a local file, creating parent directories as needed.
type FileWriteTool struct{}

func (t *FileWriteTool) Name() string { return "file_write" }

func (t *FileWriteTool) Description() string {
	return "Write text content to a local file. Creates parent directories automatically. Overwrites existing files."
}

func (t *FileWriteTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"path": {
				Type:        "string",
				Description: "Absolute or relative path of the file to write.",
			},
			"content": {
				Type:        "string",
				Description: "Text content to write into the file.",
			},
		},
		Required: []string{"path", "content"},
	}
}

func (t *FileWriteTool) Execute(_ context.Context, args map[string]interface{}) (string, error) {
	path, ok := args["path"].(string)
	if !ok || path == "" {
		return "", fmt.Errorf("file_write: 'path' argument is required")
	}
	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("file_write: 'content' argument is required")
	}

	// Create parent directories if they don't exist.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("file_write: create directories for %q: %w", path, err)
	}

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("file_write: write %q: %w", path, err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil
}
