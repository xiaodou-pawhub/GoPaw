// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package flow

import (
	"database/sql"
	"encoding/json"
	"time"
)

// FlowType 流程类型
type FlowType string

const (
	FlowTypeConversation FlowType = "conversation" // 对话流：支持人工介入、状态保持
	FlowTypeTask         FlowType = "task"         // 任务流：自动化执行、支持触发器
)

// FlowStatus 流程状态
type FlowStatus string

const (
	FlowStatusDraft    FlowStatus = "draft"
	FlowStatusActive   FlowStatus = "active"
	FlowStatusDisabled FlowStatus = "disabled"
)

// ExecutionStatus 执行状态
type ExecutionStatus string

const (
	ExecutionStatusPending   ExecutionStatus = "pending"
	ExecutionStatusRunning   ExecutionStatus = "running"
	ExecutionStatusWaiting   ExecutionStatus = "waiting" // 等待人工输入
	ExecutionStatusCompleted ExecutionStatus = "completed"
	ExecutionStatusFailed    ExecutionStatus = "failed"
	ExecutionStatusCancelled ExecutionStatus = "cancelled"
)

// NodeType 节点类型
type NodeType string

const (
	NodeTypeStart     NodeType = "start"     // 开始节点
	NodeTypeAgent     NodeType = "agent"     // Agent 节点
	NodeTypeHuman     NodeType = "human"     // 人工节点
	NodeTypeCondition NodeType = "condition" // 条件分支
	NodeTypeParallel  NodeType = "parallel"  // 并行执行
	NodeTypeLoop      NodeType = "loop"      // 循环执行
	NodeTypeSubFlow   NodeType = "subflow"   // 子流程
	NodeTypeWebhook   NodeType = "webhook"   // Webhook 等待
	NodeTypeEnd       NodeType = "end"       // 结束节点
	// 新增节点类型
	NodeTypeTransform NodeType = "transform" // 数据转换
	NodeTypeHTTP      NodeType = "http"      // HTTP 请求
	NodeTypeDelay     NodeType = "delay"     // 延迟等待
	NodeTypeSwitch    NodeType = "switch"    // 多路分支
	NodeTypeMerge     NodeType = "merge"     // 分支合并
)

// Flow 流程定义
type Flow struct {
	ID          string         `json:"id" db:"id"`
	Name        string         `json:"name" db:"name"`
	Description string         `json:"description" db:"description"`
	Type        FlowType       `json:"type" db:"type"`                   // conversation/task
	Definition  FlowDefinition `json:"definition" db:"definition"`       // 流程定义
	Trigger     *TriggerConfig `json:"trigger,omitempty" db:"trigger"`   // 触发配置
	Status      FlowStatus     `json:"status" db:"status"`               // draft/active/disabled
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at" db:"updated_at"`
}

// FlowDefinition 流程定义结构
type FlowDefinition struct {
	Nodes       []FlowNode           `json:"nodes"`
	Edges       []FlowEdge           `json:"edges"`
	Variables   map[string]Variable  `json:"variables,omitempty"`   // 全局变量（已废弃，保留兼容）
	InputVars   map[string]Variable  `json:"input_vars,omitempty"`  // 流程输入变量
	OutputVars  map[string]Variable  `json:"output_vars,omitempty"` // 流程输出变量
	StartNodeID string               `json:"start_node_id"`         // 起始节点 ID
}

// Value 实现 driver.Valuer
func (d FlowDefinition) Value() (interface{}, error) {
	return json.Marshal(d)
}

// Scan 实现 sql.Scanner
func (d *FlowDefinition) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, d)
	case string:
		return json.Unmarshal([]byte(v), d)
	default:
		return nil
	}
}

// FlowNode 流程节点
type FlowNode struct {
	ID       string                 `json:"id"`
	Type     NodeType               `json:"type"`
	Name     string                 `json:"name"`
	AgentID  string                 `json:"agent_id,omitempty"`  // Agent 节点引用
	Role     string                 `json:"role,omitempty"`      // 角色描述
	Prompt   string                 `json:"prompt,omitempty"`    // Prompt 模板
	Config   map[string]interface{} `json:"config,omitempty"`    // 节点配置
	Inputs   map[string]string      `json:"inputs,omitempty"`    // 输入映射: 本节点变量名 -> 来源表达式
	Outputs  map[string]string      `json:"outputs,omitempty"`   // 输出映射: 输出名 -> 存储变量名
	Position Position               `json:"position"`            // 画布位置
	// 重试配置
	RetryConfig *RetryConfig        `json:"retry_config,omitempty"` // 重试配置
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries   int    `json:"max_retries,omitempty"`   // 最大重试次数
	RetryDelay   int    `json:"retry_delay,omitempty"`   // 重试延迟（毫秒）
	RetryOn      string `json:"retry_on,omitempty"`      // 重试条件: always, error, timeout
	FallbackNode string `json:"fallback_node,omitempty"` // 失败后跳转的节点
}

