// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package flow

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
	db          *sql.DB
	agentMgr    *agent.Manager
	agentRouter *agent.Router
	msgMgr      *message.Manager
	logger      *zap.Logger
	running     map[string]*Execution // 正在执行的实例
	mu          sync.RWMutex
}

// NewEngine 创建执行引擎
func NewEngine(db *sql.DB, agentMgr *agent.Manager, msgMgr *message.Manager, logger *zap.Logger) (*Engine, error) {
	return &Engine{
		db:       db,
		agentMgr: agentMgr,
		msgMgr:   msgMgr,
		logger:   logger.Named("flow_engine"),
		running:  make(map[string]*Execution),
	}, nil
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
		ID:        generateID(),
		FlowID:    flow.ID,
		Status:    ExecutionStatusRunning,
		Input:     req.Input,
		Variables: req.Variables,
		Context:   req.Context,
		History:   []ExecutionStep{},
		StartedAt: time.Now(),
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
	defer func() {
		if r := recover(); r != nil {
			e.logger.Error("flow panic", zap.Any("error", r))
			exec.Status = ExecutionStatusFailed
			exec.Error = fmt.Sprintf("panic: %v", r)
			now := time.Now()
			exec.CompletedAt = &now
			e.saveExecution(exec)
		}

		e.mu.Lock()
		delete(e.running, exec.ID)
		e.mu.Unlock()
	}()

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

	// 执行循环
	for exec.Status == ExecutionStatusRunning {
		node, ok := nodeMap[exec.CurrentNode]
		if !ok {
			e.logger.Error("node not found", zap.String("node_id", exec.CurrentNode))
			exec.Status = ExecutionStatusFailed
			exec.Error = fmt.Sprintf("node not found: %s", exec.CurrentNode)
			break
		}

		// 记录步骤开始
		step := ExecutionStep{
			NodeID:    node.ID,
			NodeType:  node.Type,
			Status:    ExecutionStatusRunning,
			StartedAt: time.Now(),
		}

		// 执行节点
		output, err := e.executeNode(flow, &node, exec)
		if err != nil {
			step.Status = ExecutionStatusFailed
			step.Error = err.Error()
			exec.Status = ExecutionStatusFailed
			exec.Error = err.Error()
		} else {
			step.Status = ExecutionStatusCompleted
			step.Output = output
		}

		now := time.Now()
		step.EndedAt = &now
		exec.History = append(exec.History, step)

		// 检查是否结束
		if exec.Status != ExecutionStatusRunning || node.Type == NodeTypeEnd {
			exec.Status = ExecutionStatusCompleted
			exec.CompletedAt = &now
			break
		}

		// 检查是否等待人工
		if exec.Status == ExecutionStatusWaiting {
			break
		}

		// 获取下一个节点
		edges := edgeMap[node.ID]
		if len(edges) == 0 {
			// 没有出边，结束
			exec.Status = ExecutionStatusCompleted
			exec.CompletedAt = &now
			break
		}

		// 选择下一条边
		nextNodeID := e.selectNextNode(edges, exec)
		if nextNodeID == "" {
			exec.Status = ExecutionStatusFailed
			exec.Error = "no valid next node"
			break
		}
		exec.CurrentNode = nextNodeID

		// 保存状态
		e.saveExecution(exec)
	}

	// 最终保存
	e.saveExecution(exec)
}

// executeNode 执行单个节点
func (e *Engine) executeNode(flow *Flow, node *FlowNode, exec *Execution) (map[string]interface{}, error) {
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
		// 条件节点在 selectNextNode 中处理
		return map[string]interface{}{"evaluated": true}, nil

	case NodeTypeParallel:
		return e.executeParallelNode(node, exec)

	case NodeTypeLoop:
		return e.executeLoopNode(node, exec)

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

	// 保存状态
	e.saveExecution(exec)

	return map[string]interface{}{"waiting": true, "node_id": node.ID}, nil
}

