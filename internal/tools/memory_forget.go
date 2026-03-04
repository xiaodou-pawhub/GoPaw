package tools

import (
	"context"
	"fmt"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&MemoryForgetTool{})
}

// MemoryForgetTool deletes a specific memory entry by its key.
type MemoryForgetTool struct{}

func (t *MemoryForgetTool) Name() string { return "memory_forget" }

func (t *MemoryForgetTool) Description() string {
	return "Delete a specific memory entry by its key. " +
		"Use this when a memory is outdated, incorrect, or explicitly requested for removal. " +
		"Only deletes from structured memory (memories.db), not from MEMORY.md files."
}

func (t *MemoryForgetTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"key": {
				Type:        "string",
				Description: "The unique key of the memory entry to delete.",
			},
		},
		Required: []string{"key"},
	}
}

func (t *MemoryForgetTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	store := getLTMStore()
	if store == nil {
		return "", fmt.Errorf("memory_forget: LTM store not initialized")
	}

	key, _ := params["key"].(string)
	if key == "" {
		return "", fmt.Errorf("memory_forget: 'key' is required")
	}

	found, err := store.Forget(key)
	if err != nil {
		return "", fmt.Errorf("memory_forget: %w", err)
	}
	if !found {
		return fmt.Sprintf("Memory key '%s' not found (nothing deleted).", key), nil
	}
	return fmt.Sprintf("Memory deleted: %s", key), nil
}
