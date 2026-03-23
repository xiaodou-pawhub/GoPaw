// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package nodes

import (
	"context"
	"fmt"

	"github.com/gopaw/gopaw/internal/flow"
)

// WebhookNodeExecutor Webhook 节点执行器
type WebhookNodeExecutor struct {
	flow.BaseNodeExecutor
}

// NewWebhookNodeExecutor 创建 Webhook 节点执行器
func NewWebhookNodeExecutor() *WebhookNodeExecutor {
	return &WebhookNodeExecutor{
		BaseNodeExecutor: flow.NewBaseNodeExecutor(flow.NodeTypeWebhook),
	}
}

// Execute 执行 Webhook 节点
func (e *WebhookNodeExecutor) Execute(ctx context.Context, node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	// 设置等待状态
	exec.Status = flow.ExecutionStatusWaiting
	exec.Context["waiting_for"] = node.ID
	exec.Context["waiting_type"] = "webhook"

	// 生成 webhook ID
	webhookID := fmt.Sprintf("%s_%s", exec.FlowID, node.ID)
	exec.Context["webhook_id"] = webhookID

	// 获取超时配置
	timeout := 3600 // 默认 1 小时
	if val, ok := node.Config["timeout"].(int); ok && val > 0 {
		timeout = val
	}

	return map[string]interface{}{
		"waiting":    true,
		"node_id":    node.ID,
		"webhook_id": webhookID,
		"timeout":    timeout,
	}, nil
}

// Validate 验证节点配置
func (e *WebhookNodeExecutor) Validate(node *flow.FlowNode) error {
	return e.BaseNodeExecutor.Validate(node)
}

func init() {
	flow.MustRegisterNode(NewWebhookNodeExecutor())
}