// Position 画布位置
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// FlowEdge 流程连线
type FlowEdge struct {
	ID         string         `json:"id"`
	Source     string         `json:"source"`               // 源节点 ID
	Target     string         `json:"target"`               // 目标节点 ID
	Label      string         `json:"label,omitempty"`      // 显示标签
	Condition  *EdgeCondition `json:"condition,omitempty"`  // 条件配置
	Transform  *Transform     `json:"transform,omitempty"`  // 消息转换
	SourceType string         `json:"source_type,omitempty"` // 源端口（条件节点多出口）
}

// EdgeCondition 连线条件
type EdgeCondition struct {
	Type       string `json:"type"`                  // expression/intent/llm/always
	Expression string `json:"expression,omitempty"`  // 表达式
	Intent     string `json:"intent,omitempty"`      // 意图匹配
	LLMQuery   string `json:"llm_query,omitempty"`   // LLM 判断提示
}

// Transform 消息转换
type Transform struct {
	Template string `json:"template,omitempty"` // 输出模板
}

// Variable 变量定义
type Variable struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`        // string/number/boolean/object/array
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Description string      `json:"description,omitempty"`
}

// TriggerConfig 触发配置
type TriggerConfig struct {
	Type   string                 `json:"type"`             // manual/cron/webhook/event
	Config map[string]interface{} `json:"config,omitempty"` // 触发器配置
}

// Value 实现 driver.Valuer
func (t TriggerConfig) Value() (interface{}, error) {
	return json.Marshal(t)
}

// Scan 实现 sql.Scanner
func (t *TriggerConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, t)
	case string:
		return json.Unmarshal([]byte(v), t)
	default:
		return nil
	}
}

// Execution 流程执行实例
type Execution struct {
	ID           string                 `json:"id" db:"id"`
	FlowID       string                 `json:"flow_id" db:"flow_id"`
	Status       ExecutionStatus        `json:"status" db:"status"`
	Trigger      string                 `json:"trigger" db:"trigger"`           // 触发来源: manual/webhook/cron
	Input        string                 `json:"input" db:"input"`
	Output       string                 `json:"output" db:"output"`
	Variables    map[string]interface{} `json:"variables" db:"variables"`
	CurrentNode  string                 `json:"current_node" db:"current_node"`
	Context      map[string]interface{} `json:"context" db:"context"`      // 执行上下文
	History      []ExecutionStep        `json:"history" db:"history"`      // 执行历史
	StartedAt    time.Time              `json:"started_at" db:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at" db:"completed_at"`
	Error        string                 `json:"error,omitempty" db:"error"`
	// 调试模式
	DebugMode    bool                   `json:"debug_mode" db:"debug_mode"`       // 是否调试模式
	Breakpoints  []string               `json:"breakpoints" db:"breakpoints"`     // 断点节点 ID 列表
	StepMode     bool                   `json:"step_mode" db:"step_mode"`         // 单步执行模式
}

// ExecutionStep 执行步骤记录
type ExecutionStep struct {
	NodeID    string                 `json:"node_id"`
	NodeType  NodeType               `json:"node_type"`
	Status    ExecutionStatus        `json:"status"`
	Input     map[string]interface{} `json:"input,omitempty"`
	Output    map[string]interface{} `json:"output,omitempty"`
	Error     string                 `json:"error,omitempty"`
	StartedAt time.Time              `json:"started_at"`
	EndedAt   *time.Time             `json:"ended_at,omitempty"`
}

// CreateFlowRequest 创建流程请求
type CreateFlowRequest struct {
	ID          string         `json:"id" binding:"required"`
	Name        string         `json:"name" binding:"required"`
	Description string         `json:"description"`
	Type        FlowType       `json:"type"`                    // 默认 conversation
	Definition  FlowDefinition `json:"definition"`
	Trigger     *TriggerConfig `json:"trigger,omitempty"`
}

