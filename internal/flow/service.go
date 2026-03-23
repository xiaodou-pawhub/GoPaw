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
	"strings"
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
func (s *Service) ListFlows(flowType FlowType, status FlowStatus, search string) ([]*Flow, error) {
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
	if search != "" {
		query += " AND (name LIKE ? OR description LIKE ?)"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern)
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

// BatchUpdateStatus 批量更新流程状态
func (s *Service) BatchUpdateStatus(ids []string, status FlowStatus) (success []string, failed []string) {
	success = make([]string, 0)
	failed = make([]string, 0)

	for _, id := range ids {
		_, err := s.db.Exec("UPDATE flows SET status = ?, updated_at = ? WHERE id = ?", status, time.Now(), id)
		if err != nil {
			s.logger.Warn("failed to update flow status", zap.String("id", id), zap.Error(err))
			failed = append(failed, id)
		} else {
			success = append(success, id)
		}
	}

	return success, failed
}

// BatchDeleteFlows 批量删除流程
func (s *Service) BatchDeleteFlows(ids []string) (success []string, failed []string) {
	success = make([]string, 0)
	failed = make([]string, 0)

	for _, id := range ids {
		err := s.DeleteFlow(id)
		if err != nil {
			s.logger.Warn("failed to delete flow", zap.String("id", id), zap.Error(err))
			failed = append(failed, id)
		} else {
			success = append(success, id)
		}
	}

	return success, failed
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

// ========== 节点模板管理 ==========

// CreateNodeTemplate 创建节点模板
func (s *Service) CreateNodeTemplate(template *NodeTemplate) (*NodeTemplate, error) {
	if template.ID == "" {
		template.ID = "ntpl_" + generateID()
	}
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()

	configJSON, err := json.Marshal(template.NodeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal node config: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO node_templates (id, name, description, category, node_type, node_config, is_public, use_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, template.ID, template.Name, template.Description, template.Category, template.NodeType, string(configJSON), template.IsPublic, template.UseCount, template.CreatedAt, template.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create node template: %w", err)
	}

	return template, nil
}

// ListNodeTemplates 列出节点模板
func (s *Service) ListNodeTemplates(category string, nodeType NodeType) ([]*NodeTemplate, error) {
	query := "SELECT id, name, description, category, node_type, node_config, is_public, use_count, created_at, updated_at FROM node_templates WHERE 1=1"
	var args []interface{}

	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}
	if nodeType != "" {
		query += " AND node_type = ?"
		args = append(args, nodeType)
	}
	query += " ORDER BY use_count DESC, created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list node templates: %w", err)
	}
	defer rows.Close()

	var templates []*NodeTemplate
	for rows.Next() {
		var t NodeTemplate
		var configJSON sql.NullString
		var isPublic int
		err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.Category, &t.NodeType, &configJSON, &isPublic, &t.UseCount, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			s.logger.Warn("failed to scan node template", zap.Error(err))
			continue
		}
		t.IsPublic = isPublic == 1
		if configJSON.Valid && configJSON.String != "" {
			json.Unmarshal([]byte(configJSON.String), &t.NodeConfig)
		}
		templates = append(templates, &t)
	}

	return templates, nil
}

// GetNodeTemplate 获取节点模板
func (s *Service) GetNodeTemplate(id string) (*NodeTemplate, error) {
	var t NodeTemplate
	var configJSON sql.NullString
	var isPublic int

	err := s.db.QueryRow(`
		SELECT id, name, description, category, node_type, node_config, is_public, use_count, created_at, updated_at
		FROM node_templates WHERE id = ?
	`, id).Scan(&t.ID, &t.Name, &t.Description, &t.Category, &t.NodeType, &configJSON, &isPublic, &t.UseCount, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get node template: %w", err)
	}

	t.IsPublic = isPublic == 1
	if configJSON.Valid && configJSON.String != "" {
		json.Unmarshal([]byte(configJSON.String), &t.NodeConfig)
	}

	return &t, nil
}

