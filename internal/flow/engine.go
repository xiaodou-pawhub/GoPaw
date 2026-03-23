// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package flow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gopaw/gopaw/internal/agent"
	"github.com/gopaw/gopaw/internal/agent/message"
	"github.com/gopaw/gopaw/pkg/types"
	"go.uber.org/zap"
)

// Engine 流程执行引擎
type Engine struct {
	db           *sql.DB
	agentMgr     *agent.Manager
	agentRouter  *agent.Router
	msgMgr       *message.Manager
	wsHub        *WebSocketHub
	traceService *TraceService
	logger       *zap.Logger
	running      map[string]*Execution // 正在执行的实例
	mu           sync.RWMutex
}

// NewEngine 创建执行引擎
func NewEngine(db *sql.DB, agentMgr *agent.Manager, msgMgr *message.Manager, logger *zap.Logger) (*Engine, error) {
	// 初始化追踪表
	if err := InitTraceSchema(db); err != nil {
		return nil, fmt.Errorf("failed to init trace schema: %w", err)
	}

	traceService := NewTraceService(db, logger)

	return &Engine{
		db:           db,
		agentMgr:     agentMgr,
		msgMgr:       msgMgr,
		traceService: traceService,
		logger:       logger.Named("flow_engine"),
		running:      make(map[string]*Execution),
	}, nil
}

// GetTraceService 获取追踪服务
func (e *Engine) GetTraceService() *TraceService {
	return e.traceService
}

// SetWebSocketHub 设置 WebSocket Hub
func (e *Engine) SetWebSocketHub(hub *WebSocketHub) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.wsHub = hub
}

// emitEvent 发送执行事件
func (e *Engine) emitEvent(event ExecutionEvent) {
	e.mu.RLock()
	hub := e.wsHub
	e.mu.RUnlock()

	if hub != nil {
		event.Timestamp = time.Now().UnixNano()
		hub.BroadcastEvent(event)
	}
}

// RestoreWaitingExecutions 恢复等待中的执行实例
// 服务启动时调用，从数据库加载所有 waiting 状态的执行实例
func (e *Engine) RestoreWaitingExecutions() error {
	executions, err := e.ListExecutionsByStatus(ExecutionStatusWaiting, 1000)
	if err != nil {
		return fmt.Errorf("failed to load waiting executions: %w", err)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	for _, exec := range executions {
		e.running[exec.ID] = exec
		e.logger.Info("restored waiting execution",
			zap.String("execution_id", exec.ID),
			zap.String("flow_id", exec.FlowID),
			zap.String("waiting_for", exec.Context["waiting_for"].(string)),
		)
	}

	if len(executions) > 0 {
		e.logger.Info("restored waiting executions", zap.Int("count", len(executions)))
	}

	return nil
}

// SetAgentRouter 设置 Agent Router（用于延迟注入）
func (e *Engine) SetAgentRouter(router *agent.Router) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.agentRouter = router
}

// Execute 执行流程
func (e *Engine) Execute(flow *Flow, req ExecuteRequest) (*ExecuteResponse, error) {
	// 创建执行实例
	exec := &Execution{
		ID:          generateID(),
		FlowID:      flow.ID,
		Status:      ExecutionStatusRunning,
		Trigger:     req.Trigger,
		Input:       req.Input,
		Variables:   req.Variables,
		Context:     req.Context,
		History:     []ExecutionStep{},
		StartedAt:   time.Now(),
		DebugMode:   req.DebugMode,
		Breakpoints: req.Breakpoints,
		StepMode:    req.StepMode,
	}

	if exec.Trigger == "" {
		exec.Trigger = "manual"
	}
	if exec.Variables == nil {
		exec.Variables = make(map[string]interface{})
	}
	if exec.Context == nil {
		exec.Context = make(map[string]interface{})
	}

	// 设置初始上下文
	exec.Context["input"] = req.Input
	exec.CurrentNode = flow.Definition.StartNodeID

	// 保存执行记录
	if err := e.saveExecution(exec); err != nil {
		return nil, fmt.Errorf("failed to save execution: %w", err)
	}

	// 加入运行中的实例
	e.mu.Lock()
	e.running[exec.ID] = exec
	e.mu.Unlock()

	// 开始执行
	go e.runFlow(flow, exec)

	return &ExecuteResponse{
		ExecutionID: exec.ID,
		Status:      string(exec.Status),
	}, nil
}

// Step 单步执行（调试模式）
func (e *Engine) Step(executionID string) (*ExecuteResponse, error) {
	e.mu.RLock()
	exec, ok := e.running[executionID]
	e.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("execution not found: %s", executionID)
	}

	if !exec.DebugMode {
		return nil, fmt.Errorf("execution is not in debug mode")
	}

	if exec.Status != ExecutionStatusWaiting {
		return nil, fmt.Errorf("execution is not waiting for step")
	}

	// 清除单步等待状态
	exec.Context["__step_waiting__"] = false
	exec.Status = ExecutionStatusRunning

	// 获取流程定义
	flow, err := e.getFlow(exec.FlowID)
	if err != nil {
		return nil, err
	}

	// 继续执行
	go e.runFlow(flow, exec)

	return &ExecuteResponse{
		ExecutionID: exec.ID,
		Status:      string(exec.Status),
	}, nil
}

