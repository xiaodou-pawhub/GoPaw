// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package flow

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// FlowTemplate 流程模板
type FlowTemplate struct {
	ID          string         `json:"id" db:"id"`
	Name        string         `json:"name" db:"name"`
	Description string         `json:"description" db:"description"`
	Category    string         `json:"category" db:"category"`       // 分类：customer_service, automation, analysis, etc.
	Tags        []string       `json:"tags" db:"tags"`               // 标签
	Definition  FlowDefinition `json:"definition" db:"definition"`   // 流程定义
	Trigger     *TriggerConfig `json:"trigger,omitempty" db:"trigger"` // 触发器配置
	Icon        string         `json:"icon" db:"icon"`               // 图标名称
	Author      string         `json:"author" db:"author"`           // 作者
	IsPublic    bool           `json:"is_public" db:"is_public"`     // 是否公开
	UseCount    int            `json:"use_count" db:"use_count"`     // 使用次数
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// TemplateCategory 模板分类
type TemplateCategory struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Category    string         `json:"category"`
	Tags        []string       `json:"tags"`
	Definition  FlowDefinition `json:"definition"`
	Trigger     *TriggerConfig `json:"trigger,omitempty"`
	Icon        string         `json:"icon"`
	IsPublic    bool           `json:"is_public"`
}

// TemplateService 模板服务
type TemplateService struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewTemplateService 创建模板服务
func NewTemplateService(db *sql.DB, logger *zap.Logger) *TemplateService {
	return &TemplateService{db: db, logger: logger.Named("template")}
}