// UpdateNodeTemplate 更新节点模板
func (s *Service) UpdateNodeTemplate(id string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	query := "UPDATE node_templates SET "
	var setClauses []string
	var args []interface{}

	for key, value := range updates {
		if key == "node_config" {
			configJSON, err := json.Marshal(value)
			if err != nil {
				return fmt.Errorf("failed to marshal node config: %w", err)
			}
			setClauses = append(setClauses, "node_config = ?")
			args = append(args, string(configJSON))
		} else if key == "is_public" {
			setClauses = append(setClauses, "is_public = ?")
			if v, ok := value.(bool); ok {
				args = append(args, v)
			}
		} else {
			setClauses = append(setClauses, fmt.Sprintf("%s = ?", key))
			args = append(args, value)
		}
	}

	query += strings.Join(setClauses, ", ") + " WHERE id = ?"
	args = append(args, id)

	_, err := s.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update node template: %w", err)
	}

	return nil
}

// DeleteNodeTemplate 删除节点模板
func (s *Service) DeleteNodeTemplate(id string) error {
	_, err := s.db.Exec("DELETE FROM node_templates WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete node template: %w", err)
	}
	return nil
}

// UseNodeTemplate 使用节点模板（增加使用次数）
func (s *Service) UseNodeTemplate(id string) (*NodeTemplate, error) {
	template, err := s.GetNodeTemplate(id)
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec("UPDATE node_templates SET use_count = use_count + 1 WHERE id = ?", id)
	if err != nil {
		s.logger.Warn("failed to increment use count", zap.Error(err))
	}

	template.UseCount++
	return template, nil
}

// GetNodeTemplateCategories 获取节点模板分类
func (s *Service) GetNodeTemplateCategories() ([]map[string]interface{}, error) {
	rows, err := s.db.Query(`
		SELECT category, COUNT(*) as count
		FROM node_templates
		GROUP BY category
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}
	defer rows.Close()

	var categories []map[string]interface{}
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			continue
		}
		categories = append(categories, map[string]interface{}{
			"name":  category,
			"count": count,
		})
	}

	return categories, nil
}

// ========== 流程文档生成 ==========

// FlowDocumentation 流程文档
type FlowDocumentation struct {
	FlowID      string            `json:"flow_id"`
	FlowName    string            `json:"flow_name"`
	Description string            `json:"description"`
	FlowType    FlowType          `json:"flow_type"`
	Status      FlowStatus        `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Overview    string            `json:"overview"`
	Nodes       []NodeDoc         `json:"nodes"`
	Variables   VariablesDoc      `json:"variables"`
	Trigger     *TriggerDoc       `json:"trigger,omitempty"`
	FlowChart   string            `json:"flow_chart"`
}

// NodeDoc 节点文档
type NodeDoc struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Inputs      []string `json:"inputs,omitempty"`
	Outputs     []string `json:"outputs,omitempty"`
	Config      string   `json:"config,omitempty"`
	NextNodes   []string `json:"next_nodes,omitempty"`
}

// VariablesDoc 变量文档
type VariablesDoc struct {
	Inputs  []VariableDoc `json:"inputs,omitempty"`
	Outputs []VariableDoc `json:"outputs,omitempty"`
}

// VariableDoc 变量文档项
type VariableDoc struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
}

// TriggerDoc 触发器文档
type TriggerDoc struct {
	Type        string            `json:"type"`
	Description string            `json:"description"`
	Config      map[string]string `json:"config,omitempty"`
}

// GenerateDocumentation 生成流程文档
func (s *Service) GenerateDocumentation(flowID string) (*FlowDocumentation, error) {
	flow, err := s.GetFlow(flowID)
	if err != nil {
		return nil, err
	}

	doc := &FlowDocumentation{
		FlowID:      flow.ID,
		FlowName:    flow.Name,
		Description: flow.Description,
		FlowType:    flow.Type,
		Status:      flow.Status,
		CreatedAt:   flow.CreatedAt,
		UpdatedAt:   flow.UpdatedAt,
	}

	// 生成概述
	doc.Overview = s.generateOverview(flow)

	// 生成节点文档
	doc.Nodes = s.generateNodeDocs(flow)

	// 生成变量文档
	doc.Variables = s.generateVariablesDoc(flow)

	// 生成触发器文档
	if flow.Trigger != nil {
		doc.Trigger = s.generateTriggerDoc(flow.Trigger)
	}

	// 生成流程图
	doc.FlowChart = s.generateFlowChart(flow)

	return doc, nil
}

