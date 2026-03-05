package builtin

import (
	"context"
	"fmt"
	"strings"

	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&MemoryRecallTool{})
}

type MemoryRecallTool struct{}

func (t *MemoryRecallTool) Name() string { return "memory_recall" }

func (t *MemoryRecallTool) Description() string {
	return "Search long-term memory for relevant facts or notes by keyword or category."
}

func (t *MemoryRecallTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"query": {
				Type:        "string",
				Description: "Keyword or phrase to search for.",
			},
			"category": {
				Type:        "string",
				Description: "Optional category filter.",
			},
		},
		Required: []string{"query"},
	}
}

func (t *MemoryRecallTool) Execute(_ context.Context, params map[string]any) *plugin.ToolResult {
	store := getLTMStore()
	if store == nil {
		return plugin.ErrorResult("LTM store not initialized")
	}

	query, _ := params["query"].(string)
	catStr, _ := params["category"].(string)

	entries, err := store.Recall(query, 5, memory.Category(catStr))
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("search failed: %v", err))
	}

	if len(entries) == 0 {
		return plugin.NewToolResult("No matching memories found.")
	}

	var sb strings.Builder
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("[%s] %s: %s\n", e.Category, e.Key, e.Content))
	}

	return plugin.NewToolResult(sb.String())
}
