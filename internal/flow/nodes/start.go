// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package nodes

import (
	"context"

	"github.com/gopaw/gopaw/internal/flow"
)

// StartNodeExecutor 开始节点执行器
type StartNodeExecutor struct {
	flow.BaseNodeExecutor
}

// NewStartNodeExecutor 创建开始节点执行器
func NewStartNodeExecutor() *StartNodeExecutor {
	return &StartNodeExecutor{
		BaseNodeExecutor: flow.NewBaseNodeExecutor(flow.NodeTypeStart),
	}
}

// Execute 执行开始节点
func (e *StartNodeExecutor) Execute(ctx context.Context, node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	// 开始节点直接返回输入
	return map[string]interface{}{"input": exec.Input}, nil
}

// Validate 验证节点配置
func (e *StartNodeExecutor) Validate(node *flow.FlowNode) error {
	if err := e.BaseNodeExecutor.Validate(node); err != nil {
		return err
	}
	// 开始节点不需要额外配置
	return nil
}

func init() {
	// 自动注册
	flow.MustRegisterNode(NewStartNodeExecutor())
}