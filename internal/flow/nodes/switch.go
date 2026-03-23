// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package nodes

import (
	"context"
	"fmt"

	"github.com/gopaw/gopaw/internal/flow"
)

// SwitchNodeExecutor 多路分支节点执行器
// 比条件节点更强大，支持多个分支选择
type SwitchNodeExecutor struct {
	flow.BaseNodeExecutor
}

// NewSwitchNodeExecutor 创建多路分支节点执行器
func NewSwitchNodeExecutor() *SwitchNodeExecutor {
	return &SwitchNodeExecutor{
		BaseNodeExecutor: flow.NewBaseNodeExecutor(flow.NodeTypeSwitch),
	}
}

// Execute 执行多路分支节点
func (e *SwitchNodeExecutor) Execute(ctx context.Context, node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	// 获取要匹配的变量
	variableName, _ := node.Config["variable"].(string)
	if variableName == "" {
		variableName = "input"
	}

	// 获取变量值
	var value interface{}
	if val, ok := exec.Variables[variableName]; ok {
		value = val
	} else if val, ok := exec.Context[variableName]; ok {
		value = val
	} else {
		value = exec.Input
	}

	// 获取分支配置
	cases, ok := node.Config["cases"].([]interface{})
	if !ok || len(cases) == 0 {
		return map[string]interface{}{
			"matched":    false,
			"default":    true,
			"node_id":    node.ID,
		}, nil
	}

	// 遍历分支进行匹配
	for i, c := range cases {
		caseConfig, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		caseValue := caseConfig["value"]
		caseLabel, _ := caseConfig["label"].(string)

		// 匹配值
		if e.matchValue(value, caseValue) {
			return map[string]interface{}{
				"matched":    true,
				"case_index": i,
				"case_label": caseLabel,
				"value":      value,
				"node_id":    node.ID,
			}, nil
		}
	}

	// 没有匹配，使用默认分支
	return map[string]interface{}{
		"matched":    false,
		"default":    true,
		"value":      value,
		"node_id":    node.ID,
	}, nil
}

// matchValue 匹配值
func (e *SwitchNodeExecutor) matchValue(a, b interface{}) bool {
	// 精确匹配
	if a == b {
		return true
	}

	// 字符串匹配
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	if aStr == bStr {
		return true
	}

	return false
}

// Validate 验证节点配置
func (e *SwitchNodeExecutor) Validate(node *flow.FlowNode) error {
	return e.BaseNodeExecutor.Validate(node)
}

func init() {
	flow.MustRegisterNode(NewSwitchNodeExecutor())
}