// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package nodes

import (
	"context"
	"fmt"

	"github.com/gopaw/gopaw/internal/flow"
)

// MergeNodeExecutor 分支合并节点执行器
// 用于合并多个并行分支的结果
type MergeNodeExecutor struct {
	flow.BaseNodeExecutor
}

// NewMergeNodeExecutor 创建分支合并节点执行器
func NewMergeNodeExecutor() *MergeNodeExecutor {
	return &MergeNodeExecutor{
		BaseNodeExecutor: flow.NewBaseNodeExecutor(flow.NodeTypeMerge),
	}
}

// Execute 执行分支合并节点
func (e *MergeNodeExecutor) Execute(ctx context.Context, node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	// 获取合并策略
	strategy, _ := node.Config["strategy"].(string)
	if strategy == "" {
		strategy = "all" // 默认等待所有分支
	}

	// 获取需要合并的分支数量
	expectedBranches := 2
	if val, ok := node.Config["branches"].(int); ok && val > 0 {
		expectedBranches = val
	}

	// 合并节点的实际执行逻辑在 engine 中处理
	// 这里返回基本信息
	return map[string]interface{}{
		"merge":             true,
		"strategy":          strategy,
		"expected_branches": expectedBranches,
		"node_id":           node.ID,
	}, nil
}

// Validate 验证节点配置
func (e *MergeNodeExecutor) Validate(node *flow.FlowNode) error {
	return e.BaseNodeExecutor.Validate(node)
}

// MergeResults 合并多个分支的结果
func MergeResults(results []map[string]interface{}, strategy string) map[string]interface{} {
	switch strategy {
	case "all":
		// 等待所有分支完成，合并所有结果
		merged := make(map[string]interface{})
		for i, result := range results {
			merged[fmt.Sprintf("branch_%d", i)] = result
		}
		return merged

	case "any":
		// 任一分支完成即可
		if len(results) > 0 {
			return results[0]
		}
		return nil

	case "first":
		// 只取第一个完成的结果
		if len(results) > 0 {
			return results[0]
		}
		return nil

	default:
		// 默认合并所有
		merged := make(map[string]interface{})
		for i, result := range results {
			merged[fmt.Sprintf("branch_%d", i)] = result
		}
		return merged
	}
}

func init() {
	flow.MustRegisterNode(NewMergeNodeExecutor())
}