// UpdateFlowRequest 更新流程请求
type UpdateFlowRequest struct {
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Type        FlowType       `json:"type,omitempty"`
	Definition  FlowDefinition `json:"definition,omitempty"`
	Trigger     *TriggerConfig `json:"trigger,omitempty"`
	Status      FlowStatus     `json:"status,omitempty"`
}

// ExecuteRequest 执行请求
type ExecuteRequest struct {
	Input       string                 `json:"input"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"` // 额外上下文
	Async       bool                   `json:"async,omitempty"`   // 是否异步执行
	Trigger     string                 `json:"trigger,omitempty"` // 触发来源: manual/webhook/cron
	// 调试模式
	DebugMode   bool                   `json:"debug_mode,omitempty"`   // 是否调试模式
	Breakpoints []string               `json:"breakpoints,omitempty"`  // 断点节点 ID 列表
	StepMode    bool                   `json:"step_mode,omitempty"`    // 单步执行模式
}

// ExecuteResponse 执行响应
type ExecuteResponse struct {
	ExecutionID string `json:"execution_id"`
	Status      string `json:"status"`
	Output      string `json:"output,omitempty"`
	WaitingFor  string `json:"waiting_for,omitempty"` // 等待人工输入的节点 ID
}

// NodeTypeInfo 节点类型信息（用于前端展示）
type NodeTypeInfo struct {
	Type        NodeType `json:"type"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Color       string   `json:"color"`
	Icon        string   `json:"icon"`
	Category    string   `json:"category"` // basic/control/advanced
}

// GetNodeTypes 返回所有节点类型信息
func GetNodeTypes() []NodeTypeInfo {
	return []NodeTypeInfo{
		// 基础节点
		{NodeTypeStart, "开始", "流程的起点", "#22c55e", "play", "basic"},
		{NodeTypeAgent, "Agent", "调用数字员工执行任务", "#3b82f6", "bot", "basic"},
		{NodeTypeHuman, "人工", "等待人工输入或确认", "#f59e0b", "user", "basic"},
		{NodeTypeEnd, "结束", "流程的终点", "#ef4444", "square", "basic"},
		// 控制节点
		{NodeTypeCondition, "条件", "根据条件分支执行", "#4facfe", "git-branch", "control"},
		{NodeTypeParallel, "并行", "同时执行多个分支", "#8b5cf6", "git-merge", "control"},
		{NodeTypeLoop, "循环", "重复执行直到条件满足", "#ec4899", "repeat", "control"},
		{NodeTypeSwitch, "开关", "多路分支选择", "#f97316", "git-branch", "control"},
		{NodeTypeMerge, "合并", "合并多个分支结果", "#14b8a6", "git-merge", "control"},
		// 数据节点
		{NodeTypeTransform, "转换", "数据格式转换和映射", "#8b5cf6", "shuffle", "data"},
		{NodeTypeHTTP, "HTTP", "发送 HTTP 请求", "#0ea5e9", "globe", "data"},
		{NodeTypeDelay, "延迟", "等待指定时间", "#64748b", "clock", "data"},
		// 高级节点
		{NodeTypeSubFlow, "子流程", "嵌套执行另一个流程", "#06b6d4", "folder", "advanced"},
		{NodeTypeWebhook, "Webhook", "等待外部事件触发", "#64748b", "webhook", "advanced"},
	}
}

// InitSchema 初始化数据库表
func InitSchema(db *sql.DB) error {
	// 流程定义表
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS flows (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			type TEXT DEFAULT 'conversation',
			definition TEXT,
			trigger TEXT,
			status TEXT DEFAULT 'draft',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// 执行记录表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS flow_executions (
			id TEXT PRIMARY KEY,
			flow_id TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			trigger TEXT DEFAULT 'manual',
			input TEXT,
			output TEXT,
			variables TEXT,
			current_node TEXT,
			context TEXT,
			history TEXT,
			started_at DATETIME,
			completed_at DATETIME,
			error TEXT,
			FOREIGN KEY (flow_id) REFERENCES flows(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 流程版本表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS flow_versions (
			id TEXT PRIMARY KEY,
			flow_id TEXT NOT NULL,
			version INTEGER NOT NULL,
			name TEXT,
			description TEXT,
			definition TEXT NOT NULL,
			trigger TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_by TEXT,
			FOREIGN KEY (flow_id) REFERENCES flows(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建索引
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_flows_type ON flows(type)`,
		`CREATE INDEX IF NOT EXISTS idx_flows_status ON flows(status)`,
		`CREATE INDEX IF NOT EXISTS idx_exec_flow ON flow_executions(flow_id)`,
		`CREATE INDEX IF NOT EXISTS idx_exec_status ON flow_executions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_versions_flow ON flow_versions(flow_id)`,
		`CREATE INDEX IF NOT EXISTS idx_versions_version ON flow_versions(flow_id, version)`,
		`CREATE INDEX IF NOT EXISTS idx_templates_category ON flow_templates(category)`,
		`CREATE INDEX IF NOT EXISTS idx_templates_public ON flow_templates(is_public)`,
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return err
		}
	}

	// 初始化模板表
	if err := InitTemplateSchema(db); err != nil {
		return err
	}

	// 迁移：添加 trigger 字段（如果不存在）
	// SQLite 不支持 IF NOT EXISTS，使用忽略错误的方式
	db.Exec(`ALTER TABLE flow_executions ADD COLUMN trigger TEXT DEFAULT 'manual'`)

	// 初始化节点模板表
	if err := InitNodeTemplateSchema(db); err != nil {
		return err
	}

	// 初始化测试用例表
	if err := InitTestCaseSchema(db); err != nil {
		return err
	}

	return nil
}

