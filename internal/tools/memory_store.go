package tools

import (
	"context"
	"fmt"
	"sync"

	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

var (
	ltmStoreMu sync.RWMutex
	ltmStore   *memory.LTMStore
)

// SetLTMStore 设置长期记忆存储，应在 main.go 中完成初始化后调用。
func SetLTMStore(s *memory.LTMStore) {
	ltmStoreMu.Lock()
	defer ltmStoreMu.Unlock()
	ltmStore = s
}

func getLTMStore() *memory.LTMStore {
	ltmStoreMu.RLock()
	defer ltmStoreMu.RUnlock()
	return ltmStore
}

func init() {
	tool.Register(&MemoryStoreTool{})
}

// MemoryStoreTool 将一条信息存入结构化长期记忆（memories.db）。
type MemoryStoreTool struct{}

func (t *MemoryStoreTool) Name() string { return "memory_store" }

func (t *MemoryStoreTool) Description() string {
	return "Save a fact, preference, or important note into long-term structured memory. " +
		"Use category 'core' for permanent user preferences and facts (default), " +
		"'daily' for today's notes, 'conversation' for session context to preserve. " +
		"Each key is unique — storing with the same key updates the existing entry."
}

func (t *MemoryStoreTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"key": {
				Type:        "string",
				Description: "Unique identifier for this memory (e.g. 'user.language', 'project.gopaw.status'). Use dot-notation for namespacing.",
			},
			"content": {
				Type:        "string",
				Description: "The information to remember.",
			},
			"category": {
				Type:        "string",
				Description: "Memory category: 'core' (permanent, default), 'daily' (today's notes), 'conversation' (session context), or any custom label.",
			},
		},
		Required: []string{"key", "content"},
	}
}

func (t *MemoryStoreTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	store := getLTMStore()
	if store == nil {
		return "", fmt.Errorf("memory_store: LTM store not initialized")
	}

	key, _ := params["key"].(string)
	content, _ := params["content"].(string)
	catStr, _ := params["category"].(string)

	if key == "" {
		return "", fmt.Errorf("memory_store: 'key' is required")
	}
	if content == "" {
		return "", fmt.Errorf("memory_store: 'content' is required")
	}

	cat := memory.CategoryCore
	switch catStr {
	case "daily":
		cat = memory.CategoryDaily
	case "conversation":
		cat = memory.CategoryConversation
	case "":
		cat = memory.CategoryCore
	default:
		cat = memory.Category(catStr)
	}

	if err := store.Store(key, content, cat); err != nil {
		return "", fmt.Errorf("memory_store: %w", err)
	}

	return fmt.Sprintf("Memory stored: [%s] %s", cat, key), nil
}
