// Package tools provides the built-in Tool implementations for GoPaw.
package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&FileSearchTool{})
}

// FileSearchTool searches for files matching a glob pattern within a directory.
type FileSearchTool struct{}

func (t *FileSearchTool) Name() string { return "file_search" }

func (t *FileSearchTool) Description() string {
	return "Search for files matching a glob pattern in a directory. Returns a list of matching file paths."
}

func (t *FileSearchTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"dir": {
				Type:        "string",
				Description: "Directory to search in.",
			},
			"pattern": {
				Type:        "string",
				Description: "Glob pattern to match filenames, e.g. '*.go' or '**/*.md'.",
			},
		},
		Required: []string{"dir", "pattern"},
	}
}

func (t *FileSearchTool) Execute(_ context.Context, args map[string]interface{}) (string, error) {
	dir, ok := args["dir"].(string)
	if !ok || dir == "" {
		return "", fmt.Errorf("file_search: 'dir' argument is required")
	}
	pattern, ok := args["pattern"].(string)
	if !ok || pattern == "" {
		return "", fmt.Errorf("file_search: 'pattern' argument is required")
	}

	// Build a full glob expression.
	globExpr := filepath.Join(dir, pattern)
	matches, err := filepath.Glob(globExpr)
	if err != nil {
		return "", fmt.Errorf("file_search: glob %q: %w", globExpr, err)
	}

	if len(matches) == 0 {
		return "No files found matching the pattern.", nil
	}

	return strings.Join(matches, "\n"), nil
}
