// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package nodes

import (
	"context"
	"fmt"

	"github.com/gopaw/gopaw/internal/flow"
)

// EndNodeExecutor 结束节点执行器
type EndNodeExecutor struct {
	flow.BaseNodeExecutor
}

// NewEndNodeExecutor 创建结束节点执行器
func NewEndNodeExecutor() *EndNodeExecutor {
	return &EndNodeExecutor{
		BaseNodeExecutor: flow.NewBaseNodeExecutor(flow.NodeTypeEnd),
	}
}

// Execute 执行结束节点
func (e *EndNodeExecutor) Execute(ctx context.Context, node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	// 结束节点，返回最终输出
	output := buildOutput(node, exec)
	exec.Output = output
	return map[string]interface{}{"output": output}, nil
}

// Validate 验证节点配置
func (e *EndNodeExecutor) Validate(node *flow.FlowNode) error {
	return e.BaseNodeExecutor.Validate(node)
}

// buildOutput 构建输出
func buildOutput(node *flow.FlowNode, exec *flow.Execution) string {
	// 如果有输出映射，构建输出
	if len(node.Outputs) > 0 {
		outputs := make(map[string]interface{})
		for name, varName := range node.Outputs {
			if val, ok := exec.Context[varName]; ok {
				outputs[name] = val
			} else if val, ok := exec.Variables[varName]; ok {
				outputs[name] = val
			}
		}
		return fmt.Sprintf("%v", outputs)
	}

	// 默认返回最后一条消息
	if lastOutput, ok := exec.Context["last_output"].(string); ok {
		return lastOutput
	}

	return exec.Input
}

func init() {
	flow.MustRegisterNode(NewEndNodeExecutor())
}