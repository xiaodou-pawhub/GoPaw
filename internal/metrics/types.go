// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package metrics

import (
	"time"
)

// MetricType represents the type of metric.
type MetricType string

const (
	MetricTypeAgent    MetricType = "agent"
	MetricTypeQueue    MetricType = "queue"
	MetricTypeWorkflow MetricType = "workflow"
	MetricTypeSystem   MetricType = "system"
)

// MetricName represents the name of a metric.
type MetricName string

const (
	// Agent metrics
	MetricAgentCallCount   MetricName = "call_count"
	MetricAgentSuccessRate MetricName = "success_rate"
	MetricAgentAvgDuration MetricName = "avg_duration_ms"
	MetricAgentErrorRate   MetricName = "error_rate"

	// Queue metrics
	MetricQueuePendingCount    MetricName = "pending_count"
	MetricQueueProcessingCount MetricName = "processing_count"
	MetricQueueCompletedCount  MetricName = "completed_count"
	MetricQueueFailedCount     MetricName = "failed_count"
	MetricQueueProcessRate     MetricName = "process_rate"

	// Workflow metrics
	MetricWorkflowExecutionCount MetricName = "execution_count"
	MetricWorkflowSuccessRate    MetricName = "success_rate"
	MetricWorkflowAvgDuration    MetricName = "avg_duration_sec"
	MetricWorkflowStepSuccess    MetricName = "step_success_rate"

	// System metrics
	MetricSystemMemoryMB     MetricName = "memory_mb"
	MetricSystemGoroutines   MetricName = "goroutines"
	MetricSystemDBSizeMB     MetricName = "db_size_mb"
	MetricSystemUptimeSec    MetricName = "uptime_sec"
)

// Metric represents a single metric data point.
type Metric struct {
	ID        int64       `json:"id"`
	Type      MetricType  `json:"type"`
	Name      MetricName  `json:"name"`
	Value     float64     `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Period    string      `json:"period"` // 1m, 5m, 1h, 1d
}

// DashboardData represents the dashboard data.
type DashboardData struct {
	Agent    AgentMetrics    `json:"agent"`
	Queue    QueueMetrics    `json:"queue"`
	Workflow WorkflowMetrics `json:"workflow"`
	System   SystemMetrics   `json:"system"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// AgentMetrics represents agent-related metrics.
type AgentMetrics struct {
	CallCount      int64   `json:"call_count"`
	SuccessRate    float64 `json:"success_rate"`
	AvgDurationMs  int64   `json:"avg_duration_ms"`
	ErrorRate      float64 `json:"error_rate"`
	TopAgents      []AgentStat `json:"top_agents,omitempty"`
}

// AgentStat represents statistics for a single agent.
type AgentStat struct {
	AgentID       string  `json:"agent_id"`
	AgentName     string  `json:"agent_name"`
	CallCount     int64   `json:"call_count"`
	SuccessRate   float64 `json:"success_rate"`
	AvgDurationMs int64   `json:"avg_duration_ms"`
}

// QueueMetrics represents queue-related metrics.
type QueueMetrics struct {
	PendingCount    int64       `json:"pending_count"`
	ProcessingCount int64       `json:"processing_count"`
	CompletedCount  int64       `json:"completed_count"`
	FailedCount     int64       `json:"failed_count"`
	QueueStats      []QueueStat `json:"queue_stats,omitempty"`
}

// QueueStat represents statistics for a single queue.
type QueueStat struct {
	QueueName       string `json:"queue_name"`
	PendingCount    int64  `json:"pending_count"`
	ProcessingCount int64  `json:"processing_count"`
	FailedCount     int64  `json:"failed_count"`
}

// WorkflowMetrics represents workflow-related metrics.
type WorkflowMetrics struct {
	ExecutionCount int64          `json:"execution_count"`
	SuccessRate    float64        `json:"success_rate"`
	AvgDurationSec int64          `json:"avg_duration_sec"`
	TopWorkflows   []WorkflowStat `json:"top_workflows,omitempty"`
}

// WorkflowStat represents statistics for a single workflow.
type WorkflowStat struct {
	WorkflowID    string  `json:"workflow_id"`
	WorkflowName  string  `json:"workflow_name"`
	ExecutionCount int64  `json:"execution_count"`
	SuccessRate   float64 `json:"success_rate"`
}

// SystemMetrics represents system-related metrics.
type SystemMetrics struct {
	MemoryMB    int64 `json:"memory_mb"`
	Goroutines  int   `json:"goroutines"`
	DBSizeMB    int64 `json:"db_size_mb"`
	UptimeSec   int64 `json:"uptime_sec"`
}

// HistoryQuery represents a query for historical metrics.
type HistoryQuery struct {
	Type      MetricType `json:"type"`
	Name      MetricName `json:"name"`
	StartTime time.Time  `json:"start_time"`
	EndTime   time.Time  `json:"end_time"`
	Period    string     `json:"period"` // 1m, 5m, 1h, 1d
}

// HistoryData represents historical metric data.
type HistoryData struct {
	Type   MetricType `json:"type"`
	Name   MetricName `json:"name"`
	Labels map[string]string `json:"labels,omitempty"`
	Points []DataPoint `json:"points"`
}

// DataPoint represents a single data point in time series.
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// RecentActivity represents a recent activity item.
type RecentActivity struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // agent, workflow, queue
	Action      string    `json:"action"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // success, failed, pending
	Timestamp   time.Time `json:"timestamp"`
}
