// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package flow

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gopaw/gopaw/internal/agent"
	"github.com/gopaw/gopaw/internal/agent/message"
	"go.uber.org/zap"
)

// Service 流程服务
type Service struct {
	db              *sql.DB
	engine          *Engine
	agentMgr        *agent.Manager
	msgMgr          *message.Manager
	triggerManager  *TriggerManager
	versionService  *VersionService
	templateService *TemplateService
	wsHub           *WebSocketHub
	taskQueue       *TaskQueue
	logger          *zap.Logger
}

// NewService 创建流程服务
func NewService(db *sql.DB, agentMgr *agent.Manager, msgMgr *message.Manager, logger *zap.Logger) (*Service, error) {
	if err := InitSchema(db); err != nil {
		return nil, fmt.Errorf("failed to init flow schema: %w", err)
	}

	s := &Service{
		db:       db,
		agentMgr: agentMgr,
		msgMgr:   msgMgr,
		logger:   logger.Named("flow_service"),
	}

	// 创建执行引擎
	engine, err := NewEngine(db, agentMgr, msgMgr, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create flow engine: %w", err)
	}
	s.engine = engine

	// 创建版本服务
	s.versionService = NewVersionService(db, s.logger)

	// 创建模板服务
	s.templateService = NewTemplateService(db, s.logger)

	// 填充默认模板
	if err := s.templateService.SeedDefaultTemplates(); err != nil {
		s.logger.Warn("failed to seed default templates", zap.Error(err))
	}

	// 创建任务队列（默认 10 个 Worker）
	s.taskQueue = NewTaskQueue(db, 10, logger)

	return s, nil
}

// SetWebSocketHub 设置 WebSocket Hub
func (s *Service) SetWebSocketHub(hub *WebSocketHub) {
	s.wsHub = hub
	s.engine.SetWebSocketHub(hub)
}

// GetWebSocketHub 获取 WebSocket Hub
func (s *Service) GetWebSocketHub() *WebSocketHub {
	return s.wsHub
}

// SetCronService 设置 CronService（用于触发器）
func (s *Service) SetCronService(cronService CronServiceInterface) {
	s.triggerManager = NewTriggerManager(cronService, s.logger)
}

// SetAgentRouter 设置 Agent Router（用于延迟注入）
func (s *Service) SetAgentRouter(router *agent.Router) {
	if s.engine != nil {
		s.engine.SetAgentRouter(router)
	}
}

// StartTaskQueue 启动任务队列
func (s *Service) StartTaskQueue() error {
	if s.taskQueue == nil {
		return fmt.Errorf("task queue not initialized")
	}

	// 注册流程执行处理器
	s.taskQueue.RegisterHandler(TaskTypeFlowExecute, s.handleFlowExecuteTask)

	return s.taskQueue.Start()
}

// StopTaskQueue 停止任务队列
func (s *Service) StopTaskQueue() {
	if s.taskQueue != nil {
		s.taskQueue.Stop()
	}
}

// handleFlowExecuteTask 处理流程执行任务
func (s *Service) handleFlowExecuteTask(ctx context.Context, task *Task) (map[string]interface{}, error) {
	flowID, ok := task.Payload["flow_id"].(string)
	if !ok {
		return nil, fmt.Errorf("flow_id is required")
	}

	input, _ := task.Payload["input"].(string)
	variables, _ := task.Payload["variables"].(map[string]interface{})

	flow, err := s.GetFlow(flowID)
	if err != nil {
		return nil, fmt.Errorf("failed to get flow: %w", err)
	}

	resp, err := s.engine.Execute(flow, ExecuteRequest{
		Input:     input,
		Variables: variables,
		Trigger:   "queue",
	})
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"execution_id": resp.ExecutionID,
		"status":       resp.Status,
	}, nil
}

