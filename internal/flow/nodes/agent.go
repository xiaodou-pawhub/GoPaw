// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package nodes

import (
	"context"
	"fmt"

	"github.com/gopaw/gopaw/internal/agent"
	"github.com/gopaw/gopaw/internal/flow"
	"github.com/gopaw/gopaw/pkg/types"
)

// AgentNodeExecutor Agent 节点执行器
type AgentNodeExecutor struct {
	flow.BaseNodeExecutor
	agentRouter *agent.Router
}

// NewAgentNodeExecutor 创建 Agent 节点执行器
func NewAgentNodeExecutor(router *agent.Router) *AgentNodeExecutor {
	return &AgentNodeExecutor{
		BaseNodeExecutor: flow.NewBaseNodeExecutor(flow.NodeTypeAgent),
		agentRouter:      router,
	}
}

// SetAgentRouter 设置 Agent Router
func (e *AgentNodeExecutor) SetAgentRouter(router *agent.Router) {
	e.agentRouter = router
}

// Execute 执行 Agent 节点
func (e *AgentNodeExecutor) Execute(ctx context.Context, node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	if node.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required for agent node")
	}

	if e.agentRouter == nil {
		return nil, fmt.Errorf("agent router not initialized")
	}

	// 获取 Agent 实例
	agentInstance, err := e.agentRouter.GetOrCreateAgent(node.AgentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// 构建输入
	input := buildInput(node, exec)

	// 执行 Agent
	resp, err := agentInstance.Process(ctx, &types.Request{
		Content:   input,
		SessionID: exec.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	// 更新上下文
	exec.Context[node.ID+"_output"] = resp.Content

	return map[string]interface{}{"result": resp.Content}, nil
}

// Validate 验证节点配置
func (e *AgentNodeExecutor) Validate(node *flow.FlowNode) error {
	if err := e.BaseNodeExecutor.Validate(node); err != nil {
		return err
	}
	if node.AgentID == "" {
		return fmt.Errorf("agent_id is required for agent node")
	}
	return nil
}

// buildInput 构建节点输入
func buildInput(node *flow.FlowNode, exec *flow.Execution) string {
	// 如果有 prompt 模板，使用模板
	if node.Prompt != "" {
		return resolveTemplate(node.Prompt, exec.Variables, exec.Context)
	}

	// 如果有输入映射，构建输入
	if len(node.Inputs) > 0 {
		inputs := make(map[string]interface{})
		for name, expr := range node.Inputs {
			inputs[name] = resolveExpression(expr, exec.Variables, exec.Context)
		}
		return fmt.Sprintf("%v", inputs)
	}

	// 默认使用执行输入
	return exec.Input
}

// resolveTemplate 解析模板
func resolveTemplate(template string, variables, context map[string]interface{}) string {
	// 简单实现，后续可以增强
	return template
}

// resolveExpression 解析表达式
func resolveExpression(expr string, variables, context map[string]interface{}) interface{} {
	// 简单实现，后续可以增强
	if val, ok := variables[expr]; ok {
		return val
	}
	if val, ok := context[expr]; ok {
		return val
	}
	return expr
}

func init() {
	// Agent 节点需要依赖注入，不在此处自动注册
	// 使用 flow.RegisterNode(NewAgentNodeExecutor(router)) 手动注册
}