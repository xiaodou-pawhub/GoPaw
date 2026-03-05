package builtin

import (
	"context"
	"fmt"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&MemoryReadTool{})
}

type MemoryReadTool struct{}

func (t *MemoryReadTool) Name() string { return "memory_read" }

func (t *MemoryReadTool) Description() string {
	return "Retrieve a specific fact or note from long-term memory by its key."
}

func (t *MemoryReadTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"key": {
				Type:        "string",
				Description: "The unique key of the memory to retrieve.",
			},
		},
		Required: []string{"key"},
	}
}

func (t *MemoryReadTool) Execute(_ context.Context, params map[string]any) *plugin.ToolResult {
	store := getLTMStore()
	if store == nil {
		return plugin.ErrorResult("LTM store not initialized")
	}

	key, _ := params["key"].(string)
	if key == "" {
		return plugin.ErrorResult("'key' is required")
	}

	entry, err := store.Get(key)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("memory not found for key: %s", key))
	}

	return plugin.NewToolResult(entry.Content)
}
