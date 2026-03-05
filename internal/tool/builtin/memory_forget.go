package builtin

import (
	"context"
	"fmt"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&MemoryForgetTool{})
}

type MemoryForgetTool struct{}

func (t *MemoryForgetTool) Name() string { return "memory_forget" }

func (t *MemoryForgetTool) Description() string {
	return "Permanently remove a fact or note from long-term memory by its key."
}

func (t *MemoryForgetTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"key": {
				Type:        "string",
				Description: "The unique key of the memory to remove.",
			},
		},
		Required: []string{"key"},
	}
}

func (t *MemoryForgetTool) Execute(_ context.Context, params map[string]any) *plugin.ToolResult {
	store := getLTMStore()
	if store == nil {
		return plugin.ErrorResult("LTM store not initialized")
	}

	key, _ := params["key"].(string)
	if key == "" {
		return plugin.ErrorResult("'key' is required")
	}

	found, err := store.Forget(key)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to forget memory: %v", err))
	}
	if !found {
		return plugin.NewToolResult(fmt.Sprintf("Key %q not found, nothing to forget.", key))
	}

	return plugin.SilentResult(fmt.Sprintf("Memory forgotten: %s", key))
}
