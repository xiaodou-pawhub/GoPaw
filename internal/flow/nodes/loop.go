// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package nodes

import (
	"context"

	"github.com/gopaw/gopaw/internal/flow"
)

// LoopNodeExecutor 循环节点执行器
type LoopNodeExecutor struct {
	flow.BaseNodeExecutor
}

// NewLoopNodeExecutor 创建循环节点执行器
func NewLoopNodeExecutor() *LoopNodeExecutor {
	return &LoopNodeExecutor{
		BaseNodeExecutor: flow.NewBaseNodeExecutor(flow.NodeTypeLoop),
	}
}

// Execute 执行循环节点
// 循环节点的实际执行逻辑在 engine 中处理
func (e *LoopNodeExecutor) Execute(ctx context.Context, node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	// 循环节点需要边信息，这里返回基本信息
	maxLoop := 10 // 默认最大循环次数
	if val, ok := node.Config["max_loop"].(int); ok && val > 0 {
		maxLoop = val
	}

	return map[string]interface{}{
		"loop":      true,
		"node_id":   node.ID,
		"max_loop":  maxLoop,
	}, nil
}

// Validate 验证节点配置
func (e *LoopNodeExecutor) Validate(node *flow.FlowNode) error {
	return e.BaseNodeExecutor.Validate(node)
}

func init() {
	flow.MustRegisterNode(NewLoopNodeExecutor())
}