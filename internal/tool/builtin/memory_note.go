package builtin

import (
	"context"
	"fmt"
	"time"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&MemoryNoteTool{})
}

type MemoryNoteTool struct{}

func (t *MemoryNoteTool) Name() string { return "memory_note" }

func (t *MemoryNoteTool) Description() string {
	return "Append a note to the daily log. Useful for tracking events, thoughts, and small discoveries."
}

func (t *MemoryNoteTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"content": {
				Type:        "string",
				Description: "The note content to append.",
			},
		},
		Required: []string{"content"},
	}
}

func (t *MemoryNoteTool) Execute(_ context.Context, params map[string]any) *plugin.ToolResult {
	store := getLTMStore()
	if store == nil {
		return plugin.ErrorResult("LTM store not initialized")
	}

	content, _ := params["content"].(string)
	if content == "" {
		return plugin.ErrorResult("'content' is required")
	}

	key := fmt.Sprintf("note.%d", time.Now().Unix())
	if err := store.Store(key, content, "daily"); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to save note: %v", err))
	}

	return plugin.SilentResult(fmt.Sprintf("Note saved with key: %s", key))
}
