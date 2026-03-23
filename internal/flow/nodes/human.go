// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package nodes

import (
	"context"

	"github.com/gopaw/gopaw/internal/flow"
)

// HumanNodeExecutor 人工节点执行器
type HumanNodeExecutor struct {
	flow.BaseNodeExecutor
}

// NewHumanNodeExecutor 创建人工节点执行器
func NewHumanNodeExecutor() *HumanNodeExecutor {
	return &HumanNodeExecutor{
		BaseNodeExecutor: flow.NewBaseNodeExecutor(flow.NodeTypeHuman),
	}
}

// Execute 执行人工节点
func (e *HumanNodeExecutor) Execute(ctx context.Context, node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	// 设置等待状态
	exec.Status = flow.ExecutionStatusWaiting
	exec.Context["waiting_for"] = node.ID

	// 把 prompt 和快捷选项存入 context，供前端展示
	if node.Prompt != "" {
		exec.Context["waiting_node_prompt"] = node.Prompt
	}
	if opts, ok := node.Config["options"]; ok {
		exec.Context["waiting_node_options"] = opts
	}

	return map[string]interface{}{"waiting": true, "node_id": node.ID}, nil
}

// Validate 验证节点配置
func (e *HumanNodeExecutor) Validate(node *flow.FlowNode) error {
	return e.BaseNodeExecutor.Validate(node)
}

func init() {
	flow.MustRegisterNode(NewHumanNodeExecutor())
}