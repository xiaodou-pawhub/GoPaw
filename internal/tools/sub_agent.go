package tools

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
)

// subAgentDepthKey is the context key used to track sub-agent recursion depth.
type subAgentDepthKey struct{}

// maxSubAgentDepth caps the nesting level to prevent runaway recursion.
const maxSubAgentDepth = 1

var (
	subAgentFnMu sync.RWMutex
	subAgentFn   func(ctx context.Context, req *types.Request) (string, error)
)

// SetSubAgentFn injects the agent process callback from main.go.
// This avoids a circular import between internal/tools and internal/agent.
// Must be called before any spawn_agent tool invocation.
func SetSubAgentFn(fn func(ctx context.Context, req *types.Request) (string, error)) {
	subAgentFnMu.Lock()
	defer subAgentFnMu.Unlock()
	subAgentFn = fn
}

func getSubAgentFn() func(ctx context.Context, req *types.Request) (string, error) {
	subAgentFnMu.RLock()
	defer subAgentFnMu.RUnlock()
	return subAgentFn
}

func init() {
	tool.Register(&SubAgentTool{})
}

// SubAgentTool spawns an independent agent instance to handle a self-contained sub-task.
type SubAgentTool struct{}

func (t *SubAgentTool) Name() string { return "spawn_agent" }

func (t *SubAgentTool) Description() string {
	return "Spawn an independent sub-agent to handle a complex, self-contained sub-task. " +
		"The sub-agent runs its own reasoning loop with access to all tools. " +
		"Use this when a task can be fully delegated and its result summarised in text. " +
		"Returns the sub-agent's final answer."
}

func (t *SubAgentTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"task": {
				Type:        "string",
				Description: "The complete, self-contained task description for the sub-agent.",
			},
		},
		Required: []string{"task"},
	}
}

func (t *SubAgentTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	// Depth guard: prevent the sub-agent from spawning another sub-agent.
	depth, _ := ctx.Value(subAgentDepthKey{}).(int)
	if depth >= maxSubAgentDepth {
		return "", fmt.Errorf("spawn_agent: max sub-agent depth (%d) reached", maxSubAgentDepth)
	}

	fn := getSubAgentFn()
	if fn == nil {
		return "", fmt.Errorf("spawn_agent: not initialized — call tools.SetSubAgentFn in main.go")
	}

	task, _ := params["task"].(string)
	if task == "" {
		return "", fmt.Errorf("spawn_agent: 'task' parameter is required")
	}

	// Pass incremented depth in child context so nested calls are blocked.
	childCtx := context.WithValue(ctx, subAgentDepthKey{}, depth+1)

	req := &types.Request{
		SessionID: "sub-" + randomHex(8),
		Content:   task,
		Channel:   "internal",
	}

	return fn(childCtx, req)
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "000000000000000"
	}
	return hex.EncodeToString(b)
}
