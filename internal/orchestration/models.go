package orchestration

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Orchestration Agent 编排定义
type Orchestration struct {
	ID          string                  `json:"id" db:"id"`
	Name        string                  `json:"name" db:"name"`
	Description string                  `json:"description" db:"description"`
	Status      string                  `json:"status" db:"status"` // active/draft/disabled
	Definition  OrchestrationDefinition `json:"definition" db:"definition"`
	CreatedAt   time.Time               `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time               `json:"updated_at" db:"updated_at"`
}

// OrchestrationDefinition 编排定义（存储在 JSON 字段）
type OrchestrationDefinition struct {
	Nodes       []OrchestrationNode    `json:"nodes"`
	Edges       []OrchestrationEdge    `json:"edges"`
	Variables   map[string]interface{} `json:"variables"`     // 全局变量
	StartNodeID string                 `json:"start_node_id"` // 起始节点
}

// Value 实现 driver.Valuer 接口
func (d OrchestrationDefinition) Value() (interface{}, error) {
	return json.Marshal(d)
}

// Scan 实现 sql.Scanner 接口
func (d *OrchestrationDefinition) Scan(value interface{}) error {
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

// OrchestrationNode 编排节点（代表一个 Agent 角色）
type OrchestrationNode struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`      // agent/human/condition/workflow/end
	AgentID  string                 `json:"agent_id"`  // 引用的 Agent ID
	Name     string                 `json:"name"`      // 显示名称
	Role     string                 `json:"role"`      // 角色描述
	Prompt   string                 `json:"prompt"`    // 角色 Prompt 前缀
	Config   map[string]interface{} `json:"config"`    // 额外配置
	Position Position               `json:"position"`  // 画布位置
}

// Position 画布位置
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// OrchestrationEdge 编排连线（代表消息传递）
type OrchestrationEdge struct {
	ID          string            `json:"id"`
	Source      string            `json:"source"`       // 源节点 ID
	Target      string            `json:"target"`       // 目标节点 ID
	MessageType string            `json:"message_type"` // task/notify/query/response
	Condition   *EdgeCondition    `json:"condition"`    // 条件（可选）
	Transform   *MessageTransform `json:"transform"`    // 消息转换（可选）
	Label       string            `json:"label"`        // 显示标签
}

// EdgeCondition 连线条件
type EdgeCondition struct {
	Type       string `json:"type"`       // expression/intent/llm
	Expression string `json:"expression"` // 简单表达式
	Intent     string `json:"intent"`     // 意图匹配
	LLMQuery   string `json:"llm_query"`  // LLM 判断
}

// MessageTransform 消息转换
type MessageTransform struct {
	Template string `json:"template"` // 消息模板
}

// OrchestrationExecution 编排执行记录
type OrchestrationExecution struct {
	ID              string                 `json:"id" db:"id"`
	OrchestrationID string                 `json:"orchestration_id" db:"orchestration_id"`
	Status          string                 `json:"status" db:"status"` // running/paused/completed/failed
	Input           string                 `json:"input" db:"input"`
	Output          string                 `json:"output" db:"output"`
	Variables       map[string]interface{} `json:"variables" db:"variables"`
	CurrentNodeID   string                 `json:"current_node_id" db:"current_node_id"`
	StartedAt       time.Time              `json:"started_at" db:"started_at"`
	CompletedAt     *time.Time             `json:"completed_at" db:"completed_at"`
}

// ExecutionMessage 执行消息记录
type ExecutionMessage struct {
	ID          string    `json:"id" db:"id"`
	ExecutionID string    `json:"execution_id" db:"execution_id"`
	FromNodeID  string    `json:"from_node_id" db:"from_node_id"`
	ToNodeID    string    `json:"to_node_id" db:"to_node_id"`
	MessageType string    `json:"message_type" db:"message_type"`
	Content     string    `json:"content" db:"content"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// CreateOrchestrationRequest 创建编排请求
type CreateOrchestrationRequest struct {
	ID          string                  `json:"id" binding:"required"`
	Name        string                  `json:"name" binding:"required"`
	Description string                  `json:"description"`
	Definition  OrchestrationDefinition `json:"definition"`
}

// UpdateOrchestrationRequest 更新编排请求
type UpdateOrchestrationRequest struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Status      string                  `json:"status"`
	Definition  OrchestrationDefinition `json:"definition"`
}

// ExecuteRequest 执行请求
type ExecuteRequest struct {
	Input     string                 `json:"input"`
	Variables map[string]interface{} `json:"variables"`
}

// ExecuteResponse 执行响应
type ExecuteResponse struct {
	ExecutionID string `json:"execution_id"`
	Status      string `json:"status"`
	Output      string `json:"output,omitempty"`
}

// InitSchema 初始化数据库表
func InitSchema(db *sql.DB) error {
	// 编排定义表
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS orchestrations (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'draft',
			definition TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// 编排执行记录表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS orchestration_executions (
			id TEXT PRIMARY KEY,
			orchestration_id TEXT NOT NULL,
			status TEXT,
			input TEXT,
			output TEXT,
			variables TEXT,
			current_node_id TEXT,
			started_at DATETIME,
			completed_at DATETIME,
			FOREIGN KEY (orchestration_id) REFERENCES orchestrations(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 执行消息记录表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS execution_messages (
			id TEXT PRIMARY KEY,
			execution_id TEXT NOT NULL,
			from_node_id TEXT,
			to_node_id TEXT,
			message_type TEXT,
			content TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (execution_id) REFERENCES orchestration_executions(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建索引
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_orch_exec_orch ON orchestration_executions(orchestration_id)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_exec_msg_exec ON execution_messages(execution_id)`)
	if err != nil {
		return err
	}

	return nil
}
