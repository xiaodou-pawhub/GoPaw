package builtin

import (
	"context"
	"fmt"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&MemoryWriteTool{})
}

type MemoryWriteTool struct{}

func (t *MemoryWriteTool) Name() string { return "memory_write" }

func (t *MemoryWriteTool) Description() string {
	return "Update an existing fact or note in long-term memory."
}

func (t *MemoryWriteTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"key": {
				Type:        "string",
				Description: "The unique key of the memory to update.",
			},
			"content": {
				Type:        "string",
				Description: "The new content.",
			},
		},
		Required: []string{"key", "content"},
	}
}

func (t *MemoryWriteTool) Execute(_ context.Context, params map[string]any) *plugin.ToolResult {
	store := getLTMStore()
	if store == nil {
		return plugin.ErrorResult("LTM store not initialized")
	}

	key, _ := params["key"].(string)
	content, _ := params["content"].(string)

	if key == "" || content == "" {
		return plugin.ErrorResult("both 'key' and 'content' are required")
	}

	if err := store.Store(key, content, ""); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to update memory: %v", err))
	}

	return plugin.SilentResult(fmt.Sprintf("Memory updated: %s", key))
}
