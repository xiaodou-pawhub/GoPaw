package orchestration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Service 编排服务
type Service struct {
	db     *sql.DB
	engine *Engine
}

// NewService 创建编排服务
func NewService(db *sql.DB, engine *Engine) *Service {
	return &Service{
		db:     db,
		engine: engine,
	}
}

// SetDB 设置数据库连接（用于延迟初始化）
func (s *Service) SetDB(db *sql.DB) {
	s.db = db
}

// CreateOrchestration 创建编排
func (s *Service) CreateOrchestration(ctx context.Context, req CreateOrchestrationRequest) (*Orchestration, error) {
	orch := &Orchestration{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
		Status:      "draft",
		Definition:  req.Definition,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 如果没有指定起始节点，使用第一个节点
	if orch.Definition.StartNodeID == "" && len(orch.Definition.Nodes) > 0 {
		orch.Definition.StartNodeID = orch.Definition.Nodes[0].ID
	}

	defJSON, err := json.Marshal(orch.Definition)
	if err != nil {
		return nil, err
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO orchestrations (id, name, description, status, definition, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, orch.ID, orch.Name, orch.Description, orch.Status, string(defJSON), orch.CreatedAt, orch.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create orchestration: %w", err)
	}

	return orch, nil
}

// GetOrchestration 获取编排
func (s *Service) GetOrchestration(ctx context.Context, id string) (*Orchestration, error) {
	var orch Orchestration
	var defStr string

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, status, definition, created_at, updated_at
		FROM orchestrations WHERE id = ?
	`, id).Scan(&orch.ID, &orch.Name, &orch.Description, &orch.Status, &defStr, &orch.CreatedAt, &orch.UpdatedAt)

	if err != nil {
		return nil, err
	}

	if defStr != "" {
		json.Unmarshal([]byte(defStr), &orch.Definition)
	}

	return &orch, nil
}

// ListOrchestrations 列出编排
func (s *Service) ListOrchestrations(ctx context.Context) ([]Orchestration, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, status, definition, created_at, updated_at
		FROM orchestrations ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orchestrations := make([]Orchestration, 0)
	for rows.Next() {
		var orch Orchestration
		var defStr string
		err := rows.Scan(&orch.ID, &orch.Name, &orch.Description, &orch.Status, &defStr, &orch.CreatedAt, &orch.UpdatedAt)
		if err != nil {
			continue
		}

		if defStr != "" {
			json.Unmarshal([]byte(defStr), &orch.Definition)
		}

		orchestrations = append(orchestrations, orch)
	}

	return orchestrations, rows.Err()
}

// UpdateOrchestration 更新编排
func (s *Service) UpdateOrchestration(ctx context.Context, id string, req UpdateOrchestrationRequest) error {
	// 获取现有编排
	existing, err := s.GetOrchestration(ctx, id)
	if err != nil {
		return err
	}

	// 更新字段
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Status != "" {
		existing.Status = req.Status
	}
	// 如果 Definition 不为空（有 Nodes 或 Edges），则更新
	// 注意：允许清空节点（空数组）
	if req.Definition.Nodes != nil || req.Definition.Edges != nil {
		if req.Definition.Nodes != nil {
			existing.Definition.Nodes = req.Definition.Nodes
		}
		if req.Definition.Edges != nil {
			existing.Definition.Edges = req.Definition.Edges
		}
		if req.Definition.StartNodeID != "" {
			existing.Definition.StartNodeID = req.Definition.StartNodeID
		}
		if req.Definition.Variables != nil {
			existing.Definition.Variables = req.Definition.Variables
		}
	}

	existing.UpdatedAt = time.Now()

	defJSON, err := json.Marshal(existing.Definition)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE orchestrations SET
			name = COALESCE(NULLIF(?, ''), name),
			description = COALESCE(NULLIF(?, ''), description),
			status = COALESCE(NULLIF(?, ''), status),
			definition = ?,
			updated_at = ?
		WHERE id = ?
	`, req.Name, req.Description, req.Status, string(defJSON), existing.UpdatedAt, id)

	return err
}

// DeleteOrchestration 删除编排
func (s *Service) DeleteOrchestration(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM orchestrations WHERE id = ?", id)
	return err
}

// ExecuteOrchestration 执行编排
func (s *Service) ExecuteOrchestration(ctx context.Context, id string, req ExecuteRequest) (*ExecutionContext, error) {
	return s.engine.Execute(ctx, id, req.Input, req.Variables)
}

// GetExecution 获取执行记录
func (s *Service) GetExecution(ctx context.Context, executionID string) (*ExecutionContext, error) {
	return s.engine.GetExecution(executionID)
}

// ListExecutions 列出执行记录
func (s *Service) ListExecutions(ctx context.Context, orchestrationID string) ([]OrchestrationExecution, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, orchestration_id, status, input, output, variables, current_node_id, started_at, completed_at
		FROM orchestration_executions
		WHERE orchestration_id = ?
		ORDER BY started_at DESC
	`, orchestrationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	executions := make([]OrchestrationExecution, 0)
	for rows.Next() {
		var exec OrchestrationExecution
		var variablesStr string
		err := rows.Scan(&exec.ID, &exec.OrchestrationID, &exec.Status, &exec.Input, &exec.Output,
			&variablesStr, &exec.CurrentNodeID, &exec.StartedAt, &exec.CompletedAt)
		if err != nil {
			continue
		}

		if variablesStr != "" {
			json.Unmarshal([]byte(variablesStr), &exec.Variables)
		}

		executions = append(executions, exec)
	}

	return executions, rows.Err()
}

// SubmitHumanInput 提交人工输入
func (s *Service) SubmitHumanInput(ctx context.Context, executionID string, input string) error {
	return s.engine.SubmitHumanInput(executionID, input)
}

// GetExecutionMessages 获取执行消息
func (s *Service) GetExecutionMessages(ctx context.Context, executionID string) ([]ExecutionMessage, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, execution_id, from_node_id, to_node_id, message_type, content, created_at
		FROM execution_messages
		WHERE execution_id = ?
		ORDER BY created_at ASC
	`, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]ExecutionMessage, 0)
	for rows.Next() {
		var msg ExecutionMessage
		err := rows.Scan(&msg.ID, &msg.ExecutionID, &msg.FromNodeID, &msg.ToNodeID,
			&msg.MessageType, &msg.Content, &msg.CreatedAt)
		if err != nil {
			continue
		}
		messages = append(messages, msg)
	}

	return messages, rows.Err()
}