// NodeTemplate 节点模板
type NodeTemplate struct {
	ID          string                 `json:"id" db:"id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	Category    string                 `json:"category" db:"category"`       // 分类：agent, human, condition, etc.
	NodeType    NodeType               `json:"node_type" db:"node_type"`     // 节点类型
	NodeConfig  map[string]interface{} `json:"node_config" db:"node_config"` // 节点配置模板
	IsPublic    bool                   `json:"is_public" db:"is_public"`     // 是否公开
	UseCount    int                    `json:"use_count" db:"use_count"`     // 使用次数
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// InitNodeTemplateSchema 初始化节点模板表
func InitNodeTemplateSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS node_templates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			category TEXT NOT NULL,
			node_type TEXT NOT NULL,
			node_config TEXT,
			is_public INTEGER DEFAULT 0,
			use_count INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

// FlowTestCase 流程测试用例
type FlowTestCase struct {
	ID          string                 `json:"id" db:"id"`
	FlowID      string                 `json:"flow_id" db:"flow_id"`
	Name        string                 `json:"name" db:"name"`
	Description string                 `json:"description" db:"description"`
	Input       map[string]interface{} `json:"input" db:"input"`             // 测试输入
	Expected    map[string]interface{} `json:"expected" db:"expected"`       // 期望输出
	LastRunAt   *time.Time             `json:"last_run_at" db:"last_run_at"` // 最后执行时间
	LastStatus  string                 `json:"last_status" db:"last_status"` // 最后执行状态
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
}

// FlowTestRun 测试执行记录
type FlowTestRun struct {
	ID         string                 `json:"id" db:"id"`
	TestCaseID string                 `json:"test_case_id" db:"test_case_id"`
	FlowID     string                 `json:"flow_id" db:"flow_id"`
	Status     string                 `json:"status" db:"status"` // passed, failed, error
	Input      map[string]interface{} `json:"input" db:"input"`
	Output     map[string]interface{} `json:"output" db:"output"`
	Expected   map[string]interface{} `json:"expected" db:"expected"`
	Duration   int64                  `json:"duration" db:"duration"` // 毫秒
	Error      string                 `json:"error" db:"error"`
	CreatedAt  time.Time              `json:"created_at" db:"created_at"`
}

// InitTestCaseSchema 初始化测试用例表
func InitTestCaseSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS flow_test_cases (
			id TEXT PRIMARY KEY,
			flow_id TEXT NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			input TEXT,
			expected TEXT,
			last_run_at DATETIME,
			last_status TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE IF NOT EXISTS flow_test_runs (
			id TEXT PRIMARY KEY,
			test_case_id TEXT NOT NULL,
			flow_id TEXT NOT NULL,
			status TEXT NOT NULL,
			input TEXT,
			output TEXT,
			expected TEXT,
			duration INTEGER DEFAULT 0,
			error TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_test_cases_flow ON flow_test_cases(flow_id);
		CREATE INDEX IF NOT EXISTS idx_test_runs_case ON flow_test_runs(test_case_id);
	`)
	return err
}