// generateOverview 生成流程概述
func (s *Service) generateOverview(flow *Flow) string {
	var overview strings.Builder

	overview.WriteString(fmt.Sprintf("## 流程概述\n\n"))
	overview.WriteString(fmt.Sprintf("**流程名称**: %s\n\n", flow.Name))

	if flow.Description != "" {
		overview.WriteString(fmt.Sprintf("**描述**: %s\n\n", flow.Description))
	}

	overview.WriteString(fmt.Sprintf("**类型**: %s\n\n", map[string]string{
		"conversation": "对话流",
		"task":         "任务流",
	}[string(flow.Type)]))

	overview.WriteString(fmt.Sprintf("**状态**: %s\n\n", map[string]string{
		"draft":    "草稿",
		"active":   "已启用",
		"disabled": "已停用",
	}[string(flow.Status)]))

	overview.WriteString(fmt.Sprintf("**节点数量**: %d\n\n", len(flow.Definition.Nodes)))
	overview.WriteString(fmt.Sprintf("**连线数量**: %d\n\n", len(flow.Definition.Edges)))

	return overview.String()
}

// generateNodeDocs 生成节点文档
func (s *Service) generateNodeDocs(flow *Flow) []NodeDoc {
	var docs []NodeDoc

	nodeTypeNames := map[NodeType]string{
		NodeTypeStart:     "开始节点",
		NodeTypeAgent:     "Agent 节点",
		NodeTypeHuman:     "人工节点",
		NodeTypeCondition: "条件分支",
		NodeTypeParallel:  "并行执行",
		NodeTypeLoop:      "循环执行",
		NodeTypeSubFlow:   "子流程",
		NodeTypeWebhook:   "Webhook 等待",
		NodeTypeEnd:       "结束节点",
	}

	// 构建节点连接关系
	nextNodesMap := make(map[string][]string)
	for _, edge := range flow.Definition.Edges {
		nextNodesMap[edge.Source] = append(nextNodesMap[edge.Source], edge.Target)
	}

	for _, node := range flow.Definition.Nodes {
		doc := NodeDoc{
			ID:   node.ID,
			Name: node.Name,
			Type: nodeTypeNames[node.Type],
		}

		// 生成描述
		switch node.Type {
		case NodeTypeAgent:
			if node.AgentID != "" {
				doc.Description = fmt.Sprintf("调用 Agent: %s", node.AgentID)
			}
			if node.Role != "" {
				doc.Description += fmt.Sprintf(", 角色: %s", node.Role)
			}
		case NodeTypeHuman:
			doc.Description = "等待人工输入"
			if node.Prompt != "" {
				doc.Description += fmt.Sprintf(": %s", truncate(node.Prompt, 50))
			}
		case NodeTypeCondition:
			doc.Description = "条件判断分支"
		case NodeTypeParallel:
			doc.Description = "并行执行多个分支"
		case NodeTypeLoop:
			doc.Description = "循环执行"
		case NodeTypeSubFlow:
			doc.Description = "调用子流程"
		case NodeTypeWebhook:
			doc.Description = "等待 Webhook 回调"
		}

		// 输入输出
		if node.Inputs != nil {
			for k := range node.Inputs {
				doc.Inputs = append(doc.Inputs, k)
			}
		}
		if node.Outputs != nil {
			for k := range node.Outputs {
				doc.Outputs = append(doc.Outputs, k)
			}
		}

		// 下游节点
		if nextNodes, ok := nextNodesMap[node.ID]; ok {
			for _, nextID := range nextNodes {
				for _, n := range flow.Definition.Nodes {
					if n.ID == nextID {
						doc.NextNodes = append(doc.NextNodes, n.Name)
						break
					}
				}
			}
		}

		docs = append(docs, doc)
	}

	return docs
}