// ValidateOrchestration 验证编排定义
func (s *Service) ValidateOrchestration(def OrchestrationDefinition) error {
	// 检查是否有节点
	if len(def.Nodes) == 0 {
		return fmt.Errorf("orchestration must have at least one node")
	}

	// 检查是否有起始节点
	if def.StartNodeID == "" {
		return fmt.Errorf("start node must be specified")
	}

	// 检查起始节点是否存在
	startNodeExists := false
	for _, node := range def.Nodes {
		if node.ID == def.StartNodeID {
			startNodeExists = true
			break
		}
	}
	if !startNodeExists {
		return fmt.Errorf("start node not found: %s", def.StartNodeID)
	}

	// 检查节点类型是否有效
	validTypes := map[string]bool{
		"agent":     true,
		"human":     true,
		"condition": true,
		"workflow":  true,
		"end":       true,
	}

	for _, node := range def.Nodes {
		if !validTypes[node.Type] {
			return fmt.Errorf("invalid node type: %s", node.Type)
		}

		// Agent 节点必须有 AgentID
		if node.Type == "agent" && node.AgentID == "" {
			return fmt.Errorf("agent node must have agent_id: %s", node.ID)
		}
	}

	// 检查连线是否有效
	nodeIDs := make(map[string]bool)
	for _, node := range def.Nodes {
		nodeIDs[node.ID] = true
	}

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