// InitTemplateSchema 初始化模板表
func InitTemplateSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS flow_templates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			category TEXT DEFAULT 'general',
			tags TEXT,
			definition TEXT NOT NULL,
			trigger TEXT,
			icon TEXT,
			author TEXT,
			is_public INTEGER DEFAULT 1,
			use_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// 创建索引
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_templates_category ON flow_templates(category)`,
		`CREATE INDEX IF NOT EXISTS idx_templates_public ON flow_templates(is_public)`,
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return err
		}
	}

	return nil
}

// GetCategories 获取模板分类列表
func GetCategories() []TemplateCategory {
	return []TemplateCategory{
		{ID: "general", Name: "通用", Description: "通用流程模板", Icon: "LayoutGridIcon"},
		{ID: "customer_service", Name: "客服", Description: "客服对话流程", Icon: "HeadphonesIcon"},
		{ID: "automation", Name: "自动化", Description: "自动化任务流程", Icon: "ZapIcon"},
		{ID: "analysis", Name: "分析", Description: "数据分析流程", Icon: "BarChartIcon"},
		{ID: "approval", Name: "审批", Description: "审批流程", Icon: "CheckCircleIcon"},
		{ID: "notification", Name: "通知", Description: "通知提醒流程", Icon: "BellIcon"},
	}
}

// CreateTemplate 创建模板
func (s *TemplateService) CreateTemplate(req CreateTemplateRequest, author string) (*FlowTemplate, error) {
	id := generateID()

	// 序列化定义
	defBytes, err := json.Marshal(req.Definition)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal definition: %w", err)
	}

	// 序列化标签
	tagsBytes, _ := json.Marshal(req.Tags)

	// 序列化触发器
	var triggerBytes []byte
	if req.Trigger != nil {
		triggerBytes, _ = json.Marshal(req.Trigger)
	}

	now := time.Now()
	_, err = s.db.Exec(`
		INSERT INTO flow_templates (id, name, description, category, tags, definition, trigger, icon, author, is_public, use_count, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?)
	`, id, req.Name, req.Description, req.Category, string(tagsBytes), string(defBytes), string(triggerBytes), req.Icon, author, req.IsPublic, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return &FlowTemplate{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		Tags:        req.Tags,
		Definition:  req.Definition,
		Trigger:     req.Trigger,
		Icon:        req.Icon,
		Author:      author,
		IsPublic:    req.IsPublic,
		UseCount:    0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// ListTemplates 列出模板
func (s *TemplateService) ListTemplates(category string, publicOnly bool) ([]*FlowTemplate, error) {
	query := `SELECT id, name, description, category, tags, definition, trigger, icon, author, is_public, use_count, created_at, updated_at FROM flow_templates`
	args := []interface{}{}
	conditions := []string{}

	if category != "" {
		conditions = append(conditions, "category = ?")
		args = append(args, category)
	}

	if publicOnly {
		conditions = append(conditions, "is_public = 1")
	}

	if len(conditions) > 0 {
		query += " WHERE "
		for i, c := range conditions {
			if i > 0 {
				query += " AND "
			}
			query += c
		}
	}

	query += " ORDER BY use_count DESC, created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []*FlowTemplate
	for rows.Next() {
		t := &FlowTemplate{}
		var tagsJSON, defJSON, triggerJSON sql.NullString
		var desc, icon, author sql.NullString

		err := rows.Scan(&t.ID, &t.Name, &desc, &t.Category, &tagsJSON, &defJSON, &triggerJSON, &icon, &author, &t.IsPublic, &t.UseCount, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			continue
		}

		if desc.Valid {
			t.Description = desc.String
		}
		if icon.Valid {
			t.Icon = icon.String
		}
		if author.Valid {
			t.Author = author.String
		}

		if tagsJSON.Valid {
			json.Unmarshal([]byte(tagsJSON.String), &t.Tags)
		}
		if defJSON.Valid {
			json.Unmarshal([]byte(defJSON.String), &t.Definition)
		}
		if triggerJSON.Valid && triggerJSON.String != "" {
			var trigger TriggerConfig
			if err := json.Unmarshal([]byte(triggerJSON.String), &trigger); err == nil {
				t.Trigger = &trigger
			}
		}

		templates = append(templates, t)
	}

	return templates, nil
}

// GetTemplate 获取模板
func (s *TemplateService) GetTemplate(id string) (*FlowTemplate, error) {
	t := &FlowTemplate{}
	var tagsJSON, defJSON, triggerJSON sql.NullString
	var desc, icon, author sql.NullString

	err := s.db.QueryRow(`
		SELECT id, name, description, category, tags, definition, trigger, icon, author, is_public, use_count, created_at, updated_at
		FROM flow_templates WHERE id = ?
	`, id).Scan(&t.ID, &t.Name, &desc, &t.Category, &tagsJSON, &defJSON, &triggerJSON, &icon, &author, &t.IsPublic, &t.UseCount, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("template not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	if desc.Valid {
		t.Description = desc.String
	}
	if icon.Valid {
		t.Icon = icon.String
	}
	if author.Valid {
		t.Author = author.String
	}

	if tagsJSON.Valid {
		json.Unmarshal([]byte(tagsJSON.String), &t.Tags)
	}
	if defJSON.Valid {
		json.Unmarshal([]byte(defJSON.String), &t.Definition)
	}
	if triggerJSON.Valid && triggerJSON.String != "" {
		var trigger TriggerConfig
		if err := json.Unmarshal([]byte(triggerJSON.String), &trigger); err == nil {
			t.Trigger = &trigger
		}
	}

	return t, nil
}

// UseTemplate 使用模板（创建流程）
func (s *TemplateService) UseTemplate(templateID string, flowService *Service, req CreateFlowRequest) (*Flow, error) {
	template, err := s.GetTemplate(templateID)
	if err != nil {
		return nil, err
	}

	// 增加使用次数
	s.db.Exec(`UPDATE flow_templates SET use_count = use_count + 1 WHERE id = ?`, templateID)

	// 创建流程
	if req.Name == "" {
		req.Name = template.Name
	}
	if req.Description == "" {
		req.Description = template.Description
	}
	if req.Type == "" {
		req.Type = FlowTypeConversation
	}
	req.Definition = template.Definition

	return flowService.CreateFlow(req)
}

// DeleteTemplate 删除模板
func (s *TemplateService) DeleteTemplate(id string) error {
	result, err := s.db.Exec(`DELETE FROM flow_templates WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("template not found: %s", id)
	}

	return nil
}