// generateVariablesDoc 生成变量文档
func (s *Service) generateVariablesDoc(flow *Flow) VariablesDoc {
	var doc VariablesDoc

	// 输入变量
	if flow.Definition.InputVars != nil {
		for name, v := range flow.Definition.InputVars {
			defaultVal := ""
			if v.Default != nil {
				defaultVal = fmt.Sprintf("%v", v.Default)
			}
			doc.Inputs = append(doc.Inputs, VariableDoc{
				Name:        name,
				Type:        v.Type,
				Required:    v.Required,
				Default:     defaultVal,
				Description: v.Description,
			})
		}
	}

	// 输出变量
	if flow.Definition.OutputVars != nil {
		for name, v := range flow.Definition.OutputVars {
			doc.Outputs = append(doc.Outputs, VariableDoc{
				Name:        name,
				Type:        v.Type,
				Required:    v.Required,
				Description: v.Description,
			})
		}
	}

	return doc
}

// generateTriggerDoc 生成触发器文档
func (s *Service) generateTriggerDoc(trigger *TriggerConfig) *TriggerDoc {
	doc := &TriggerDoc{
		Config: make(map[string]string),
	}

	switch trigger.Type {
	case "manual":
		doc.Type = "手动触发"
		doc.Description = "用户手动触发执行"
	case "webhook":
		doc.Type = "Webhook 触发"
		doc.Description = "通过 HTTP 请求触发"
		if trigger.Config != nil {
			if path, ok := trigger.Config["path"].(string); ok {
				doc.Config["path"] = path
			}
			if method, ok := trigger.Config["method"].(string); ok {
				doc.Config["method"] = method
			}
		}
	case "cron":
		doc.Type = "定时触发"
		doc.Description = "按计划定时执行"
		if trigger.Config != nil {
			if cron, ok := trigger.Config["cron"].(string); ok {
				doc.Config["cron"] = cron
			}
		}
	case "event":
		doc.Type = "事件触发"
		doc.Description = "响应系统事件执行"
		if trigger.Config != nil {
			if eventType, ok := trigger.Config["event_type"].(string); ok {
				doc.Config["event_type"] = eventType
			}
		}
	}

	return doc
}

