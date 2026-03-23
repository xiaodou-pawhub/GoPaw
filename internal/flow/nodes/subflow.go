// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package nodes

import (
	"context"
	"fmt"

	"github.com/gopaw/gopaw/internal/flow"
)

// SubFlowNodeExecutor 子流程节点执行器
type SubFlowNodeExecutor struct {
	flow.BaseNodeExecutor
	flowService *flow.Service
}

// NewSubFlowNodeExecutor 创建子流程节点执行器
func NewSubFlowNodeExecutor(svc *flow.Service) *SubFlowNodeExecutor {
	return &SubFlowNodeExecutor{
		BaseNodeExecutor: flow.NewBaseNodeExecutor(flow.NodeTypeSubFlow),
		flowService:      svc,
	}
}

// SetFlowService 设置流程服务
func (e *SubFlowNodeExecutor) SetFlowService(svc *flow.Service) {
	e.flowService = svc
}

// Execute 执行子流程节点
func (e *SubFlowNodeExecutor) Execute(ctx context.Context, node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	// 子流程 ID
	subFlowID, ok := node.Config["flow_id"].(string)
	if !ok || subFlowID == "" {
		return nil, fmt.Errorf("flow_id is required for subflow node")
	}

	// 子流程的实际执行在 engine 中处理
	return map[string]interface{}{
		"subflow":   true,
		"node_id":   node.ID,
		"flow_id":   subFlowID,
	}, nil
}

// Validate 验证节点配置
func (e *SubFlowNodeExecutor) Validate(node *flow.FlowNode) error {
	if err := e.BaseNodeExecutor.Validate(node); err != nil {
		return err
	}
	if flowID, ok := node.Config["flow_id"].(string); !ok || flowID == "" {
		return fmt.Errorf("flow_id is required for subflow node")
	}
	return nil
}

func init() {
	// 子流程节点需要依赖注入，不在此处自动注册
}