// SetBreakpoints 设置断点
func (e *Engine) SetBreakpoints(executionID string, breakpoints []string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	exec, ok := e.running[executionID]
	if !ok {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	exec.Breakpoints = breakpoints
	return nil
}

// RetryFromNode 从特定节点重试执行
func (e *Engine) RetryFromNode(executionID string, nodeID string) (*ExecuteResponse, error) {
	e.mu.Lock()
	exec, ok := e.running[executionID]
	if !ok {
		e.mu.Unlock()
		// 从数据库加载
		var err error
		exec, err = e.GetExecution(executionID)
		if err != nil {
			return nil, fmt.Errorf("execution not found: %s", executionID)
		}
		e.mu.Lock()
		e.running[executionID] = exec
	}
	e.mu.Unlock()

	// 检查执行状态
	if exec.Status != ExecutionStatusFailed {
		return nil, fmt.Errorf("execution is not failed: %s", exec.Status)
	}

	// 获取流程定义
	flow, err := e.getFlow(exec.FlowID)
	if err != nil {
		return nil, err
	}

	// 验证节点存在
	nodeExists := false
	for _, n := range flow.Definition.Nodes {
		if n.ID == nodeID {
			nodeExists = true
			break
		}
	}
	if !nodeExists {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	// 重置执行状态
	exec.Status = ExecutionStatusRunning
	exec.CurrentNode = nodeID
	exec.Error = ""

	// 清除该节点之后的执行历史
	newHistory := make([]ExecutionStep, 0)
	for _, step := range exec.History {
		newHistory = append(newHistory, step)
		if step.NodeID == nodeID {
			break
		}
	}
	exec.History = newHistory

	// 保存状态
	e.saveExecution(exec)

	// 继续执行
	go e.runFlow(flow, exec)

	return &ExecuteResponse{
		ExecutionID: exec.ID,
		Status:      string(exec.Status),
	}, nil
}

// Continue 继续执行（人工节点后）
func (e *Engine) Continue(executionID string, input string) (*ExecuteResponse, error) {
	e.mu.RLock()
	exec, ok := e.running[executionID]
	e.mu.RUnlock()

	if !ok {
		// 从数据库加载
		var err error
		exec, err = e.GetExecution(executionID)
		if err != nil {
			return nil, fmt.Errorf("execution not found: %s", executionID)
		}
	}

	if exec.Status != ExecutionStatusWaiting {
		return nil, fmt.Errorf("execution is not waiting: %s", exec.Status)
	}

	// 更新上下文
	exec.Context["human_input"] = input
	exec.Status = ExecutionStatusRunning

	// 获取流程定义
	flow, err := e.getFlow(exec.FlowID)
	if err != nil {
		return nil, err
	}

	// 继续执行
	go e.runFlow(flow, exec)

	return &ExecuteResponse{
		ExecutionID: exec.ID,
		Status:      string(exec.Status),
	}, nil
}

// runFlow 执行流程
func (e *Engine) runFlow(flow *Flow, exec *Execution) {
	// 开始追踪
	trace, err := e.traceService.StartTrace(
		flow.ID,
		flow.Name,
		exec.ID,
		exec.Trigger,
		TraceMetadata{
			Input:     exec.Input,
			Variables: exec.Variables,
		},
	)
	if err != nil {
		e.logger.Warn("failed to start trace", zap.Error(err))
	}

	defer func() {
		if r := recover(); r != nil {
			e.logger.Error("flow panic", zap.Any("error", r))
			exec.Status = ExecutionStatusFailed
			exec.Error = fmt.Sprintf("panic: %v", r)
			now := time.Now()
			exec.CompletedAt = &now
			e.saveExecution(exec)
			e.emitEvent(ExecutionEvent{
				Type:        "failed",
				ExecutionID: exec.ID,
				FlowID:      flow.ID,
				Status:      "failed",
				Error:       exec.Error,
			})
			// 结束追踪
			if trace != nil {
				e.traceService.EndTrace(trace.ID, "", fmt.Errorf("panic: %v", r))
			}
		}

		e.mu.Lock()
		delete(e.running, exec.ID)
		e.mu.Unlock()
	}()

	// 发送开始事件
	e.emitEvent(ExecutionEvent{
		Type:        "started",
		ExecutionID: exec.ID,
		FlowID:      flow.ID,
		Status:      "running",
	})

	// 构建节点映射
	nodeMap := make(map[string]FlowNode)
	for _, node := range flow.Definition.Nodes {
		nodeMap[node.ID] = node
	}

	// 构建边映射（source -> edges）
	edgeMap := make(map[string][]FlowEdge)
	for _, edge := range flow.Definition.Edges {
		edgeMap[edge.Source] = append(edgeMap[edge.Source], edge)
	}

	// 循环状态跟踪
	loopStack := make([]LoopState, 0) // 循环栈，支持嵌套循环

	// 当前 Span ID（用于追踪）
	var currentSpan *Span

	// 执行循环
	for exec.Status == ExecutionStatusRunning {
		node, ok := nodeMap[exec.CurrentNode]
		if !ok {
			e.logger.Error("node not found", zap.String("node_id", exec.CurrentNode))
			exec.Status = ExecutionStatusFailed
			exec.Error = fmt.Sprintf("node not found: %s", exec.CurrentNode)
			break
		}

		// 调试模式：检查断点
		if exec.DebugMode {
			shouldBreak := false

			// 检查是否命中断点
			for _, bp := range exec.Breakpoints {
				if bp == node.ID {
					shouldBreak = true
					break
				}
			}

			// 单步模式：每步都暂停
			if exec.StepMode {
				shouldBreak = true
			}

			if shouldBreak {
				// 发送断点事件
				e.emitEvent(ExecutionEvent{
					Type:        "breakpoint",
					ExecutionID: exec.ID,
					FlowID:      flow.ID,
					NodeID:      node.ID,
					NodeName:    node.Name,
					Status:      "waiting",
				})

				// 设置等待状态
				exec.Status = ExecutionStatusWaiting
				exec.Context["__step_waiting__"] = true
				e.saveExecution(exec)
				break
			}
		}

		// 开始 Span 追踪
		if trace != nil {
			currentSpan = e.traceService.StartSpan(trace.ID, "", node.ID, node.Name, string(node.Type))
		}

		// 发送节点开始事件
		e.emitEvent(ExecutionEvent{
			Type:        "node_started",
			ExecutionID: exec.ID,
			FlowID:      flow.ID,
			NodeID:      node.ID,
			NodeName:    node.Name,
			Status:      "running",
		})

		// 添加节点开始事件到追踪
		if currentSpan != nil {
			e.traceService.AddEvent(currentSpan.ID, "node_started", EventTypeNodeStart, map[string]interface{}{
				"node_id":   node.ID,
				"node_name": node.Name,
				"node_type": node.Type,
			})
		}

		// 记录步骤开始
		step := ExecutionStep{
			NodeID:    node.ID,
			NodeType:  node.Type,
			Status:    ExecutionStatusRunning,
			StartedAt: time.Now(),
		}

		// 执行节点（带重试）
		var output map[string]interface{}
		var err error
		maxRetries := 0
		retryDelay := 1000 // 默认 1 秒
		retryOn := "error"
		fallbackNode := ""

		if node.RetryConfig != nil {
			if node.RetryConfig.MaxRetries > 0 {
				maxRetries = node.RetryConfig.MaxRetries
			}
			if node.RetryConfig.RetryDelay > 0 {
				retryDelay = node.RetryConfig.RetryDelay
			}
			if node.RetryConfig.RetryOn != "" {
				retryOn = node.RetryConfig.RetryOn
			}
			fallbackNode = node.RetryConfig.FallbackNode
		}

		for attempt := 0; attempt <= maxRetries; attempt++ {
			output, err = e.executeNode(flow, &node, exec, nodeMap, edgeMap)
			if err == nil {
				break // 成功，退出重试循环
			}

			// 检查是否需要重试
			shouldRetry := false
			switch retryOn {
			case "always":
				shouldRetry = attempt < maxRetries
			case "error":
				shouldRetry = attempt < maxRetries
			case "timeout":
				// 检查是否是超时错误
				if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded") {
					shouldRetry = attempt < maxRetries
				}
			}

			if shouldRetry {
				e.logger.Info("retrying node",
					zap.String("node_id", node.ID),
					zap.Int("attempt", attempt+1),
					zap.Int("max_retries", maxRetries),
					zap.Error(err))

				// 发送重试事件
				e.emitEvent(ExecutionEvent{
					Type:        "node_retry",
					ExecutionID: exec.ID,
					FlowID:      flow.ID,
					NodeID:      node.ID,
					NodeName:    node.Name,
					Status:      "retrying",
					Error:       err.Error(),
				})

				time.Sleep(time.Duration(retryDelay) * time.Millisecond)
			} else {
				break // 不重试，退出循环
			}
		}

		if err != nil {
			// 检查是否有 fallback 节点
			if fallbackNode != "" {
				e.logger.Info("using fallback node",
					zap.String("node_id", node.ID),
					zap.String("fallback_node", fallbackNode))

				// 发送 fallback 事件
				e.emitEvent(ExecutionEvent{
					Type:        "node_fallback",
					ExecutionID: exec.ID,
					FlowID:      flow.ID,
					NodeID:      node.ID,
					NodeName:    node.Name,
					Status:      "fallback",
					Error:       err.Error(),
				})

				// 添加 fallback 事件到追踪
				if currentSpan != nil {
					e.traceService.AddEvent(currentSpan.ID, "node_fallback", EventTypeNodeRetry, map[string]interface{}{
						"fallback_node": fallbackNode,
						"error":         err.Error(),
					})
					currentSpan.Tags.IsFallback = true
				}

				// 跳转到 fallback 节点
				exec.CurrentNode = fallbackNode
				e.saveExecution(exec)
				continue
			}

			step.Status = ExecutionStatusFailed
			step.Error = err.Error()
			exec.Status = ExecutionStatusFailed
			exec.Error = err.Error()

			// 发送节点失败事件
			e.emitEvent(ExecutionEvent{
				Type:        "node_failed",
				ExecutionID: exec.ID,
				FlowID:      flow.ID,
				NodeID:      node.ID,
				NodeName:    node.Name,
				Status:      "failed",
				Error:       err.Error(),
			})

			// 结束 Span 追踪（失败）
			if currentSpan != nil {
				e.traceService.AddEvent(currentSpan.ID, "node_failed", EventTypeNodeFail, map[string]interface{}{
					"error": err.Error(),
				})
				outputJSON, _ := json.Marshal(output)
				e.traceService.EndSpan(currentSpan.ID, string(outputJSON), err)
			}
		} else {
			step.Status = ExecutionStatusCompleted
			step.Output = output

			// 发送节点完成事件
			e.emitEvent(ExecutionEvent{
				Type:        "node_completed",
				ExecutionID: exec.ID,
				FlowID:      flow.ID,
				NodeID:      node.ID,
				NodeName:    node.Name,
				Status:      "completed",
				Output:      output,
			})

			// 结束 Span 追踪（成功）
			if currentSpan != nil {
				e.traceService.AddEvent(currentSpan.ID, "node_completed", EventTypeNodeComplete, map[string]interface{}{
					"output": output,
				})
				outputJSON, _ := json.Marshal(output)
				e.traceService.EndSpan(currentSpan.ID, string(outputJSON), nil)
			}
		}

		now := time.Now()
		step.EndedAt = &now
		exec.History = append(exec.History, step)

		// 检查是否结束
		if exec.Status != ExecutionStatusRunning || node.Type == NodeTypeEnd {
			exec.Status = ExecutionStatusCompleted
			exec.CompletedAt = &now

			// 发送完成事件
			e.emitEvent(ExecutionEvent{
				Type:        "completed",
				ExecutionID: exec.ID,
				FlowID:      flow.ID,
				Status:      "completed",
			})

			// 结束追踪
			if trace != nil {
				e.traceService.EndTrace(trace.ID, exec.Output, nil)
			}
			break
		}

		// 检查是否等待人工或 webhook
		if exec.Status == ExecutionStatusWaiting {
			break
		}

		// 处理循环节点
		if node.Type == NodeTypeLoop {
			// 检查是否应该继续循环
			shouldLoop := false
			if output != nil {
				if loop, ok := output["loop"].(bool); ok && loop {
					shouldLoop = true
				}
			}

			if shouldLoop {
				// 继续循环：跳转到循环体入口
				loopBodyEntry := ""
				if entry, ok := output["loop_body_entry"].(string); ok {
					loopBodyEntry = entry
				}

				if loopBodyEntry != "" {
					// 记录循环状态
					loopState := LoopState{
						LoopNodeID:    node.ID,
						LoopExitNode:  output["loop_exit"].(string),
						Iteration:     output["iteration"].(int),
						BodyEntryNode: loopBodyEntry,
					}
					loopStack = append(loopStack, loopState)

					exec.CurrentNode = loopBodyEntry
					e.saveExecution(exec)
					continue
				}
			} else {
				// 退出循环：跳转到循环出口
				loopExit := ""
				if exit, ok := output["loop_exit"].(string); ok {
					loopExit = exit
				}

				if loopExit != "" {
					exec.CurrentNode = loopExit
					e.saveExecution(exec)
					continue
				}
			}
		}

		// 检查是否到达循环出口（需要回到循环节点）
		if len(loopStack) > 0 {
			currentLoop := loopStack[len(loopStack)-1]
			edges := edgeMap[node.ID]

			// 检查当前节点是否有边指向循环出口
			shouldReturnToLoop := true
			for _, edge := range edges {
				if edge.Label == "exit" || edge.Label == "false" {
					// 有明确的退出边，跳出循环
					shouldReturnToLoop = false
					break
				}
			}

			// 如果当前节点是循环出口节点，回到循环节点
			if node.ID == currentLoop.LoopExitNode {
				// 弹出循环栈
				loopStack = loopStack[:len(loopStack)-1]
				// 继续执行循环出口之后的节点
			} else if shouldReturnToLoop && len(edges) > 0 {
				// 检查是否应该回到循环起点
				allEdgesToExit := true
				for _, edge := range edges {
					if edge.Target != currentLoop.LoopExitNode {
						allEdgesToExit = false
						break
					}
				}

				if !allEdgesToExit {
					// 有边不指向出口，正常执行
				} else if len(edges) > 0 {
					// 所有边都指向出口，回到循环节点继续循环
					exec.CurrentNode = currentLoop.LoopNodeID
					e.saveExecution(exec)
					continue
				}
			}
		}

		// 检查是否是并行节点（并行执行已在 executeNode 中处理）
		if node.Type == NodeTypeParallel {
			// 并行执行已完成，继续执行合并后的节点
			// Parallel 节点的输出包含 next_node 字段
			if output != nil {
				if nextNode, ok := output["next_node"].(string); ok && nextNode != "" {
					exec.CurrentNode = nextNode
					e.saveExecution(exec)
					continue
				}
			}
		}

		// 获取下一个节点
		edges := edgeMap[node.ID]
		if len(edges) == 0 {
			// 没有出边，检查是否在循环中
			if len(loopStack) > 0 {
				// 回到循环节点
				currentLoop := loopStack[len(loopStack)-1]
				exec.CurrentNode = currentLoop.LoopNodeID
				e.saveExecution(exec)
				continue
			}

			// 不在循环中，结束
			exec.Status = ExecutionStatusCompleted
			exec.CompletedAt = &now

			// 结束追踪
			if trace != nil {
				e.traceService.EndTrace(trace.ID, exec.Output, nil)
			}
			break
		}

		// 选择下一条边
		nextNodeID := e.selectNextNode(edges, exec)
		if nextNodeID == "" {
			exec.Status = ExecutionStatusFailed
			exec.Error = "no valid next node"

			// 结束追踪（失败）
			if trace != nil {
				e.traceService.EndTrace(trace.ID, "", fmt.Errorf("no valid next node"))
			}
			break
		}
		exec.CurrentNode = nextNodeID

		// 保存状态
		e.saveExecution(exec)
	}

	// 最终保存
	e.saveExecution(exec)

	// 如果流程失败，确保追踪结束
	if exec.Status == ExecutionStatusFailed && trace != nil {
		e.traceService.EndTrace(trace.ID, "", fmt.Errorf(exec.Error))
	}
}

// LoopState 循环状态
type LoopState struct {
	LoopNodeID    string // 循环节点 ID
	LoopExitNode  string // 循环出口节点 ID
	Iteration     int    // 当前迭代次数
	BodyEntryNode string // 循环体入口节点 ID
}

// executeNode 执行单个节点
func (e *Engine) executeNode(flow *Flow, node *FlowNode, exec *Execution, nodeMap map[string]FlowNode, edgeMap map[string][]FlowEdge) (map[string]interface{}, error) {
	e.logger.Info("executing node",
		zap.String("flow", flow.ID),
		zap.String("node", node.ID),
		zap.String("type", string(node.Type)),
	)

	switch node.Type {
	case NodeTypeStart:
		// 开始节点，直接返回输入
		return map[string]interface{}{"input": exec.Input}, nil

	case NodeTypeAgent:
		return e.executeAgentNode(node, exec)

	case NodeTypeHuman:
		return e.executeHumanNode(node, exec)

	case NodeTypeCondition:
		// 条件节点：评估条件并返回结果
		return e.executeConditionNode(node, exec, edgeMap)

	case NodeTypeParallel:
		// 并行节点：并行执行所有分支
		return e.executeParallelNode(flow, node, exec, nodeMap, edgeMap)

	case NodeTypeLoop:
		return e.executeLoopNode(node, exec, edgeMap)

	case NodeTypeSubFlow:
		return e.executeSubFlowNode(node, exec)

	case NodeTypeWebhook:
		return e.executeWebhookNode(node, exec)

	case NodeTypeEnd:
		// 结束节点，返回最终输出
		output := e.buildOutput(node, exec)
		exec.Output = output
		return map[string]interface{}{"output": output}, nil

	default:
		return nil, fmt.Errorf("unknown node type: %s", node.Type)
	}
}

// executeAgentNode 执行 Agent 节点
func (e *Engine) executeAgentNode(node *FlowNode, exec *Execution) (map[string]interface{}, error) {
	if node.AgentID == "" {
		return nil, fmt.Errorf("agent_id is required for agent node")
	}

	// 获取 Agent 实例
	e.mu.RLock()
	router := e.agentRouter
	e.mu.RUnlock()

	if router == nil {
		return nil, fmt.Errorf("agent router not initialized")
	}

	agentInstance, err := router.GetOrCreateAgent(node.AgentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// 构建输入
	input := e.buildInput(node, exec)

	// 执行 Agent
	ctx := context.Background()
	resp, err := agentInstance.Process(ctx, &types.Request{
		Content:   input,
		SessionID: exec.ID, // 使用执行 ID 作为会话 ID
	})
	if err != nil {
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	// 更新上下文
	exec.Context[node.ID+"_output"] = resp.Content

	return map[string]interface{}{"result": resp.Content}, nil
}

// executeHumanNode 执行人工节点
func (e *Engine) executeHumanNode(node *FlowNode, exec *Execution) (map[string]interface{}, error) {
	// 设置等待状态
	exec.Status = ExecutionStatusWaiting
	exec.Context["waiting_for"] = node.ID

	// 把 prompt 和快捷选项存入 context，供前端展示
	if node.Prompt != "" {
		exec.Context["waiting_node_prompt"] = node.Prompt
	}
	if opts, ok := node.Config["options"]; ok {
		exec.Context["waiting_node_options"] = opts
	}

	// 保存状态
	e.saveExecution(exec)

	return map[string]interface{}{"waiting": true, "node_id": node.ID}, nil
}

// executeConditionNode 执行条件节点
// 条件节点评估所有出边的条件，返回匹配的分支
func (e *Engine) executeConditionNode(node *FlowNode, exec *Execution, edgeMap map[string][]FlowEdge) (map[string]interface{}, error) {
	edges := edgeMap[node.ID]
	if len(edges) == 0 {
		return map[string]interface{}{"evaluated": true, "branch": "default"}, nil
	}

	// 评估每条边的条件
	for _, edge := range edges {
		if edge.Condition == nil || edge.Condition.Type == "" || edge.Condition.Type == "always" {
			// 无条件或 always，直接匹配
			return map[string]interface{}{
				"evaluated":     true,
				"branch":        edge.Label,
				"matched_edge":  edge.ID,
				"target_node":   edge.Target,
			}, nil
		}

		// 评估条件
		if e.evaluateCondition(edge.Condition, exec) {
			return map[string]interface{}{
				"evaluated":     true,
				"branch":        edge.Label,
				"matched_edge":  edge.ID,
				"target_node":   edge.Target,
			}, nil
		}
	}

	// 没有匹配的条件，使用第一条边作为默认
	if len(edges) > 0 {
		return map[string]interface{}{
			"evaluated":     true,
			"branch":        "default",
			"matched_edge":  edges[0].ID,
			"target_node":   edges[0].Target,
		}, nil
	}

	return map[string]interface{}{"evaluated": true, "branch": "default"}, nil
}

// executeParallelNode 执行并行节点
// 真正并行执行所有出边指向的分支
func (e *Engine) executeParallelNode(flow *Flow, node *FlowNode, exec *Execution, nodeMap map[string]FlowNode, edgeMap map[string][]FlowEdge) (map[string]interface{}, error) {
	edges := edgeMap[node.ID]
	if len(edges) == 0 {
		return map[string]interface{}{"parallel": true, "branches": 0}, nil
	}

	// 获取最大并发数
	maxConcurrent := 0
	if m, ok := node.Config["max_concurrent"].(float64); ok {
		maxConcurrent = int(m)
	}

	e.logger.Info("executing parallel node",
		zap.String("node", node.ID),
		zap.Int("branches", len(edges)),
		zap.Int("max_concurrent", maxConcurrent),
	)

	// 并行执行所有分支
	type branchResult struct {
		edgeID   string
		target   string
		output   map[string]interface{}
		err      error
		context  map[string]interface{}
	}

	results := make(chan branchResult, len(edges))
	var wg sync.WaitGroup

	// 控制并发数
	semaphore := make(chan struct{}, maxConcurrent)
	if maxConcurrent == 0 {
		// 不限制并发，创建足够大的 channel
		semaphore = make(chan struct{}, len(edges))
	}

	for _, edge := range edges {
		wg.Add(1)
		go func(edge FlowEdge) {
			defer wg.Done()
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			// 复制上下文，避免并发写入冲突
			branchCtx := e.copyContext(exec.Context)

			// 执行分支
			output, err := e.executeBranch(flow, edge.Target, branchCtx, nodeMap, edgeMap)

			results <- branchResult{
				edgeID:  edge.ID,
				target:  edge.Target,
				output:  output,
				err:     err,
				context: branchCtx,
			}
		}(edge)
	}

	// 等待所有分支完成
	go func() {
		wg.Wait()
		close(results)
	}()

	// 收集结果
	branchOutputs := make(map[string]interface{})
	branchErrors := make(map[string]string)
	allResults := make([]map[string]interface{}, 0)

	for result := range results {
		if result.err != nil {
			branchErrors[result.edgeID] = result.err.Error()
			e.logger.Warn("branch execution failed",
				zap.String("edge", result.edgeID),
				zap.Error(result.err),
			)
		} else {
			branchOutputs[result.edgeID] = result.output
			allResults = append(allResults, result.output)
		}
	}

	// 合并上下文（将各分支的输出合并到主上下文）
	for key, value := range branchOutputs {
		exec.Context["branch_"+key] = value
	}
	exec.Context["parallel_results"] = allResults

	// 查找合并点（所有分支最终汇聚的节点）
	// 简化处理：假设所有分支最终都会汇聚到同一个节点
	// 实际应该通过图分析找到汇聚点
	mergeNode := e.findMergeNode(edges, nodeMap, edgeMap)

	e.logger.Info("parallel execution completed",
		zap.Int("branches", len(edges)),
		zap.Int("success", len(branchOutputs)),
		zap.Int("failed", len(branchErrors)),
		zap.String("merge_node", mergeNode),
	)

	return map[string]interface{}{
		"parallel":       true,
		"branches":       len(edges),
		"successful":     len(branchOutputs),
		"failed":         len(branchErrors),
		"branch_outputs": branchOutputs,
		"branch_errors":  branchErrors,
		"next_node":      mergeNode,
	}, nil
}

// executeBranch 执行单个分支
func (e *Engine) executeBranch(flow *Flow, startNodeID string, ctx map[string]interface{}, nodeMap map[string]FlowNode, edgeMap map[string][]FlowEdge) (map[string]interface{}, error) {
	currentNodeID := startNodeID
	visited := make(map[string]bool) // 防止无限循环
	maxSteps := 100                  // 最大步数限制

	output := make(map[string]interface{})

	for i := 0; i < maxSteps; i++ {
		if visited[currentNodeID] {
			// 检测到循环，停止
			break
		}
		visited[currentNodeID] = true

		node, ok := nodeMap[currentNodeID]
		if !ok {
			break
		}

		// 不执行某些特殊节点（由主流程处理）
		if node.Type == NodeTypeParallel || node.Type == NodeTypeLoop || node.Type == NodeTypeHuman || node.Type == NodeTypeWebhook {
			// 遇到这些节点，停止分支执行，返回当前位置
			output["stopped_at"] = currentNodeID
			output["stopped_type"] = string(node.Type)
			return output, nil
		}

		// 执行节点
		switch node.Type {
		case NodeTypeAgent:
			result, err := e.executeAgentNode(&node, &Execution{Context: ctx})
			if err != nil {
				return nil, err
			}
			output[currentNodeID] = result
			ctx[currentNodeID+"_output"] = result

		case NodeTypeCondition:
			// 条件节点：选择分支
			edges := edgeMap[node.ID]
			nextNodeID := e.selectNextNode(edges, &Execution{Context: ctx})
			if nextNodeID != "" {
				currentNodeID = nextNodeID
				continue
			}

		case NodeTypeEnd:
			output["completed"] = true
			return output, nil

		case NodeTypeSubFlow:
			// 子流程在分支中简化处理
			output[currentNodeID] = map[string]interface{}{"subflow": "skipped_in_branch"}

		default:
			// 其他节点类型
			output[currentNodeID] = map[string]interface{}{"executed": true}
		}

		// 获取下一个节点
		edges := edgeMap[node.ID]
		if len(edges) == 0 {
			break
		}

		// 选择下一条边
		nextNodeID := e.selectNextNode(edges, &Execution{Context: ctx})
		if nextNodeID == "" {
			break
		}
		currentNodeID = nextNodeID
	}

	return output, nil
}

// copyContext 复制上下文
func (e *Engine) copyContext(ctx map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range ctx {
		result[k] = v
	}
	return result
}

// findMergeNode 查找合并节点
// 简化实现：查找所有分支最终汇聚的节点
func (e *Engine) findMergeNode(edges []FlowEdge, nodeMap map[string]FlowNode, edgeMap map[string][]FlowEdge) string {
	if len(edges) == 0 {
		return ""
	}

	// 简化处理：返回空，让主流程继续线性执行
	// 实际应该通过图分析找到所有分支的汇聚点
	return ""
}

// executeLoopNode 执行循环节点
func (e *Engine) executeLoopNode(node *FlowNode, exec *Execution, edgeMap map[string][]FlowEdge) (map[string]interface{}, error) {
	// 获取循环计数
	count := 0
	if c, ok := exec.Context[node.ID+"_loop_count"].(int); ok {
		count = c
	}

	// 检查最大循环次数
	maxLoop := 10
	if m, ok := node.Config["max_loop"].(float64); ok {
		maxLoop = int(m)
	}

	// 检查循环条件
	shouldContinue := true
	if cond, ok := node.Config["condition"].(string); ok && cond != "" {
		// 评估循环条件
		shouldContinue = e.evaluateExpression(cond, exec)
	}

	// 检查是否达到最大循环次数或条件不满足
	if count >= maxLoop || !shouldContinue {
		// 循环结束，清除计数
		delete(exec.Context, node.ID+"_loop_count")
		delete(exec.Context, node.ID+"_in_loop")
		return map[string]interface{}{"loop_completed": true, "iterations": count}, nil
	}

	// 更新计数
	exec.Context[node.ID+"_loop_count"] = count + 1
	exec.Context[node.ID+"_in_loop"] = true

	// 查找循环体入口（第一条边）和出口（标记为 exit 的边）
	edges := edgeMap[node.ID]
	var loopBodyEntry string
	var loopExit string

	for _, edge := range edges {
		if edge.Label == "exit" || edge.Label == "false" {
			loopExit = edge.Target
		} else if loopBodyEntry == "" {
			loopBodyEntry = edge.Target
		}
	}

	// 如果没有明确的出口，使用第二条边作为出口
	if loopExit == "" && len(edges) > 1 {
		loopExit = edges[1].Target
	}

	// 设置循环信息到上下文
	exec.Context[node.ID+"_loop_body_entry"] = loopBodyEntry
	exec.Context[node.ID+"_loop_exit"] = loopExit

	return map[string]interface{}{
		"loop":           true,
		"iteration":      count + 1,
		"loop_body_entry": loopBodyEntry,
		"loop_exit":       loopExit,
	}, nil
}

// executeSubFlowNode 执行子流程节点
func (e *Engine) executeSubFlowNode(node *FlowNode, exec *Execution) (map[string]interface{}, error) {
	subFlowID, ok := node.Config["flow_id"].(string)
	if !ok || subFlowID == "" {
		return nil, fmt.Errorf("subflow flow_id is required")
	}

	// 获取子流程
	subFlow, err := e.getFlow(subFlowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get subflow: %w", err)
	}

	// 构建子流程输入
	// 支持从节点配置中获取输入映射
	subInput := e.buildInput(node, exec)

	// 构建子流程变量
	subVariables := make(map[string]interface{})
	// 复制父流程变量
	for k, v := range exec.Variables {
		subVariables[k] = v
	}
	// 添加节点配置中的变量
	if node.Inputs != nil {
		for k, v := range node.Inputs {
			resolved := e.resolveVariables(fmt.Sprintf("%v", v), exec.Context)
			subVariables[k] = resolved
		}
	}

	// 执行子流程
	resp, err := e.Execute(subFlow, ExecuteRequest{
		Input:     fmt.Sprintf("%v", subInput),
		Variables: subVariables,
		Context:   exec.Context,
	})
	if err != nil {
		return nil, fmt.Errorf("subflow execution failed: %w", err)
	}

	// 等待子流程完成
	for {
		time.Sleep(100 * time.Millisecond)
		subExec, err := e.GetExecution(resp.ExecutionID)
		if err != nil {
			return nil, err
		}
		if subExec.Status == ExecutionStatusCompleted {
			// 处理输出映射
			result := map[string]interface{}{
				"output":       subExec.Output,
				"execution_id": subExec.ID,
			}

			// 应用输出映射
			if node.Outputs != nil {
				for outputName, storeName := range node.Outputs {
					if val, ok := subExec.Context[outputName]; ok {
						exec.Context[storeName] = val
						result[storeName] = val
					} else if subExec.Output != "" {
						// 尝试解析输出为 JSON
						var outputMap map[string]interface{}
						if err := json.Unmarshal([]byte(subExec.Output), &outputMap); err == nil {
							if val, exists := outputMap[outputName]; exists {
								exec.Context[storeName] = val
								result[storeName] = val
							}
						}
					}
				}
			}

			// 合并子流程变量到父流程
			for k, v := range subExec.Variables {
				exec.Variables[k] = v
			}

			return result, nil
		}
		if subExec.Status == ExecutionStatusFailed {
			return nil, fmt.Errorf("subflow failed: %s", subExec.Error)
		}
	}
}

// executeWebhookNode 执行 Webhook 节点
func (e *Engine) executeWebhookNode(node *FlowNode, exec *Execution) (map[string]interface{}, error) {
	// 生成 webhook URL
	webhookID := fmt.Sprintf("%s_%s", exec.ID, node.ID)
	webhookURL := fmt.Sprintf("/api/webhooks/%s", webhookID)

	// 设置等待状态
	exec.Status = ExecutionStatusWaiting
	exec.Context["waiting_for"] = node.ID
	exec.Context["webhook_id"] = webhookID
	exec.Context["webhook_url"] = webhookURL

	// 设置超时（默认 1 小时）
	timeout := 3600
	if t, ok := node.Config["timeout"].(float64); ok {
		timeout = int(t)
	}
	exec.Context["webhook_timeout"] = timeout

	// 保存状态
	e.saveExecution(exec)

	return map[string]interface{}{
		"waiting":     true,
		"webhook_id":  webhookID,
		"webhook_url": webhookURL,
		"timeout":     timeout,
	}, nil
}

// WebhookCallback 处理 Webhook 回调
func (e *Engine) WebhookCallback(webhookID string, payload map[string]interface{}) error {
	// 解析 webhook ID: executionID_nodeID
	parts := strings.SplitN(webhookID, "_", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid webhook ID format")
	}
	executionID := parts[0]

	// 获取执行实例
	e.mu.RLock()
	exec, ok := e.running[executionID]
	e.mu.RUnlock()

	if !ok {
		// 从数据库加载
		var err error
		exec, err = e.GetExecution(executionID)
		if err != nil {
			return fmt.Errorf("execution not found: %s", executionID)
		}
	}

	if exec.Status != ExecutionStatusWaiting {
		return fmt.Errorf("execution is not waiting: %s", exec.Status)
	}

	// 更新上下文
	exec.Context["webhook_payload"] = payload
	exec.Context["webhook_received_at"] = time.Now().Format(time.RFC3339)
	exec.Status = ExecutionStatusRunning

	// 获取流程定义
	flow, err := e.getFlow(exec.FlowID)
	if err != nil {
		return err
	}

	// 继续执行
	go e.runFlow(flow, exec)

	return nil
}

// selectNextNode 选择下一个节点
// 支持多分支条件匹配：按顺序评估每条边的条件，匹配成功则返回对应目标节点
func (e *Engine) selectNextNode(edges []FlowEdge, exec *Execution) string {
	// 记录默认边（无条件或 always）
	var defaultEdge *FlowEdge

	for i := range edges {
		edge := &edges[i]

		// 无条件边，作为默认选项
		if edge.Condition == nil || edge.Condition.Type == "" || edge.Condition.Type == "always" {
			if defaultEdge == nil {
				defaultEdge = edge
			}
			continue
		}

		// 评估条件
		if e.evaluateCondition(edge.Condition, exec) {
			e.logger.Info("condition matched",
				zap.String("edge", edge.ID),
				zap.String("type", edge.Condition.Type),
				zap.String("target", edge.Target),
			)
			return edge.Target
		}
	}

	// 没有匹配的条件，使用默认边
	if defaultEdge != nil {
		return defaultEdge.Target
	}

	// 最后兜底：返回第一条边
	if len(edges) > 0 {
		return edges[0].Target
	}

	return ""
}

// evaluateCondition 评估条件
func (e *Engine) evaluateCondition(cond *EdgeCondition, exec *Execution) bool {
	switch cond.Type {
	case "always", "":
		return true

	case "expression":
		return e.evaluateExpression(cond.Expression, exec)

	case "intent":
		return e.evaluateIntent(cond.Intent, exec)

	case "llm":
		// LLM 判断 - 同步执行
		return e.evaluateLLM(cond.LLMQuery, exec)

	default:
		return false
	}
}

// evaluateLLM 使用 LLM 评估条件
// 支持单条件判断和多选项判断
func (e *Engine) evaluateLLM(query string, exec *Execution) bool {
	if query == "" {
		return true
	}

	// 获取 Agent Router
	e.mu.RLock()
	router := e.agentRouter
	agentMgr := e.agentMgr
	e.mu.RUnlock()

	if router == nil || agentMgr == nil {
		e.logger.Warn("agent router or manager not available for LLM evaluation")
		return true
	}

	// 获取第一个可用的 Agent
	agents := agentMgr.List()
	if len(agents) == 0 {
		return true
	}

	// 构建评估 prompt
	input, _ := exec.Context["input"].(string)
	
	// 获取上下文中的额外信息
	contextInfo := ""
	if ctxData, ok := exec.Context["last_output"].(string); ok && ctxData != "" {
		contextInfo = fmt.Sprintf("\n上下文信息: %s", ctxData)
	}

	prompt := fmt.Sprintf(`你是一个条件判断助手。请根据以下信息判断条件是否满足。

用户输入: %s%s

判断问题: %s

请只回答 "是" 或 "否"，不要有其他内容。`, input, contextInfo, query)

	agentInstance, err := router.GetOrCreateAgent(agents[0].ID)
	if err != nil {
		e.logger.Warn("failed to get agent for LLM evaluation", zap.Error(err))
		return true
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := agentInstance.Process(ctx, &types.Request{
		Content:   prompt,
		SessionID: exec.ID + "_llm_eval",
	})
	if err != nil {
		e.logger.Warn("LLM evaluation failed", zap.Error(err))
		return true
	}

	// 解析结果
	result := strings.TrimSpace(strings.ToLower(resp.Content))
	return result == "是" || result == "yes" || result == "true" || result == "1"
}

// evaluateLLMBranch 使用 LLM 进行多分支判断
// 返回匹配的分支标签
func (e *Engine) evaluateLLMBranch(query string, branches []string, exec *Execution) string {
	if query == "" || len(branches) == 0 {
		return branches[0]
	}

	// 获取 Agent Router
	e.mu.RLock()
	router := e.agentRouter
	agentMgr := e.agentMgr
	e.mu.RUnlock()

	if router == nil || agentMgr == nil {
		e.logger.Warn("agent router or manager not available for LLM branch evaluation")
		return branches[0]
	}

	agents := agentMgr.List()
	if len(agents) == 0 {
		return branches[0]
	}

	input, _ := exec.Context["input"].(string)
	
	// 构建分支选项
	branchList := ""
	for i, b := range branches {
		branchList += fmt.Sprintf("%d. %s\n", i+1, b)
	}

	prompt := fmt.Sprintf(`你是一个分类助手。请根据用户输入选择最合适的分类。

用户输入: %s

判断问题: %s

可选分类:
%s

请只回答分类名称，不要有其他内容。`, input, query, branchList)

	agentInstance, err := router.GetOrCreateAgent(agents[0].ID)
	if err != nil {
		e.logger.Warn("failed to get agent for LLM branch evaluation", zap.Error(err))
		return branches[0]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := agentInstance.Process(ctx, &types.Request{
		Content:   prompt,
		SessionID: exec.ID + "_llm_branch",
	})
	if err != nil {
		e.logger.Warn("LLM branch evaluation failed", zap.Error(err))
		return branches[0]
	}

	// 解析结果，匹配分支
	result := strings.TrimSpace(resp.Content)
	for _, b := range branches {
		if strings.EqualFold(result, b) {
			return b
		}
	}

	// 尝试数字匹配
	for i, b := range branches {
		if result == fmt.Sprintf("%d", i+1) {
			return b
		}
	}

	return branches[0]
}

// evaluateExpression 评估表达式
// 支持: ==, !=, >, <, >=, <=, &&, ||, contains, startsWith, endsWith
func (e *Engine) evaluateExpression(expr string, exec *Execution) bool {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true
	}

	// 替换变量
	resolved := e.resolveVariables(expr, exec.Context)

	// 处理逻辑运算符
	if strings.Contains(resolved, "&&") {
		parts := strings.Split(resolved, "&&")
		for _, p := range parts {
			if !e.evaluateExpression(p, exec) {
				return false
			}
		}
		return true
	}

	if strings.Contains(resolved, "||") {
		parts := strings.Split(resolved, "||")
		for _, p := range parts {
			if e.evaluateExpression(p, exec) {
				return true
			}
		}
		return false
	}

	// 处理比较运算符
	// >= 和 <= 需要先检查
	if strings.Contains(resolved, ">=") {
		parts := strings.SplitN(resolved, ">=", 2)
		if len(parts) == 2 {
			return compareNumbers(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])) >= 0
		}
	}

	if strings.Contains(resolved, "<=") {
		parts := strings.SplitN(resolved, "<=", 2)
		if len(parts) == 2 {
			return compareNumbers(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])) <= 0
		}
	}

	if strings.Contains(resolved, "==") {
		parts := strings.SplitN(resolved, "==", 2)
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			// 移除引号
			left = strings.Trim(left, "\"'")
			right = strings.Trim(right, "\"'")
			return left == right
		}
	}

	if strings.Contains(resolved, "!=") {
		parts := strings.SplitN(resolved, "!=", 2)
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			left = strings.Trim(left, "\"'")
			right = strings.Trim(right, "\"'")
			return left != right
		}
	}

	if strings.Contains(resolved, ">") {
		parts := strings.SplitN(resolved, ">", 2)
		if len(parts) == 2 {
			return compareNumbers(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])) > 0
		}
	}

	if strings.Contains(resolved, "<") {
		parts := strings.SplitN(resolved, "<", 2)
		if len(parts) == 2 {
			return compareNumbers(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])) < 0
		}
	}

	// 处理字符串函数
	if strings.Contains(resolved, "contains(") {
		// 格式: contains(haystack, needle)
		match := regexp.MustCompile(`contains\(([^,]+),\s*([^)]+)\)`).FindStringSubmatch(resolved)
		if len(match) == 3 {
			haystack := strings.Trim(strings.TrimSpace(match[1]), "\"'")
			needle := strings.Trim(strings.TrimSpace(match[2]), "\"'")
			return strings.Contains(haystack, needle)
		}
	}

	if strings.Contains(resolved, "startsWith(") {
		match := regexp.MustCompile(`startsWith\(([^,]+),\s*([^)]+)\)`).FindStringSubmatch(resolved)
		if len(match) == 3 {
			str := strings.Trim(strings.TrimSpace(match[1]), "\"'")
			prefix := strings.Trim(strings.TrimSpace(match[2]), "\"'")
			return strings.HasPrefix(str, prefix)
		}
	}

	if strings.Contains(resolved, "endsWith(") {
		match := regexp.MustCompile(`endsWith\(([^,]+),\s*([^)]+)\)`).FindStringSubmatch(resolved)
		if len(match) == 3 {
			str := strings.Trim(strings.TrimSpace(match[1]), "\"'")
			suffix := strings.Trim(strings.TrimSpace(match[2]), "\"'")
			return strings.HasSuffix(str, suffix)
		}
	}

	// 默认：检查是否为 true
	resolved = strings.ToLower(strings.TrimSpace(resolved))
	return resolved == "true" || resolved == "1" || resolved == "yes"
}

