// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package flow

import (
	"context"
	"fmt"
	"sync"
)

// NodeExecutor 节点执行器接口
// 所有节点类型都需要实现此接口，以便在流程引擎中执行
type NodeExecutor interface {
	// Type 返回节点类型
	Type() NodeType

	// Execute 执行节点逻辑
	// ctx: 上下文，用于取消和超时控制
	// node: 节点定义
	// exec: 执行实例，包含变量、上下文等
	// 返回节点输出和错误
	Execute(ctx context.Context, node *FlowNode, exec *Execution) (map[string]interface{}, error)

	// Validate 验证节点配置是否有效
	Validate(node *FlowNode) error
}

// NodeRegistry 节点注册表
// 管理所有节点执行器的注册和查找
type NodeRegistry struct {
	executors map[NodeType]NodeExecutor
	mu        sync.RWMutex
}

// globalRegistry 全局节点注册表
var globalRegistry = &NodeRegistry{
	executors: make(map[NodeType]NodeExecutor),
}

// GetNodeRegistry 获取全局节点注册表
func GetNodeRegistry() *NodeRegistry {
	return globalRegistry
}

// Register 注册节点执行器
func (r *NodeRegistry) Register(executor NodeExecutor) error {
	if executor == nil {
		return fmt.Errorf("executor cannot be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	nodeType := executor.Type()
	if nodeType == "" {
		return fmt.Errorf("node type cannot be empty")
	}

	if _, exists := r.executors[nodeType]; exists {
		return fmt.Errorf("node type %s already registered", nodeType)
	}

	r.executors[nodeType] = executor
	return nil
}

// MustRegister 注册节点执行器，如果失败则 panic
func (r *NodeRegistry) MustRegister(executor NodeExecutor) {
	if err := r.Register(executor); err != nil {
		panic(err)
	}
}

// Get 获取节点执行器
func (r *NodeRegistry) Get(nodeType NodeType) (NodeExecutor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	executor, exists := r.executors[nodeType]
	return executor, exists
}

// List 列出所有已注册的节点类型
func (r *NodeRegistry) List() []NodeType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]NodeType, 0, len(r.executors))
	for t := range r.executors {
		types = append(types, t)
	}
	return types
}

// RegisterNode 全局注册节点执行器的便捷方法
func RegisterNode(executor NodeExecutor) error {
	return globalRegistry.Register(executor)
}

// MustRegisterNode 全局注册节点执行器，失败则 panic
func MustRegisterNode(executor NodeExecutor) {
	globalRegistry.MustRegister(executor)
}

// GetNodeExecutor 获取节点执行器
func GetNodeExecutor(nodeType NodeType) (NodeExecutor, bool) {
	return globalRegistry.Get(nodeType)
}

// BaseNodeExecutor 基础节点执行器
// 提供默认实现，方便其他节点继承
type BaseNodeExecutor struct {
	nodeType NodeType
}

// NewBaseNodeExecutor 创建基础节点执行器
func NewBaseNodeExecutor(nodeType NodeType) BaseNodeExecutor {
	return BaseNodeExecutor{nodeType: nodeType}
}

// Type 返回节点类型
func (e *BaseNodeExecutor) Type() NodeType {
	return e.nodeType
}

// Validate 默认验证实现
func (e *BaseNodeExecutor) Validate(node *FlowNode) error {
	if node.ID == "" {
		return fmt.Errorf("node id is required")
	}
	if node.Type != e.nodeType {
		return fmt.Errorf("node type mismatch: expected %s, got %s", e.nodeType, node.Type)
	}
	return nil
}

// NodeExecutionContext 节点执行上下文
// 包含执行过程中需要的所有依赖
type NodeExecutionContext struct {
	Engine     *Engine
	AgentMgr   interface{} // *agent.Manager
	MsgMgr     interface{} // *message.Manager
	TraceSvc   *TraceService
	Logger     interface{} // *zap.Logger
}

// NodeResult 节点执行结果
type NodeResult struct {
	Output    map[string]interface{}
	Status    ExecutionStatus
	WaitFor   string // 如果需要等待，这里是等待的节点 ID
	Error     error
	Metadata  map[string]interface{} // 额外元数据
}