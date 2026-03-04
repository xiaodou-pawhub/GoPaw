package tools

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

// MemoryRecallTool searches the structured long-term memory using FTS5 + time-decay ranking.
type MemoryRecallTool struct{}

func (t *MemoryRecallTool) Name() string { return "memory_recall" }

func (t *MemoryRecallTool) Description() string {
	return "Search long-term memory for relevant facts, preferences, or context. " +
		"Uses full-text search with time-decay ranking (recent entries score higher). " +
		"Core memories receive a relevance boost. Returns scored results. " +
		"Use this before answering questions about user preferences or past decisions."
}

func (t *MemoryRecallTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"query": {
				Type:        "string",
				Description: "Keywords or phrase to search for.",
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum number of results to return (default: 5).",
			},
			"category": {
				Type:        "string",
				Description: "Filter by category: 'core', 'daily', 'conversation', or a custom label. Omit for all categories.",
			},
		},
		Required: []string{"query"},
	}
}

func (t *MemoryRecallTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	store := getLTMStore()
	if store == nil {
		return "", fmt.Errorf("memory_recall: LTM store not initialized")
	}

	query, _ := params["query"].(string)
	limit := 5
	if v, ok := params["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}
	catStr, _ := params["category"].(string)
	cat := memory.Category(catStr)

	entries, err := store.Recall(query, limit, cat)
	if err != nil {
		return "", fmt.Errorf("memory_recall: %w", err)
	}

	// Apply time decay (half-life: core=∞, daily=7d, conversation=14d, others=30d)
	entries = memory.ApplyTimeDecay(entries, 30)

	if len(entries) == 0 {
		return "No memories found matching that query.", nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Found %d memories:\n\n", len(entries))
	for _, e := range entries {
		fmt.Fprintf(&sb, "**[%s]** `%s`\n%s\n\n", e.Category, e.Key, e.Content)
	}
	return sb.String(), nil
}
