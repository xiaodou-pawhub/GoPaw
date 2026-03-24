// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package audit

import (
	"encoding/json"
	"time"
)

// Category represents the category of an audit event.
type Category string

const (
	CategoryAuth      Category = "auth"      // 认证相关
	CategoryAgent     Category = "agent"     // Agent 操作
	CategoryWorkflow  Category = "workflow"  // 工作流操作
	CategoryTrigger   Category = "trigger"   // Trigger 操作
	CategoryMCP       Category = "mcp"       // MCP 操作
	CategoryMessage   Category = "message"   // Agent 消息
	CategorySystem    Category = "system"    // 系统事件
	CategoryConfig    Category = "config"    // 配置变更
	CategoryHTTP      Category = "http"      // HTTP 请求
	CategoryPermission Category = "permission" // 权限管理
)

// Action represents the action of an audit event.
type Action string

const (
	// Auth actions
	ActionLogin         Action = "login"
	ActionLogout        Action = "logout"
	ActionTokenRefresh  Action = "token_refresh"
	ActionPasswordChange Action = "password_change"

	// Agent actions
	ActionAgentCreate  Action = "agent_create"
	ActionAgentUpdate  Action = "agent_update"
	ActionAgentDelete  Action = "agent_delete"
	ActionAgentSwitch  Action = "agent_switch"
	ActionAgentExecute Action = "agent_execute"

	// Workflow actions
	ActionWorkflowCreate  Action = "workflow_create"
	ActionWorkflowUpdate  Action = "workflow_update"
	ActionWorkflowDelete  Action = "workflow_delete"
	ActionWorkflowExecute Action = "workflow_execute"
	ActionWorkflowCancel  Action = "workflow_cancel"

	// Trigger actions
	ActionTriggerCreate  Action = "trigger_create"
	ActionTriggerUpdate  Action = "trigger_update"
	ActionTriggerDelete  Action = "trigger_delete"
	ActionTriggerFire    Action = "trigger_fire"

	// MCP actions
	ActionMCPCreate Action = "mcp_create"
	ActionMCPUpdate Action = "mcp_update"
	ActionMCPDelete Action = "mcp_delete"
	ActionMCPConnect Action = "mcp_connect"

	// Message actions
	ActionMessageSend    Action = "message_send"
	ActionMessageReceive Action = "message_receive"

	// System actions
	ActionSystemStart   Action = "system_start"
	ActionSystemStop    Action = "system_stop"
	ActionSystemError   Action = "system_error"
	ActionSystemWarning Action = "system_warning"

	// Config actions
	ActionConfigUpdate Action = "config_update"

	// HTTP actions
	ActionHTTPRequest Action = "http_request"

	// Permission actions
	ActionResourceGrant    Action = "resource_grant"    // 资源授权
	ActionResourceRevoke   Action = "resource_revoke"   // 撤销授权
	ActionResourceCreate   Action = "resource_create"   // 创建资源包
	ActionResourceUpdate   Action = "resource_update"   // 更新资源包
	ActionResourceDelete   Action = "resource_delete"   // 删除资源包
	ActionAgentVisibility  Action = "agent_visibility"  // Agent 可见性变更
)

// Status represents the status of an audit event.
type Status string

const (
	StatusSuccess Status = "success"
	StatusFailed  Status = "failed"
	StatusPending Status = "pending"
)

// Log represents a single audit log entry.
type Log struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	Category     Category               `json:"category"`
	Action       Action                 `json:"action"`
	UserID       string                 `json:"user_id,omitempty"`
	UserIP       string                 `json:"user_ip,omitempty"`
	ResourceType string                 `json:"resource_type,omitempty"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	Status       Status                 `json:"status"`
	Details      map[string]interface{} `json:"details,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Duration     int                    `json:"duration,omitempty"` // milliseconds
	RequestID    string                 `json:"request_id,omitempty"`
}

// QueryOptions represents options for querying audit logs.
type QueryOptions struct {
	Category     Category
	Action       Action
	UserID       string
	ResourceType string
	ResourceID   string
	Status       Status
	StartTime    *time.Time
	EndTime      *time.Time
	Limit        int
	Offset       int
}

// Stats represents audit log statistics.
type Stats struct {
	TotalCount    int64            `json:"total_count"`
	SuccessCount  int64            `json:"success_count"`
	FailedCount   int64            `json:"failed_count"`
	ByCategory    map[string]int64 `json:"by_category"`
	ByAction      map[string]int64 `json:"by_action"`
	ByUser        map[string]int64 `json:"by_user"`
	ByDay         map[string]int64 `json:"by_day"` // YYYY-MM-DD -> count
}

// ExportOptions represents options for exporting audit logs.
type ExportOptions struct {
	Format    string     // csv, json
	StartTime *time.Time
	EndTime   *time.Time
	Category  Category
	UserID    string
}

// MarshalDetails marshals details to JSON.
func MarshalDetails(details map[string]interface{}) ([]byte, error) {
	return json.Marshal(details)
}

// UnmarshalDetails unmarshals JSON to details.
func UnmarshalDetails(data []byte) (map[string]interface{}, error) {
	var details map[string]interface{}
	if err := json.Unmarshal(data, &details); err != nil {
		return nil, err
	}
	return details, nil
}

// DefaultQueryOptions returns default query options.
func DefaultQueryOptions() QueryOptions {
	return QueryOptions{
		Limit:  50,
		Offset: 0,
	}
}