// SeedDefaultTemplates 填充默认模板
func (s *TemplateService) SeedDefaultTemplates() error {
	// 检查是否已有模板
	var count int
	s.db.QueryRow(`SELECT COUNT(*) FROM flow_templates`).Scan(&count)
	if count > 0 {
		return nil
	}

	defaultTemplates := []CreateTemplateRequest{
		{
			Name:        "智能客服对话",
			Description: "意图识别 → 分支处理 → 统一回复",
			Category:    "customer_service",
			Tags:        []string{"客服", "对话", "意图识别"},
			Icon:        "HeadphonesIcon",
			IsPublic:    true,
			Definition: FlowDefinition{
				Nodes: []FlowNode{
					{ID: "start_1", Type: NodeTypeStart, Name: "开始", Position: Position{X: 250, Y: 30}},
					{ID: "agent_1", Type: NodeTypeAgent, Name: "意图识别", Position: Position{X: 250, Y: 120}, Config: map[string]interface{}{"prompt": "分析用户意图，返回意图类型：查询、投诉、建议、其他"}},
					{ID: "condition_1", Type: NodeTypeCondition, Name: "意图判断", Position: Position{X: 250, Y: 220}},
					{ID: "agent_query", Type: NodeTypeAgent, Name: "查询处理", Position: Position{X: 100, Y: 320}},
					{ID: "agent_complaint", Type: NodeTypeAgent, Name: "投诉处理", Position: Position{X: 250, Y: 320}},
					{ID: "agent_suggest", Type: NodeTypeAgent, Name: "建议处理", Position: Position{X: 400, Y: 320}},
					{ID: "agent_reply", Type: NodeTypeAgent, Name: "统一回复", Position: Position{X: 250, Y: 420}},
					{ID: "end_1", Type: NodeTypeEnd, Name: "结束", Position: Position{X: 250, Y: 520}},
				},
				Edges: []FlowEdge{
					{ID: "e1", Source: "start_1", Target: "agent_1"},
					{ID: "e2", Source: "agent_1", Target: "condition_1"},
					{ID: "e3", Source: "condition_1", Target: "agent_query", Label: "查询"},
					{ID: "e4", Source: "condition_1", Target: "agent_complaint", Label: "投诉"},
					{ID: "e5", Source: "condition_1", Target: "agent_suggest", Label: "建议"},
					{ID: "e6", Source: "agent_query", Target: "agent_reply"},
					{ID: "e7", Source: "agent_complaint", Target: "agent_reply"},
					{ID: "e8", Source: "agent_suggest", Target: "agent_reply"},
					{ID: "e9", Source: "agent_reply", Target: "end_1"},
				},
				StartNodeID: "start_1",
			},
		},
		{
			Name:        "审批流程",
			Description: "提交申请 → 主管审批 → 结果通知",
			Category:    "approval",
			Tags:        []string{"审批", "流程"},
			Icon:        "CheckCircleIcon",
			IsPublic:    true,
			Definition: FlowDefinition{
				Nodes: []FlowNode{
					{ID: "start_1", Type: NodeTypeStart, Name: "开始", Position: Position{X: 250, Y: 30}},
					{ID: "agent_1", Type: NodeTypeAgent, Name: "提交申请", Position: Position{X: 250, Y: 120}},
					{ID: "human_1", Type: NodeTypeHuman, Name: "主管审批", Position: Position{X: 250, Y: 220}, Config: map[string]interface{}{"options": []string{"批准", "驳回"}}},
					{ID: "condition_1", Type: NodeTypeCondition, Name: "审批结果", Position: Position{X: 250, Y: 320}},
					{ID: "agent_pass", Type: NodeTypeAgent, Name: "通过处理", Position: Position{X: 150, Y: 420}},
					{ID: "agent_reject", Type: NodeTypeAgent, Name: "驳回处理", Position: Position{X: 350, Y: 420}},
					{ID: "end_1", Type: NodeTypeEnd, Name: "结束", Position: Position{X: 250, Y: 520}},
				},
				Edges: []FlowEdge{
					{ID: "e1", Source: "start_1", Target: "agent_1"},
					{ID: "e2", Source: "agent_1", Target: "human_1"},
					{ID: "e3", Source: "human_1", Target: "condition_1"},
					{ID: "e4", Source: "condition_1", Target: "agent_pass", Label: "批准"},
					{ID: "e5", Source: "condition_1", Target: "agent_reject", Label: "驳回"},
					{ID: "e6", Source: "agent_pass", Target: "end_1"},
					{ID: "e7", Source: "agent_reject", Target: "end_1"},
				},
				StartNodeID: "start_1",
			},
		},
		{
			Name:        "数据分析流程",
			Description: "数据收集 → 分析处理 → 生成报告",
			Category:    "analysis",
			Tags:        []string{"分析", "数据", "报告"},
			Icon:        "BarChartIcon",
			IsPublic:    true,
			Definition: FlowDefinition{
				Nodes: []FlowNode{
					{ID: "start_1", Type: NodeTypeStart, Name: "开始", Position: Position{X: 250, Y: 30}},
					{ID: "agent_1", Type: NodeTypeAgent, Name: "数据收集", Position: Position{X: 250, Y: 120}},
					{ID: "agent_2", Type: NodeTypeAgent, Name: "数据清洗", Position: Position{X: 250, Y: 220}},
					{ID: "agent_3", Type: NodeTypeAgent, Name: "分析处理", Position: Position{X: 250, Y: 320}},
					{ID: "agent_4", Type: NodeTypeAgent, Name: "生成报告", Position: Position{X: 250, Y: 420}},
					{ID: "end_1", Type: NodeTypeEnd, Name: "结束", Position: Position{X: 250, Y: 520}},
				},
				Edges: []FlowEdge{
					{ID: "e1", Source: "start_1", Target: "agent_1"},
					{ID: "e2", Source: "agent_1", Target: "agent_2"},
					{ID: "e3", Source: "agent_2", Target: "agent_3"},
					{ID: "e4", Source: "agent_3", Target: "agent_4"},
					{ID: "e5", Source: "agent_4", Target: "end_1"},
				},
				StartNodeID: "start_1",
			},
		},
		{
			Name:        "并行处理流程",
			Description: "并行执行多个任务 → 汇总结果",
			Category:    "automation",
			Tags:        []string{"并行", "自动化"},
			Icon:        "ZapIcon",
			IsPublic:    true,
			Definition: FlowDefinition{
				Nodes: []FlowNode{
					{ID: "start_1", Type: NodeTypeStart, Name: "开始", Position: Position{X: 250, Y: 30}},
					{ID: "parallel_1", Type: NodeTypeParallel, Name: "并行处理", Position: Position{X: 250, Y: 120}},
					{ID: "agent_1", Type: NodeTypeAgent, Name: "任务A", Position: Position{X: 100, Y: 220}},
					{ID: "agent_2", Type: NodeTypeAgent, Name: "任务B", Position: Position{X: 250, Y: 220}},
					{ID: "agent_3", Type: NodeTypeAgent, Name: "任务C", Position: Position{X: 400, Y: 220}},
					{ID: "agent_4", Type: NodeTypeAgent, Name: "汇总结果", Position: Position{X: 250, Y: 320}},
					{ID: "end_1", Type: NodeTypeEnd, Name: "结束", Position: Position{X: 250, Y: 420}},
				},
				Edges: []FlowEdge{
					{ID: "e1", Source: "start_1", Target: "parallel_1"},
					{ID: "e2", Source: "parallel_1", Target: "agent_1"},
					{ID: "e3", Source: "parallel_1", Target: "agent_2"},
					{ID: "e4", Source: "parallel_1", Target: "agent_3"},
					{ID: "e5", Source: "agent_1", Target: "agent_4"},
					{ID: "e6", Source: "agent_2", Target: "agent_4"},
					{ID: "e7", Source: "agent_3", Target: "agent_4"},
					{ID: "e8", Source: "agent_4", Target: "end_1"},
				},
				StartNodeID: "start_1",
			},
		},
		{
			Name:        "重试流程",
			Description: "执行任务 → 失败重试 → 成功结束",
			Category:    "automation",
			Tags:        []string{"重试", "容错"},
			Icon:        "RefreshCwIcon",
			IsPublic:    true,
			Definition: FlowDefinition{
				Nodes: []FlowNode{
					{ID: "start_1", Type: NodeTypeStart, Name: "开始", Position: Position{X: 250, Y: 30}},
					{ID: "loop_1", Type: NodeTypeLoop, Name: "重试循环", Position: Position{X: 250, Y: 120}, Config: map[string]interface{}{"max_loop": 3}},
					{ID: "agent_1", Type: NodeTypeAgent, Name: "执行任务", Position: Position{X: 250, Y: 220}},
					{ID: "condition_1", Type: NodeTypeCondition, Name: "执行结果", Position: Position{X: 250, Y: 320}},
					{ID: "end_1", Type: NodeTypeEnd, Name: "成功结束", Position: Position{X: 250, Y: 420}},
				},
				Edges: []FlowEdge{
					{ID: "e1", Source: "start_1", Target: "loop_1"},
					{ID: "e2", Source: "loop_1", Target: "agent_1"},
					{ID: "e3", Source: "agent_1", Target: "condition_1"},
					{ID: "e4", Source: "condition_1", Target: "end_1", Label: "成功"},
					{ID: "e5", Source: "condition_1", Target: "loop_1", Label: "失败"},
				},
				StartNodeID: "start_1",
			},
		},
	}

	for _, tpl := range defaultTemplates {
		_, err := s.CreateTemplate(tpl, "system")
		if err != nil {
			s.logger.Warn("failed to seed template", zap.String("name", tpl.Name), zap.Error(err))
		}
	}

	s.logger.Info("seeded default templates", zap.Int("count", len(defaultTemplates)))
	return nil
}