// RestoreWaitingExecutions 恢复等待中的执行实例
// 服务启动时调用
func (s *Service) RestoreWaitingExecutions() error {
	return s.engine.RestoreWaitingExecutions()
}

// CreateFlow 创建流程
func (s *Service) CreateFlow(req CreateFlowRequest) (*Flow, error) {
	// 设置默认类型
	if req.Type == "" {
		req.Type = FlowTypeConversation
	}

	// 验证流程定义
	if err := s.validateDefinition(&req.Definition); err != nil {
		return nil, fmt.Errorf("invalid definition: %w", err)
	}

	defJSON, _ := json.Marshal(req.Definition)
	var triggerJSON []byte
	if req.Trigger != nil {
		triggerJSON, _ = json.Marshal(req.Trigger)
	}

	now := time.Now()
	_, err := s.db.Exec(`
		INSERT INTO flows (id, name, description, type, definition, trigger, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, 'draft', ?, ?)
	`, req.ID, req.Name, req.Description, req.Type, string(defJSON), string(triggerJSON), now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create flow: %w", err)
	}

	return &Flow{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Definition:  req.Definition,
		Trigger:     req.Trigger,
		Status:      FlowStatusDraft,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// UpdateFlow 更新流程
func (s *Service) UpdateFlow(id string, req UpdateFlowRequest) (*Flow, error) {
	// 获取现有流程
	flow, err := s.GetFlow(id)
	if err != nil {
		return nil, err
	}

	oldStatus := flow.Status

	// 更新字段
	if req.Name != "" {
		flow.Name = req.Name
	}
	if req.Description != "" {
		flow.Description = req.Description
	}
	if req.Type != "" {
		flow.Type = req.Type
	}
	if req.Status != "" {
		flow.Status = req.Status
	}
	if len(req.Definition.Nodes) > 0 {
		if err := s.validateDefinition(&req.Definition); err != nil {
			return nil, fmt.Errorf("invalid definition: %w", err)
		}
		flow.Definition = req.Definition
	}
	if req.Trigger != nil {
		flow.Trigger = req.Trigger
	}

	defJSON, _ := json.Marshal(flow.Definition)
	var triggerJSON []byte
	if flow.Trigger != nil {
		triggerJSON, _ = json.Marshal(flow.Trigger)
	}

	_, err = s.db.Exec(`
		UPDATE flows SET name=?, description=?, type=?, definition=?, trigger=?, status=?, updated_at=?
		WHERE id=?
	`, flow.Name, flow.Description, flow.Type, string(defJSON), string(triggerJSON), flow.Status, time.Now(), id)
	if err != nil {
		return nil, fmt.Errorf("failed to update flow: %w", err)
	}

	// 处理触发器注册/注销
	if s.triggerManager != nil && flow.Status != oldStatus {
		if flow.Status == FlowStatusActive {
			// 激活流程：注册触发器
			if err := s.triggerManager.RegisterTrigger(flow, func() error {
				_, err := s.Execute(flow.ID, ExecuteRequest{Input: ""})
				return err
			}); err != nil {
				s.logger.Warn("failed to register trigger", zap.Error(err))
			}
		} else if oldStatus == FlowStatusActive {
			// 停用流程：注销触发器
			if err := s.triggerManager.UnregisterTrigger(flow); err != nil {
				s.logger.Warn("failed to unregister trigger", zap.Error(err))
			}
		}
	}

	return flow, nil
}

// GetFlow 获取流程
func (s *Service) GetFlow(id string) (*Flow, error) {
	var flow Flow
	var defJSON, triggerJSON sql.NullString

	err := s.db.QueryRow(`
		SELECT id, name, description, type, definition, trigger, status, created_at, updated_at
		FROM flows WHERE id=?
	`, id).Scan(&flow.ID, &flow.Name, &flow.Description, &flow.Type, &defJSON, &triggerJSON, &flow.Status, &flow.CreatedAt, &flow.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("flow not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get flow: %w", err)
	}

	if defJSON.Valid {
		if err := json.Unmarshal([]byte(defJSON.String), &flow.Definition); err != nil {
			s.logger.Warn("failed to unmarshal definition", zap.Error(err))
		}
	}
	if triggerJSON.Valid && triggerJSON.String != "" {
		var trigger TriggerConfig
		if err := json.Unmarshal([]byte(triggerJSON.String), &trigger); err != nil {
			s.logger.Warn("failed to unmarshal trigger", zap.Error(err))
		} else {
			flow.Trigger = &trigger
		}
	}

	return &flow, nil
}

// ListFlows 列出流程
func (s *Service) ListFlows(flowType FlowType, status FlowStatus) ([]*Flow, error) {
	query := "SELECT id, name, description, type, definition, trigger, status, created_at, updated_at FROM flows WHERE 1=1"
	var args []interface{}

	if flowType != "" {
		query += " AND type = ?"
		args = append(args, flowType)
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	query += " ORDER BY updated_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list flows: %w", err)
	}
	defer rows.Close()

	var flows []*Flow
	for rows.Next() {
		var flow Flow
		var defJSON, triggerJSON sql.NullString
		err := rows.Scan(&flow.ID, &flow.Name, &flow.Description, &flow.Type, &defJSON, &triggerJSON, &flow.Status, &flow.CreatedAt, &flow.UpdatedAt)
		if err != nil {
			s.logger.Warn("failed to scan flow", zap.Error(err))
			continue
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

		flows = append(flows, &flow)
	}

	return flows, nil
}

// DeleteFlow 删除流程
func (s *Service) DeleteFlow(id string) error {
	_, err := s.db.Exec("DELETE FROM flows WHERE id=?", id)
	if err != nil {
		return fmt.Errorf("failed to delete flow: %w", err)
	}
	return nil
}

// Execute 执行流程
func (s *Service) Execute(id string, req ExecuteRequest) (*ExecuteResponse, error) {
	flow, err := s.GetFlow(id)
	if err != nil {
		return nil, err
	}

	if flow.Status != FlowStatusActive {
		return nil, fmt.Errorf("flow is not active: %s", flow.Status)
	}

	return s.engine.Execute(flow, req)
}

// ContinueExecution 继续执行（用于人工节点后）
func (s *Service) ContinueExecution(executionID string, input string) (*ExecuteResponse, error) {
	return s.engine.Continue(executionID, input)
}

// Step 单步执行（调试模式）
func (s *Service) Step(executionID string) (*ExecuteResponse, error) {
	return s.engine.Step(executionID)
}

// SetBreakpoints 设置断点
func (s *Service) SetBreakpoints(executionID string, breakpoints []string) error {
	return s.engine.SetBreakpoints(executionID, breakpoints)
}

// RetryFromNode 从特定节点重试执行
func (s *Service) RetryFromNode(executionID string, nodeID string) (*ExecuteResponse, error) {
	return s.engine.RetryFromNode(executionID, nodeID)
}

// GetExecution 获取执行记录
func (s *Service) GetExecution(executionID string) (*Execution, error) {
	return s.engine.GetExecution(executionID)
}

// ListExecutions 列出执行记录
func (s *Service) ListExecutions(flowID string, limit int) ([]*Execution, error) {
	return s.engine.ListExecutions(flowID, limit)
}

// ListExecutionsByStatus 按状态列出执行记录
func (s *Service) ListExecutionsByStatus(status ExecutionStatus, limit int) ([]*Execution, error) {
	return s.engine.ListExecutionsByStatus(status, limit)
}

// HandleWebhook 处理 Webhook 回调
func (s *Service) HandleWebhook(webhookID string, payload map[string]interface{}) error {
	return s.engine.WebhookCallback(webhookID, payload)
}

// validateDefinition 验证流程定义
func (s *Service) validateDefinition(def *FlowDefinition) error {
	if len(def.Nodes) == 0 {
		return fmt.Errorf("flow must have at least one node")
	}

	// 检查节点 ID 唯一性
	nodeIDs := make(map[string]bool)
	for _, node := range def.Nodes {
		if nodeIDs[node.ID] {
			return fmt.Errorf("duplicate node id: %s", node.ID)
		}
		nodeIDs[node.ID] = true
	}

	// 检查起始节点
	if def.StartNodeID == "" {
		// 自动查找 start 节点
		for _, node := range def.Nodes {
			if node.Type == NodeTypeStart {
				def.StartNodeID = node.ID
				break
			}
		}
		if def.StartNodeID == "" {
			return fmt.Errorf("flow must have a start node")
		}
	}

	// 检查边引用的节点存在
	for _, edge := range def.Edges {
		if !nodeIDs[edge.Source] {
			return fmt.Errorf("edge source node not found: %s", edge.Source)
		}
		if !nodeIDs[edge.Target] {
			return fmt.Errorf("edge target node not found: %s", edge.Target)
		}
	}

	return nil
}

// generateID 生成随机 ID
func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ========== 版本管理方法 ==========

// CreateVersion 创建流程版本
func (s *Service) CreateVersion(flowID string, req CreateVersionRequest, createdBy string) (*FlowVersion, error) {
	return s.versionService.CreateVersion(flowID, req, createdBy)
}

// ListVersions 列出流程版本
func (s *Service) ListVersions(flowID string) ([]*FlowVersion, error) {
	return s.versionService.ListVersions(flowID)
}

// GetVersion 获取特定版本
func (s *Service) GetVersion(flowID string, version int) (*FlowVersion, error) {
	return s.versionService.GetVersion(flowID, version)
}

// RollbackVersion 回滚到指定版本
func (s *Service) RollbackVersion(flowID string, version int) (*Flow, error) {
	return s.versionService.RollbackVersion(flowID, version, s)
}

// DeleteVersion 删除指定版本
func (s *Service) DeleteVersion(flowID string, version int) error {
	return s.versionService.DeleteVersion(flowID, version)
}

// CompareVersions 对比两个版本
func (s *Service) CompareVersions(flowID string, fromVersion, toVersion int) (*VersionDiff, error) {
	return s.versionService.CompareVersions(flowID, fromVersion, toVersion)
}

// ========== 模板管理方法 ==========

// ListTemplates 列出模板
func (s *Service) ListTemplates(category string, publicOnly bool) ([]*FlowTemplate, error) {
	return s.templateService.ListTemplates(category, publicOnly)
}

// GetTemplate 获取模板
func (s *Service) GetTemplate(id string) (*FlowTemplate, error) {
	return s.templateService.GetTemplate(id)
}

// CreateTemplate 创建模板
func (s *Service) CreateTemplate(req CreateTemplateRequest, author string) (*FlowTemplate, error) {
	return s.templateService.CreateTemplate(req, author)
}

// UseTemplate 使用模板创建流程
func (s *Service) UseTemplate(templateID string, req CreateFlowRequest) (*Flow, error) {
	return s.templateService.UseTemplate(templateID, s, req)
}

// DeleteTemplate 删除模板
func (s *Service) DeleteTemplate(id string) error {
	return s.templateService.DeleteTemplate(id)
}

// GetTemplateCategories 获取模板分类
func (s *Service) GetTemplateCategories() []TemplateCategory {
	return GetCategories()
}

// ========== 追踪管理方法 ==========

// GetTraceService 获取追踪服务
func (s *Service) GetTraceService() *TraceService {
	return s.engine.GetTraceService()
}

// GetTrace 获取执行追踪
func (s *Service) GetTrace(executionID string) (*Trace, error) {
	return s.engine.GetTraceService().GetTrace(executionID)
}

// GetTraceSpans 获取追踪的 Span 列表
func (s *Service) GetTraceSpans(executionID string) ([]*Span, error) {
	return s.engine.GetTraceService().GetTraceSpans(executionID)
}

// ========== 任务队列方法 ==========

// GetTaskQueue 获取任务队列
func (s *Service) GetTaskQueue() *TaskQueue {
	return s.taskQueue
}

// GetQueueStats 获取队列统计
func (s *Service) GetQueueStats() *QueueStats {
	if s.taskQueue == nil {
		return &QueueStats{}
	}
	return s.taskQueue.GetQueueStats()
}

// ListQueueTasks 列出队列任务
func (s *Service) ListQueueTasks(status TaskStatus, limit int) ([]*Task, error) {
	if s.taskQueue == nil {
		return nil, fmt.Errorf("task queue not initialized")
	}
	return s.taskQueue.ListTasks(status, limit)
}

// GetQueueTask 获取队列任务
func (s *Service) GetQueueTask(taskID string) (*Task, error) {
	if s.taskQueue == nil {
		return nil, fmt.Errorf("task queue not initialized")
	}
	return s.taskQueue.GetTask(taskID)
}

// EnqueueTask 入队任务
func (s *Service) EnqueueTask(taskType string, payload map[string]interface{}, opts ...TaskOption) (*Task, error) {
	if s.taskQueue == nil {
		return nil, fmt.Errorf("task queue not initialized")
	}
	return s.taskQueue.Enqueue(taskType, payload, opts...)
}

// CancelQueueTask 取消队列任务
func (s *Service) CancelQueueTask(taskID string) error {
	if s.taskQueue == nil {
		return fmt.Errorf("task queue not initialized")
	}
	return s.taskQueue.CancelTask(taskID)
}

// ========== 导入导出方法 ==========

// FlowExport 流程导出数据结构
type FlowExport struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        FlowType       `json:"type"`
	Definition  FlowDefinition `json:"definition"`
	Trigger     *TriggerConfig `json:"trigger,omitempty"`
	ExportedAt  string         `json:"exported_at"`
	Version     string         `json:"version"` // 导出版本
}

// FlowImport 流程导入数据结构
type FlowImport struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        FlowType       `json:"type"`
	Definition  FlowDefinition `json:"definition"`
	Trigger     *TriggerConfig `json:"trigger,omitempty"`
	// 导入选项
	Overwrite bool `json:"overwrite,omitempty"` // 是否覆盖同名流程
}