// executeParallelNode 执行并行节点
func (e *Engine) executeParallelNode(node *FlowNode, exec *Execution) (map[string]interface{}, error) {
	// 并行节点标记，实际并行执行在 runFlow 中处理
	// 这里设置并行执行信息
	maxConcurrent := 0 // 0 表示不限制
	if m, ok := node.Config["max_concurrent"].(float64); ok {
		maxConcurrent = int(m)
	}
	exec.Context[node.ID+"_parallel"] = true
	exec.Context[node.ID+"_max_concurrent"] = maxConcurrent
	return map[string]interface{}{"parallel": true, "max_concurrent": maxConcurrent}, nil
}

// executeLoopNode 执行循环节点
func (e *Engine) executeLoopNode(node *FlowNode, exec *Execution) (map[string]interface{}, error) {
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
		return map[string]interface{}{"loop_completed": true, "iterations": count}, nil
	}

	// 更新计数
	exec.Context[node.ID+"_loop_count"] = count + 1
	exec.Context[node.ID+"_in_loop"] = true

	return map[string]interface{}{"loop": true, "iteration": count + 1}, nil
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

	// 执行子流程
	input := e.buildInput(node, exec)
	resp, err := e.Execute(subFlow, ExecuteRequest{
		Input:     fmt.Sprintf("%v", input),
		Variables: exec.Variables,
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
			return map[string]interface{}{"output": subExec.Output}, nil
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
func (e *Engine) selectNextNode(edges []FlowEdge, exec *Execution) string {
	for _, edge := range edges {
		if edge.Condition == nil {
			return edge.Target
		}

		// 评估条件
		if e.evaluateCondition(edge.Condition, exec) {
			return edge.Target
		}
	}

	// 没有匹配的条件，返回第一条边（默认）
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
	prompt := fmt.Sprintf(`你是一个条件判断助手。请根据以下信息判断条件是否满足。

用户输入: %s

判断问题: %s

请只回答 "是" 或 "否"，不要有其他内容。`, input, query)

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

// evaluateExpression 评估表达式
func (e *Engine) evaluateExpression(expr string, exec *Execution) bool {
	// 简单表达式解析
	// 支持: {{var}} == "value", {{var}} > 10 等
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true
	}

	// 替换变量
	resolved := e.resolveVariables(expr, exec.Context)

	// 简单比较
	if strings.Contains(resolved, "==") {
		parts := strings.SplitN(resolved, "==", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[0]) == strings.TrimSpace(parts[1])
		}
	}
	if strings.Contains(resolved, "!=") {
		parts := strings.SplitN(resolved, "!=", 2)
		if len(parts) == 2 {
			return strings.TrimSpace(parts[0]) != strings.TrimSpace(parts[1])
		}
	}

	return resolved == "true"
}

// evaluateIntent 评估意图
func (e *Engine) evaluateIntent(intent string, exec *Execution) bool {
	input, _ := exec.Context["input"].(string)
	// 简单的意图匹配
	return strings.Contains(strings.ToLower(input), strings.ToLower(intent))
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
func (e *Engine) buildInput(node *FlowNode, exec *Execution) string {
	if node.Prompt != "" {
		return e.resolveVariables(node.Prompt, exec.Context)
	}
	if input, ok := exec.Context["input"].(string); ok {
		return input
	}
	return exec.Input
}

// buildOutput 构建输出
func (e *Engine) buildOutput(node *FlowNode, exec *Execution) string {
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
		(id, flow_id, status, input, output, variables, current_node, context, history, started_at, completed_at, error)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, exec.ID, exec.FlowID, exec.Status, exec.Input, exec.Output, string(varsJSON),
		exec.CurrentNode, string(ctxJSON), string(historyJSON),
		exec.StartedAt, exec.CompletedAt, exec.Error)
	return err
}

// GetExecution 获取执行记录
func (e *Engine) GetExecution(id string) (*Execution, error) {
	var exec Execution
	var varsJSON, ctxJSON, historyJSON sql.NullString

	err := e.db.QueryRow(`
		SELECT id, flow_id, status, input, output, variables, current_node, context, history, started_at, completed_at, error
		FROM flow_executions WHERE id=?
	`, id).Scan(&exec.ID, &exec.FlowID, &exec.Status, &exec.Input, &exec.Output,
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
	query := "SELECT id, flow_id, status, input, output, variables, current_node, context, history, started_at, completed_at, error FROM flow_executions"
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
		err := rows.Scan(&exec.ID, &exec.FlowID, &exec.Status, &exec.Input, &exec.Output,
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