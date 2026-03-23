// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package nodes

import (
	"context"

	"github.com/gopaw/gopaw/internal/flow"
)

// ConditionNodeExecutor 条件节点执行器
type ConditionNodeExecutor struct {
	flow.BaseNodeExecutor
}

// NewConditionNodeExecutor 创建条件节点执行器
func NewConditionNodeExecutor() *ConditionNodeExecutor {
	return &ConditionNodeExecutor{
		BaseNodeExecutor: flow.NewBaseNodeExecutor(flow.NodeTypeCondition),
	}
}

// Execute 执行条件节点
// 条件节点评估所有出边的条件，返回匹配的分支
func (e *ConditionNodeExecutor) Execute(ctx context.Context, node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	// 条件节点需要边信息，这里返回基本信息
	// 实际的条件评估在 engine 中处理
	return map[string]interface{}{"evaluated": true, "node_id": node.ID}, nil
}

// Validate 验证节点配置
func (e *ConditionNodeExecutor) Validate(node *flow.FlowNode) error {
	return e.BaseNodeExecutor.Validate(node)
}

// EvaluateCondition 评估条件
func EvaluateCondition(condition *flow.EdgeCondition, exec *flow.Execution) bool {
	if condition == nil || condition.Type == "" || condition.Type == "always" {
		return true
	}

	switch condition.Type {
	case "expression":
		return evaluateExpression(condition.Expression, exec)
	case "intent":
		return evaluateIntent(condition.Intent, exec)
	case "llm":
		// LLM 条件评估需要额外处理
		return false
	default:
		return false
	}
}

// evaluateExpression 评估表达式
func evaluateExpression(expr string, exec *flow.Execution) bool {
	// 简单实现，后续可以增强
	// 支持: ${变量} == "值", ${变量} > 10 等
	return false
}

// evaluateIntent 评估意图
func evaluateIntent(intent string, exec *flow.Execution) bool {
	// 检查上下文中的意图
	if detectedIntent, ok := exec.Context["detected_intent"].(string); ok {
		return detectedIntent == intent
	}
	return false
}

func init() {
	flow.MustRegisterNode(NewConditionNodeExecutor())
}