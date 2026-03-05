package builtin

import (
	"context"
	"fmt"
	"sync"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
)

var (
	subAgentFnMu sync.RWMutex
	subAgentFn   func(ctx context.Context, req *types.Request) (string, error)
)

// SetSubAgentFn 注入子代理执行函数。
func SetSubAgentFn(fn func(ctx context.Context, req *types.Request) (string, error)) {
	subAgentFnMu.Lock()
	defer subAgentFnMu.Unlock()
	subAgentFn = fn
}

func init() {
	tool.Register(&SubAgentTool{})
}

// SubAgentTool 允许 Agent 启动一个新的子任务。
type SubAgentTool struct {
	channel string
	session string
	user    string
}

func (t *SubAgentTool) Name() string { return "sub_agent" }

func (t *SubAgentTool) Description() string {
	return "Delegate a complex sub-task to a specialized sub-agent. " +
		"Returns the final answer from the sub-agent. Use this to break down large problems."
}

func (t *SubAgentTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"task": {
				Type:        "string",
				Description: "The specific instruction or question for the sub-agent.",
			},
		},
		Required: []string{"task"},
	}
}

func (t *SubAgentTool) SetContext(channel, session, user string) {
	t.channel = channel
	t.session = session
	t.user = user
}

func (t *SubAgentTool) Execute(ctx context.Context, params map[string]any) *plugin.ToolResult {
	subAgentFnMu.RLock()
	fn := subAgentFn
	subAgentFnMu.RUnlock()

	if fn == nil {
		return plugin.ErrorResult("sub-agent system not initialized")
	}

	task, _ := params["task"].(string)
	if task == "" {
		return plugin.ErrorResult("'task' is required")
	}

	// Create a sub-request
	req := &types.Request{
		SessionID: t.session + ":sub", // Simple nesting
		UserID:    t.user,
		Channel:   t.channel,
		Content:   task,
		MsgType:   types.MsgTypeText,
	}

	result, err := fn(ctx, req)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("sub-agent failed: %v", err))
	}

	return plugin.NewToolResult(result)
}
