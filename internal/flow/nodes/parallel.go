// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package nodes

import (
	"context"

	"github.com/gopaw/gopaw/internal/flow"
)

// ParallelNodeExecutor 并行节点执行器
type ParallelNodeExecutor struct {
	flow.BaseNodeExecutor
}

// NewParallelNodeExecutor 创建并行节点执行器
func NewParallelNodeExecutor() *ParallelNodeExecutor {
	return &ParallelNodeExecutor{
		BaseNodeExecutor: flow.NewBaseNodeExecutor(flow.NodeTypeParallel),
	}
}

// Execute 执行并行节点
// 并行节点的实际执行逻辑在 engine 中处理
func (e *ParallelNodeExecutor) Execute(ctx context.Context, node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	// 并行节点需要边信息，这里返回基本信息
	return map[string]interface{}{"parallel": true, "node_id": node.ID}, nil
}

// Validate 验证节点配置
func (e *ParallelNodeExecutor) Validate(node *flow.FlowNode) error {
	return e.BaseNodeExecutor.Validate(node)
}

func init() {
	flow.MustRegisterNode(NewParallelNodeExecutor())
}