// ExportFlow 导出流程
func (s *Service) ExportFlow(id string) (*FlowExport, error) {
	f, err := s.GetFlow(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get flow: %w", err)
	}

	export := &FlowExport{
		Name:        f.Name,
		Description: f.Description,
		Type:        f.Type,
		Definition:  f.Definition,
		Trigger:     f.Trigger,
		ExportedAt:  time.Now().Format(time.RFC3339),
		Version:     "1.0",
	}

	return export, nil
}

// ImportFlow 导入流程
func (s *Service) ImportFlow(data *FlowImport) (*Flow, error) {
	// 检查是否存在同名流程
	var existingID string
	err := s.db.QueryRow("SELECT id FROM flows WHERE name = ?", data.Name).Scan(&existingID)
	if err == nil {
		// 存在同名流程
		if data.Overwrite {
			// 删除旧流程
			_, err = s.db.Exec("DELETE FROM flows WHERE id = ?", existingID)
			if err != nil {
				return nil, fmt.Errorf("failed to delete existing flow: %w", err)
			}
		} else {
			// 重命名
			data.Name = data.Name + " (导入)"
		}
	}

	// 创建新流程
	req := CreateFlowRequest{
		Name:        data.Name,
		Description: data.Description,
		Type:        data.Type,
		Definition:  data.Definition,
		Trigger:     data.Trigger,
	}

	created, err := s.CreateFlow(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create flow: %w", err)
	}

	return created, nil
}