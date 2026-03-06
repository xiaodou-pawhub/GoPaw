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
	tool.Register(&MemorySearchV2Tool{})
}

type MemorySearchV2Tool struct {
	manager *memory.Manager
	session string
}

func (t *MemorySearchV2Tool) Name() string { return "memory_search_v2" }

func (t *MemorySearchV2Tool) Description() string {
	return "Perform advanced semantic search within the current session's history. " +
		"Use this when you need to recall specific details, facts, or past discussions that might not match exact keywords."
}

func (t *MemorySearchV2Tool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"query": {
				Type:        "string",
				Description: "The topic or question to search for in past memories.",
			},
			"limit": {
				Type:        "integer",
				Description: "Max number of results to return. Default 5.",
			},
			"min_score": {
				Type:        "number",
				Description: "Semantic similarity threshold (0.0 to 1.0). Default 0.7.",
			},
		},
		Required: []string{"query"},
	}
}

func (t *MemorySearchV2Tool) SetMemoryManager(m *memory.Manager) {
	t.manager = m
}

func (t *MemorySearchV2Tool) SetContext(channel, chatID, session, user string) {
	t.session = session
}

func (t *MemorySearchV2Tool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	if t.manager == nil {
		return plugin.ErrorResult("memory manager not initialized")
	}

	query, _ := args["query"].(string)
	limit, ok := args["limit"].(float64)
	if !ok || limit <= 0 {
		limit = 5
	}
	minScore, ok := args["min_score"].(float64)
	if !ok || minScore <= 0 {
		minScore = 0.7
	}

	snippets, err := t.manager.Search(ctx, t.session, query, int(limit), minScore)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("search failed: %v", err))
	}

	if len(snippets) == 0 {
		return plugin.NewToolResult("No relevant memories found for the given query.")
	}

	var sb strings.Builder
	sb.WriteString("### 🧠 相关记忆检索结果 (语义匹配)\n\n")
	for i, s := range snippets {
		timeStr := s.CreatedAt.Format("2006-01-02 15:04")
		fmt.Fprintf(&sb, "**%d. [%s] %s**\n> %s\n\n", i+1, timeStr, s.Role, s.Content)
	}

	return plugin.NewToolResult(sb.String())
}