// compareNumbers 比较两个数字
func compareNumbers(a, b string) int {
	aFloat, aErr := strconv.ParseFloat(a, 64)
	bFloat, bErr := strconv.ParseFloat(b, 64)

	if aErr != nil || bErr != nil {
		// 无法解析为数字，按字符串比较
		if a < b {
			return -1
		} else if a > b {
			return 1
		}
		return 0
	}

	if aFloat < bFloat {
		return -1
	} else if aFloat > bFloat {
		return 1
	}
	return 0
}

// evaluateIntent 评估意图
// 支持多种匹配模式：关键词、正则、意图列表
func (e *Engine) evaluateIntent(intent string, exec *Execution) bool {
	input, _ := exec.Context["input"].(string)
	input = strings.ToLower(input)

	// 解析意图配置
	// 格式1: "关键词1,关键词2" - 包含任一关键词
	// 格式2: "intent:xxx" - 精确匹配意图
	// 格式3: "regex:xxx" - 正则匹配
	// 格式4: "any:关键词1,关键词2" - 包含任一
	// 格式5: "all:关键词1,关键词2" - 包含全部

	if strings.HasPrefix(intent, "regex:") {
		pattern := strings.TrimPrefix(intent, "regex:")
		matched, err := regexp.MatchString(pattern, input)
		if err != nil {
			e.logger.Warn("invalid regex pattern for intent", zap.String("pattern", pattern), zap.Error(err))
			return false
		}
		return matched
	}

	if strings.HasPrefix(intent, "intent:") {
		// 精确匹配意图（从上下文获取识别的意图）
		targetIntent := strings.TrimPrefix(intent, "intent:")
		detectedIntent, _ := exec.Context["detected_intent"].(string)
		return strings.EqualFold(detectedIntent, targetIntent)
	}

	if strings.HasPrefix(intent, "any:") {
		// 包含任一关键词
		keywords := strings.Split(strings.TrimPrefix(intent, "any:"), ",")
		for _, kw := range keywords {
			kw = strings.TrimSpace(kw)
			if kw != "" && strings.Contains(input, strings.ToLower(kw)) {
				return true
			}
		}
		return false
	}

	if strings.HasPrefix(intent, "all:") {
		// 包含全部关键词
		keywords := strings.Split(strings.TrimPrefix(intent, "all:"), ",")
		for _, kw := range keywords {
			kw = strings.TrimSpace(kw)
			if kw != "" && !strings.Contains(input, strings.ToLower(kw)) {
				return false
			}
		}
		return len(keywords) > 0
	}

	// 默认：简单的关键词包含匹配
	keywords := strings.Split(intent, ",")
	for _, kw := range keywords {
		kw = strings.TrimSpace(kw)
		if kw != "" && strings.Contains(input, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// resolveVariables 解析变量
func (e *Engine) resolveVariables(template string, ctx map[string]interface{}) string {
	result := template
	for k, v := range ctx {
		placeholder := "{{" + k + "}}"
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", v))
	}
	return result
}

// buildInput 构建节点输入
// 支持节点级别的输入映射
func (e *Engine) buildInput(node *FlowNode, exec *Execution) string {
	// 优先使用节点输入映射
	if node.Inputs != nil && len(node.Inputs) > 0 {
		// 构建输入变量映射
		inputVars := make(map[string]interface{})
		for localName, sourceExpr := range node.Inputs {
			// 解析来源表达式
			value := e.resolveVariables(sourceExpr, exec.Context)
			inputVars[localName] = value
		}

		// 如果有 prompt，使用输入变量替换
		if node.Prompt != "" {
			prompt := node.Prompt
			for k, v := range inputVars {
				placeholder := "{{" + k + "}}"
				prompt = strings.ReplaceAll(prompt, placeholder, fmt.Sprintf("%v", v))
			}
			return prompt
		}

		// 否则返回 JSON 格式的输入
		if len(inputVars) > 0 {
			data, _ := json.Marshal(inputVars)
			return string(data)
		}
	}

	// 使用 prompt 模板
	if node.Prompt != "" {
		return e.resolveVariables(node.Prompt, exec.Context)
	}

	// 默认使用流程输入
	if input, ok := exec.Context["input"].(string); ok {
		return input
	}
	return exec.Input
}

// buildOutput 构建输出
// 支持节点级别的输出映射
func (e *Engine) buildOutput(node *FlowNode, exec *Execution) string {
	// 优先使用节点输出映射
	if node.Outputs != nil && len(node.Outputs) > 0 {
		outputVars := make(map[string]interface{})
		for outputName, storeName := range node.Outputs {
			// 从上下文获取输出值
			if val, ok := exec.Context[node.ID+"_"+outputName]; ok {
				outputVars[storeName] = val
				// 同时存储到上下文
				exec.Context[storeName] = val
			}
		}
		if len(outputVars) > 0 {
			data, _ := json.Marshal(outputVars)
			return string(data)
		}
	}

	// 使用输出模板
	if template, ok := node.Config["output_template"].(string); ok && template != "" {
		return e.resolveVariables(template, exec.Context)
	}
	if output, ok := exec.Context["output"].(string); ok {
		return output
	}
	return ""
}

// saveExecution 保存执行记录
func (e *Engine) saveExecution(exec *Execution) error {
	varsJSON, _ := json.Marshal(exec.Variables)
	ctxJSON, _ := json.Marshal(exec.Context)
	historyJSON, _ := json.Marshal(exec.History)

	_, err := e.db.Exec(`
		INSERT OR REPLACE INTO flow_executions
		(id, flow_id, status, trigger, input, output, variables, current_node, context, history, started_at, completed_at, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, exec.ID, exec.FlowID, exec.Status, exec.Trigger, exec.Input, exec.Output, string(varsJSON),
		exec.CurrentNode, string(ctxJSON), string(historyJSON),
		exec.StartedAt, exec.CompletedAt, exec.Error)
	return err
}

// GetExecution 获取执行记录
func (e *Engine) GetExecution(id string) (*Execution, error) {
	var exec Execution
	var varsJSON, ctxJSON, historyJSON sql.NullString

	err := e.db.QueryRow(`
		SELECT id, flow_id, status, trigger, input, output, variables, current_node, context, history, started_at, completed_at, error
		FROM flow_executions WHERE id=?
	`, id).Scan(&exec.ID, &exec.FlowID, &exec.Status, &exec.Trigger, &exec.Input, &exec.Output,
		&varsJSON, &exec.CurrentNode, &ctxJSON, &historyJSON,
		&exec.StartedAt, &exec.CompletedAt, &exec.Error)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("execution not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	if varsJSON.Valid {
		json.Unmarshal([]byte(varsJSON.String), &exec.Variables)
	}
	if ctxJSON.Valid {
		json.Unmarshal([]byte(ctxJSON.String), &exec.Context)
	}
	if historyJSON.Valid {
		json.Unmarshal([]byte(historyJSON.String), &exec.History)
	}

	return &exec, nil
}

// ListExecutions 列出执行记录
func (e *Engine) ListExecutions(flowID string, limit int) ([]*Execution, error) {
	query := "SELECT id, flow_id, status, trigger, input, output, variables, current_node, context, history, started_at, completed_at, error FROM flow_executions"
	args := []interface{}{}

	if flowID != "" {
		query += " WHERE flow_id = ?"
		args = append(args, flowID)
	}
	query += " ORDER BY started_at DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := e.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	defer rows.Close()

	var executions []*Execution
	for rows.Next() {
		var exec Execution
		var varsJSON, ctxJSON, historyJSON sql.NullString
		err := rows.Scan(&exec.ID, &exec.FlowID, &exec.Status, &exec.Trigger, &exec.Input, &exec.Output,
			&varsJSON, &exec.CurrentNode, &ctxJSON, &historyJSON,
			&exec.StartedAt, &exec.CompletedAt, &exec.Error)
		if err != nil {
			continue
		}

		if varsJSON.Valid {
			json.Unmarshal([]byte(varsJSON.String), &exec.Variables)
		}
		if ctxJSON.Valid {
			json.Unmarshal([]byte(ctxJSON.String), &exec.Context)
		}
		if historyJSON.Valid {
			json.Unmarshal([]byte(historyJSON.String), &exec.History)
		}

		executions = append(executions, &exec)
	}

	return executions, nil
}

// ListExecutionsByStatus 按状态列出执行记录
func (e *Engine) ListExecutionsByStatus(status ExecutionStatus, limit int) ([]*Execution, error) {
	query := "SELECT id, flow_id, status, trigger, input, output, variables, current_node, context, history, started_at, completed_at, error FROM flow_executions"
	args := []interface{}{}

	if status != "" {
		query += " WHERE status = ?"
		args = append(args, string(status))
	}
	query += " ORDER BY started_at DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := e.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list executions by status: %w", err)
	}
	defer rows.Close()

	var executions []*Execution
	for rows.Next() {
		var exec Execution
		var varsJSON, ctxJSON, historyJSON sql.NullString
		err := rows.Scan(&exec.ID, &exec.FlowID, &exec.Status, &exec.Trigger, &exec.Input, &exec.Output,
			&varsJSON, &exec.CurrentNode, &ctxJSON, &historyJSON,
			&exec.StartedAt, &exec.CompletedAt, &exec.Error)
		if err != nil {
			continue
		}

		if varsJSON.Valid {
			json.Unmarshal([]byte(varsJSON.String), &exec.Variables)
		}
		if ctxJSON.Valid {
			json.Unmarshal([]byte(ctxJSON.String), &exec.Context)
		}
		if historyJSON.Valid {
			json.Unmarshal([]byte(historyJSON.String), &exec.History)
		}

		executions = append(executions, &exec)
	}

	return executions, nil
}

// getFlow 从数据库获取流程
func (e *Engine) getFlow(id string) (*Flow, error) {
	var flow Flow
	var defJSON, triggerJSON sql.NullString

	err := e.db.QueryRow(`
		SELECT id, name, description, type, definition, trigger, status, created_at, updated_at
		FROM flows WHERE id=?
	`, id).Scan(&flow.ID, &flow.Name, &flow.Description, &flow.Type, &defJSON, &triggerJSON, &flow.Status, &flow.CreatedAt, &flow.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get flow: %w", err)
	}

	if defJSON.Valid {
		json.Unmarshal([]byte(defJSON.String), &flow.Definition)
	}
	if triggerJSON.Valid && triggerJSON.String != "" {
		var trigger TriggerConfig
		if err := json.Unmarshal([]byte(triggerJSON.String), &trigger); err == nil {
			flow.Trigger = &trigger
		}
	}

	return &flow, nil
}