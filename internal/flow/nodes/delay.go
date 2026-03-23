// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package nodes

import (
	"context"
	"fmt"
	"time"

	"github.com/gopaw/gopaw/internal/flow"
)

// DelayNodeExecutor 延迟节点执行器
type DelayNodeExecutor struct {
	flow.BaseNodeExecutor
}

// NewDelayNodeExecutor 创建延迟节点执行器
func NewDelayNodeExecutor() *DelayNodeExecutor {
	return &DelayNodeExecutor{
		BaseNodeExecutor: flow.NewBaseNodeExecutor(flow.NodeTypeDelay),
	}
}

// Execute 执行延迟节点
func (e *DelayNodeExecutor) Execute(ctx context.Context, node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	// 获取延迟时间（秒）
	delaySeconds := 5 // 默认 5 秒
	if val, ok := node.Config["delay"].(int); ok && val > 0 {
		delaySeconds = val
	}
	if val, ok := node.Config["delay_seconds"].(int); ok && val > 0 {
		delaySeconds = val
	}

	// 最大延迟时间限制（5 分钟）
	maxDelay := 300
	if delaySeconds > maxDelay {
		delaySeconds = maxDelay
	}

	// 等待
	select {
	case <-time.After(time.Duration(delaySeconds) * time.Second):
		return map[string]interface{}{
			"delayed":    true,
			"seconds":    delaySeconds,
			"completed_at": time.Now().Format(time.RFC3339),
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Validate 验证节点配置
func (e *DelayNodeExecutor) Validate(node *flow.FlowNode) error {
	if err := e.BaseNodeExecutor.Validate(node); err != nil {
		return err
	}

	// 检查延迟时间是否合理
	if val, ok := node.Config["delay"].(int); ok && val < 0 {
		return fmt.Errorf("delay must be positive")
	}

	return nil
}

func init() {
	flow.MustRegisterNode(NewDelayNodeExecutor())
}