// generateFlowChart 生成流程图（Mermaid 格式）
func (s *Service) generateFlowChart(flow *Flow) string {
	var chart strings.Builder

	chart.WriteString("```mermaid\n")
	chart.WriteString("flowchart TD\n")

	// 添加节点
	for _, node := range flow.Definition.Nodes {
		var label string
		switch node.Type {
		case NodeTypeStart:
			label = fmt.Sprintf("%s([%s])", node.ID, node.Name)
		case NodeTypeEnd:
			label = fmt.Sprintf("%s([%s])", node.ID, node.Name)
		case NodeTypeCondition:
			label = fmt.Sprintf("%s{%s}", node.ID, node.Name)
		default:
			label = fmt.Sprintf("%s[%s]", node.ID, node.Name)
		}
		chart.WriteString(fmt.Sprintf("    %s\n", label))
	}

	chart.WriteString("\n")

	// 添加连线
	for _, edge := range flow.Definition.Edges {
		label := ""
		if edge.Label != "" {
			label = fmt.Sprintf("|%s|", edge.Label)
		}
		chart.WriteString(fmt.Sprintf("    %s -->%s %s\n", edge.Source, label, edge.Target))
	}

	chart.WriteString("```\n")

	return chart.String()
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ========== 测试用例管理 ==========

// CreateTestCase 创建测试用例
func (s *Service) CreateTestCase(tc *FlowTestCase) (*FlowTestCase, error) {
	if tc.ID == "" {
		tc.ID = "tc_" + generateID()
	}
	tc.CreatedAt = time.Now()
	tc.UpdatedAt = time.Now()

	inputJSON, _ := json.Marshal(tc.Input)
	expectedJSON, _ := json.Marshal(tc.Expected)

	_, err := s.db.Exec(`
		INSERT INTO flow_test_cases (id, flow_id, name, description, input, expected, last_run_at, last_status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, tc.ID, tc.FlowID, tc.Name, tc.Description, string(inputJSON), string(expectedJSON), tc.LastRunAt, tc.LastStatus, tc.CreatedAt, tc.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create test case: %w", err)
	}

	return tc, nil
}

// ListTestCases 列出测试用例
func (s *Service) ListTestCases(flowID string) ([]*FlowTestCase, error) {
	query := "SELECT id, flow_id, name, description, input, expected, last_run_at, last_status, created_at, updated_at FROM flow_test_cases WHERE flow_id = ? ORDER BY created_at DESC"
	rows, err := s.db.Query(query, flowID)
	if err != nil {
		return nil, fmt.Errorf("failed to list test cases: %w", err)
	}
	defer rows.Close()

	var cases []*FlowTestCase
	for rows.Next() {
		var tc FlowTestCase
		var inputJSON, expectedJSON sql.NullString
		var lastRunAt sql.NullTime
		err := rows.Scan(&tc.ID, &tc.FlowID, &tc.Name, &tc.Description, &inputJSON, &expectedJSON, &lastRunAt, &tc.LastStatus, &tc.CreatedAt, &tc.UpdatedAt)
		if err != nil {
			continue
		}
		if inputJSON.Valid {
			json.Unmarshal([]byte(inputJSON.String), &tc.Input)
		}
		if expectedJSON.Valid {
			json.Unmarshal([]byte(expectedJSON.String), &tc.Expected)
		}
		if lastRunAt.Valid {
			tc.LastRunAt = &lastRunAt.Time
		}
		cases = append(cases, &tc)
	}

	return cases, nil
}

// GetTestCase 获取测试用例
func (s *Service) GetTestCase(id string) (*FlowTestCase, error) {
	var tc FlowTestCase
	var inputJSON, expectedJSON sql.NullString
	var lastRunAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, flow_id, name, description, input, expected, last_run_at, last_status, created_at, updated_at
		FROM flow_test_cases WHERE id = ?
	`, id).Scan(&tc.ID, &tc.FlowID, &tc.Name, &tc.Description, &inputJSON, &expectedJSON, &lastRunAt, &tc.LastStatus, &tc.CreatedAt, &tc.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get test case: %w", err)
	}

	if inputJSON.Valid {
		json.Unmarshal([]byte(inputJSON.String), &tc.Input)
	}
	if expectedJSON.Valid {
		json.Unmarshal([]byte(expectedJSON.String), &tc.Expected)
	}
	if lastRunAt.Valid {
		tc.LastRunAt = &lastRunAt.Time
	}

	return &tc, nil
}

// UpdateTestCase 更新测试用例
func (s *Service) UpdateTestCase(id string, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	query := "UPDATE flow_test_cases SET "
	var setClauses []string
	var args []interface{}

	for key, value := range updates {
		if key == "input" || key == "expected" {
			jsonBytes, _ := json.Marshal(value)
			setClauses = append(setClauses, fmt.Sprintf("%s = ?", key))
			args = append(args, string(jsonBytes))
		} else {
			setClauses = append(setClauses, fmt.Sprintf("%s = ?", key))
			args = append(args, value)
		}
	}

	query += strings.Join(setClauses, ", ") + " WHERE id = ?"
	args = append(args, id)

	_, err := s.db.Exec(query, args...)
	return err
}

// DeleteTestCase 删除测试用例
func (s *Service) DeleteTestCase(id string) error {
	_, err := s.db.Exec("DELETE FROM flow_test_cases WHERE id = ?", id)
	return err
}

// RunTestCase 执行测试用例
func (s *Service) RunTestCase(id string) (*FlowTestRun, error) {
	tc, err := s.GetTestCase(id)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	run := &FlowTestRun{
		ID:         "tr_" + generateID(),
		TestCaseID: tc.ID,
		FlowID:     tc.FlowID,
		Input:      tc.Input,
		Expected:   tc.Expected,
		CreatedAt:  startTime,
	}

	// 执行流程
	inputJSON, _ := json.Marshal(tc.Input)
	execReq := ExecuteRequest{
		Input: string(inputJSON),
	}

	execResp, err := s.Execute(tc.FlowID, execReq)
	if err != nil {
		run.Status = "error"
		run.Error = err.Error()
	} else if execResp.Status == "failed" {
		run.Status = "error"
		run.Error = "execution failed"
	} else {
		// 解析输出
		if execResp.Output != "" {
			var outputMap map[string]interface{}
			if err := json.Unmarshal([]byte(execResp.Output), &outputMap); err == nil {
				run.Output = outputMap
			} else {
				run.Output = map[string]interface{}{"result": execResp.Output}
			}
		}

		// 比较输出与期望
		if s.compareOutput(run.Output, tc.Expected) {
			run.Status = "passed"
		} else {
			run.Status = "failed"
		}
	}

	run.Duration = time.Since(startTime).Milliseconds()

	// 保存执行记录
	inputJSON2, _ := json.Marshal(run.Input)
	outputJSON, _ := json.Marshal(run.Output)
	expectedJSON, _ := json.Marshal(run.Expected)

	_, err = s.db.Exec(`
		INSERT INTO flow_test_runs (id, test_case_id, flow_id, status, input, output, expected, duration, error, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, run.ID, run.TestCaseID, run.FlowID, run.Status, string(inputJSON2), string(outputJSON), string(expectedJSON), run.Duration, run.Error, run.CreatedAt)

	if err != nil {
		s.logger.Warn("failed to save test run", zap.Error(err))
	}

	// 更新测试用例状态
	now := time.Now()
	s.UpdateTestCase(id, map[string]interface{}{
		"last_run_at":  &now,
		"last_status":  run.Status,
	})

	return run, nil
}

// compareOutput 比较输出与期望
func (s *Service) compareOutput(output, expected map[string]interface{}) bool {
	if expected == nil || len(expected) == 0 {
		return true // 没有期望则默认通过
	}

	for key, expectedValue := range expected {
		outputValue, ok := output[key]
		if !ok {
			return false
		}

		if !s.compareValues(outputValue, expectedValue) {
			return false
		}
	}

	return true
}

// compareValues 比较两个值
func (s *Service) compareValues(a, b interface{}) bool {
	aJSON, _ := json.Marshal(a)
	bJSON, _ := json.Marshal(b)
	return string(aJSON) == string(bJSON)
}

// ListTestRuns 列出测试执行记录
func (s *Service) ListTestRuns(testCaseID string, limit int) ([]*FlowTestRun, error) {
	query := "SELECT id, test_case_id, flow_id, status, input, output, expected, duration, error, created_at FROM flow_test_runs WHERE test_case_id = ? ORDER BY created_at DESC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.Query(query, testCaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to list test runs: %w", err)
	}
	defer rows.Close()

	var runs []*FlowTestRun
	for rows.Next() {
		var run FlowTestRun
		var inputJSON, outputJSON, expectedJSON, errMsg sql.NullString
		err := rows.Scan(&run.ID, &run.TestCaseID, &run.FlowID, &run.Status, &inputJSON, &outputJSON, &expectedJSON, &run.Duration, &errMsg, &run.CreatedAt)
		if err != nil {
			continue
		}
		if inputJSON.Valid {
			json.Unmarshal([]byte(inputJSON.String), &run.Input)
		}
		if outputJSON.Valid {
			json.Unmarshal([]byte(outputJSON.String), &run.Output)
		}
		if expectedJSON.Valid {
			json.Unmarshal([]byte(expectedJSON.String), &run.Expected)
		}
		if errMsg.Valid {
			run.Error = errMsg.String
		}
		runs = append(runs, &run)
	}

	return runs, nil
}

// GetTestStats 获取测试统计
func (s *Service) GetTestStats(flowID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总用例数
	var totalCases int
	s.db.QueryRow("SELECT COUNT(*) FROM flow_test_cases WHERE flow_id = ?", flowID).Scan(&totalCases)
	stats["total_cases"] = totalCases

	// 通过/失败数
	var passed, failed int
	s.db.QueryRow("SELECT COUNT(*) FROM flow_test_cases WHERE flow_id = ? AND last_status = 'passed'", flowID).Scan(&passed)
	s.db.QueryRow("SELECT COUNT(*) FROM flow_test_cases WHERE flow_id = ? AND last_status = 'failed'", flowID).Scan(&failed)
	stats["passed"] = passed
	stats["failed"] = failed

	// 通过率
	if totalCases > 0 {
		stats["pass_rate"] = float64(passed) / float64(totalCases) * 100
	} else {
		stats["pass_rate"] = 0.0
	}

	return stats, nil
}