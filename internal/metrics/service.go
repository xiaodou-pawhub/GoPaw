// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package metrics

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"runtime"
	"time"

	"go.uber.org/zap"
)

// Service provides metrics collection and querying.
type Service struct {
	db        *sql.DB
	logger    *zap.Logger
	startTime time.Time
}

// NewService creates a new metrics service.
func NewService(db *sql.DB, logger *zap.Logger) (*Service, error) {
	s := &Service{
		db:        db,
		logger:    logger.Named("metrics"),
		startTime: time.Now(),
	}

	if err := s.initSchema(); err != nil {
		return nil, err
	}

	return s, nil
}

// initSchema creates the metrics tables.
func (s *Service) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS metrics_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metric_type TEXT NOT NULL,
    metric_name TEXT NOT NULL,
    value REAL NOT NULL,
    labels TEXT,
    timestamp TIMESTAMP NOT NULL,
    period TEXT
);

CREATE INDEX IF NOT EXISTS idx_metrics_type_name ON metrics_stats(metric_type, metric_name);
CREATE INDEX IF NOT EXISTS idx_metrics_time ON metrics_stats(timestamp);
`
	_, err := s.db.Exec(schema)
	return err
}

// Collect collects all metrics and stores them.
func (s *Service) Collect() error {
	s.logger.Debug("collecting metrics")

	// Collect agent metrics
	if err := s.collectAgentMetrics(); err != nil {
		s.logger.Warn("failed to collect agent metrics", zap.Error(err))
	}

	// Collect queue metrics
	if err := s.collectQueueMetrics(); err != nil {
		s.logger.Warn("failed to collect queue metrics", zap.Error(err))
	}

	// Collect workflow metrics
	if err := s.collectWorkflowMetrics(); err != nil {
		s.logger.Warn("failed to collect workflow metrics", zap.Error(err))
	}

	// Collect system metrics
	if err := s.collectSystemMetrics(); err != nil {
		s.logger.Warn("failed to collect system metrics", zap.Error(err))
	}

	return nil
}

// collectAgentMetrics collects agent metrics from audit logs.
func (s *Service) collectAgentMetrics() error {
	// Get total agent calls in last hour
	var totalCalls int64
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM audit_logs 
		WHERE category = 'agent' 
		AND timestamp > datetime('now', '-1 hour')
	`).Scan(&totalCalls)
	if err != nil {
		return err
	}

	// Get successful calls
	var successCalls int64
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM audit_logs 
		WHERE category = 'agent' 
		AND status = 'success'
		AND timestamp > datetime('now', '-1 hour')
	`).Scan(&successCalls)
	if err != nil {
		return err
	}

	// Get failed calls
	var failedCalls int64
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM audit_logs 
		WHERE category = 'agent' 
		AND status = 'failed'
		AND timestamp > datetime('now', '-1 hour')
	`).Scan(&failedCalls)
	if err != nil {
		return err
	}

	// Calculate success rate
	successRate := float64(0)
	if totalCalls > 0 {
		successRate = float64(successCalls) / float64(totalCalls)
	}

	// Calculate error rate
	errorRate := float64(0)
	if totalCalls > 0 {
		errorRate = float64(failedCalls) / float64(totalCalls)
	}

	// Store metrics
	now := time.Now().UTC()
	s.storeMetric(MetricTypeAgent, MetricAgentCallCount, float64(totalCalls), nil, now, "1h")
	s.storeMetric(MetricTypeAgent, MetricAgentSuccessRate, successRate, nil, now, "1h")
	s.storeMetric(MetricTypeAgent, MetricAgentErrorRate, errorRate, nil, now, "1h")

	// Collect per-agent metrics
	rows, err := s.db.Query(`
		SELECT resource_id, COUNT(*) as count,
		SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success
		FROM audit_logs 
		WHERE category = 'agent' 
		AND timestamp > datetime('now', '-1 hour')
		GROUP BY resource_id
		ORDER BY count DESC
		LIMIT 10
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var agentID string
		var count, success int64
		if err := rows.Scan(&agentID, &count, &success); err != nil {
			continue
		}

		agentSuccessRate := float64(0)
		if count > 0 {
			agentSuccessRate = float64(success) / float64(count)
		}

		labels := map[string]string{"agent_id": agentID}
		s.storeMetric(MetricTypeAgent, MetricAgentCallCount, float64(count), labels, now, "1h")
		s.storeMetric(MetricTypeAgent, MetricAgentSuccessRate, agentSuccessRate, labels, now, "1h")
	}

	return rows.Err()
}

// collectQueueMetrics collects queue metrics.
func (s *Service) collectQueueMetrics() error {
	// Get counts by status
	rows, err := s.db.Query(`
		SELECT queue, status, COUNT(*) 
		FROM queue_messages 
		GROUP BY queue, status
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	now := time.Now().UTC()
	for rows.Next() {
		var queueName, status string
		var count int64
		if err := rows.Scan(&queueName, &status, &count); err != nil {
			continue
		}

		labels := map[string]string{"queue": queueName}
		switch status {
		case "pending":
			s.storeMetric(MetricTypeQueue, MetricQueuePendingCount, float64(count), labels, now, "current")
		case "processing":
			s.storeMetric(MetricTypeQueue, MetricQueueProcessingCount, float64(count), labels, now, "current")
		case "completed":
			s.storeMetric(MetricTypeQueue, MetricQueueCompletedCount, float64(count), labels, now, "1h")
		case "failed":
			s.storeMetric(MetricTypeQueue, MetricQueueFailedCount, float64(count), labels, now, "1h")
		}
	}

	return rows.Err()
}

// collectWorkflowMetrics collects workflow metrics.
func (s *Service) collectWorkflowMetrics() error {
	// Get total executions in last hour
	var totalExecutions int64
	err := s.db.QueryRow(`
		SELECT COUNT(*) FROM workflow_executions 
		WHERE started_at > datetime('now', '-1 hour')
	`).Scan(&totalExecutions)
	if err != nil {
		return err
	}

	// Get successful executions
	var successExecutions int64
	err = s.db.QueryRow(`
		SELECT COUNT(*) FROM workflow_executions 
		WHERE status = 'completed'
		AND started_at > datetime('now', '-1 hour')
	`).Scan(&successExecutions)
	if err != nil {
		return err
	}

	// Calculate success rate
	successRate := float64(0)
	if totalExecutions > 0 {
		successRate = float64(successExecutions) / float64(totalExecutions)
	}

	// Get average duration
	var avgDuration float64
	err = s.db.QueryRow(`
		SELECT IFNULL(AVG(
			CASE
				WHEN completed_at IS NOT NULL AND started_at IS NOT NULL
				THEN (julianday(completed_at) - julianday(started_at)) * 86400
				ELSE 0
			END
		), 0) FROM workflow_executions
		WHERE started_at > datetime('now', '-1 hour')
	`).Scan(&avgDuration)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	s.storeMetric(MetricTypeWorkflow, MetricWorkflowExecutionCount, float64(totalExecutions), nil, now, "1h")
	s.storeMetric(MetricTypeWorkflow, MetricWorkflowSuccessRate, successRate, nil, now, "1h")
	s.storeMetric(MetricTypeWorkflow, MetricWorkflowAvgDuration, avgDuration, nil, now, "1h")

	// Collect per-workflow metrics
	rows, err := s.db.Query(`
		SELECT w.id, w.name, COUNT(e.id) as count,
		SUM(CASE WHEN e.status = 'completed' THEN 1 ELSE 0 END) as success
		FROM workflows w
		LEFT JOIN workflow_executions e ON w.id = e.workflow_id
		AND e.started_at > datetime('now', '-1 hour')
		GROUP BY w.id
		ORDER BY count DESC
		LIMIT 10
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var workflowID, workflowName string
		var count, success int64
		if err := rows.Scan(&workflowID, &workflowName, &count, &success); err != nil {
			continue
		}

		wfSuccessRate := float64(0)
		if count > 0 {
			wfSuccessRate = float64(success) / float64(count)
		}

		labels := map[string]string{"workflow_id": workflowID}
		s.storeMetric(MetricTypeWorkflow, MetricWorkflowExecutionCount, float64(count), labels, now, "1h")
		s.storeMetric(MetricTypeWorkflow, MetricWorkflowSuccessRate, wfSuccessRate, labels, now, "1h")
	}

	return rows.Err()
}

// collectSystemMetrics collects system metrics.
func (s *Service) collectSystemMetrics() error {
	now := time.Now().UTC()

	// Memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryMB := int64(m.Alloc / 1024 / 1024)
	s.storeMetric(MetricTypeSystem, MetricSystemMemoryMB, float64(memoryMB), nil, now, "current")

	// Goroutine count
	goroutines := runtime.NumGoroutine()
	s.storeMetric(MetricTypeSystem, MetricSystemGoroutines, float64(goroutines), nil, now, "current")

	// Uptime
	uptimeSec := int64(time.Since(s.startTime).Seconds())
	s.storeMetric(MetricTypeSystem, MetricSystemUptimeSec, float64(uptimeSec), nil, now, "current")

	// Database size (if file exists)
	// This is a simplified version - in production, get the actual DB file path
	dbSizeMB := int64(0)
	s.storeMetric(MetricTypeSystem, MetricSystemDBSizeMB, float64(dbSizeMB), nil, now, "current")

	return nil
}

// storeMetric stores a metric to the database.
func (s *Service) storeMetric(metricType MetricType, metricName MetricName, value float64, labels map[string]string, timestamp time.Time, period string) error {
	labelsJSON, _ := json.Marshal(labels)
	
	_, err := s.db.Exec(`
		INSERT INTO metrics_stats (metric_type, metric_name, value, labels, timestamp, period)
		VALUES (?, ?, ?, ?, ?, ?)
	`, string(metricType), string(metricName), value, string(labelsJSON), timestamp, period)
	
	return err
}

// GetDashboardData returns dashboard data.
func (s *Service) GetDashboardData() (*DashboardData, error) {
	data := &DashboardData{
		UpdatedAt: time.Now().UTC(),
	}

	// Get agent metrics
	if err := s.getAgentMetrics(&data.Agent); err != nil {
		s.logger.Warn("failed to get agent metrics", zap.Error(err))
	}

	// Get queue metrics
	if err := s.getQueueMetrics(&data.Queue); err != nil {
		s.logger.Warn("failed to get queue metrics", zap.Error(err))
	}

	// Get workflow metrics
	if err := s.getWorkflowMetrics(&data.Workflow); err != nil {
		s.logger.Warn("failed to get workflow metrics", zap.Error(err))
	}

	// Get system metrics
	if err := s.getSystemMetrics(&data.System); err != nil {
		s.logger.Warn("failed to get system metrics", zap.Error(err))
	}

	return data, nil
}

// getAgentMetrics gets agent metrics for dashboard.
func (s *Service) getAgentMetrics(metrics *AgentMetrics) error {
	// Get latest metrics from database
	row := s.db.QueryRow(`
		SELECT value FROM metrics_stats 
		WHERE metric_type = 'agent' AND metric_name = 'call_count' AND labels IS NULL
		ORDER BY timestamp DESC LIMIT 1
	`)
	row.Scan(&metrics.CallCount)

	row = s.db.QueryRow(`
		SELECT value FROM metrics_stats 
		WHERE metric_type = 'agent' AND metric_name = 'success_rate' AND labels IS NULL
		ORDER BY timestamp DESC LIMIT 1
	`)
	row.Scan(&metrics.SuccessRate)

	row = s.db.QueryRow(`
		SELECT value FROM metrics_stats 
		WHERE metric_type = 'agent' AND metric_name = 'error_rate' AND labels IS NULL
		ORDER BY timestamp DESC LIMIT 1
	`)
	row.Scan(&metrics.ErrorRate)

	// Get top agents
	rows, err := s.db.Query(`
		SELECT labels, value FROM metrics_stats 
		WHERE metric_type = 'agent' AND metric_name = 'call_count' AND labels IS NOT NULL
		ORDER BY timestamp DESC, value DESC
		LIMIT 5
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	seen := make(map[string]bool)
	for rows.Next() {
		var labelsJSON string
		var value float64
		if err := rows.Scan(&labelsJSON, &value); err != nil {
			continue
		}

		var labels map[string]string
		if err := json.Unmarshal([]byte(labelsJSON), &labels); err != nil {
			continue
		}

		agentID := labels["agent_id"]
		if seen[agentID] {
			continue
		}
		seen[agentID] = true

		metrics.TopAgents = append(metrics.TopAgents, AgentStat{
			AgentID:   agentID,
			CallCount: int64(value),
		})
	}

	return rows.Err()
}

// getQueueMetrics gets queue metrics for dashboard.
func (s *Service) getQueueMetrics(metrics *QueueMetrics) error {
	// Get current queue stats
	rows, err := s.db.Query(`
		SELECT queue, status, COUNT(*) as count
		FROM queue_messages
		GROUP BY queue, status
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	queueMap := make(map[string]*QueueStat)
	for rows.Next() {
		var queueName, status string
		var count int64
		if err := rows.Scan(&queueName, &status, &count); err != nil {
			continue
		}

		if _, ok := queueMap[queueName]; !ok {
			queueMap[queueName] = &QueueStat{QueueName: queueName}
		}

		switch status {
		case "pending":
			queueMap[queueName].PendingCount = count
			metrics.PendingCount += count
		case "processing":
			queueMap[queueName].ProcessingCount = count
			metrics.ProcessingCount += count
		case "completed":
			metrics.CompletedCount += count
		case "failed":
			queueMap[queueName].FailedCount = count
			metrics.FailedCount += count
		}
	}

	for _, stat := range queueMap {
		metrics.QueueStats = append(metrics.QueueStats, *stat)
	}

	return rows.Err()
}

// getWorkflowMetrics gets workflow metrics for dashboard.
func (s *Service) getWorkflowMetrics(metrics *WorkflowMetrics) error {
	// Get latest metrics from database
	row := s.db.QueryRow(`
		SELECT value FROM metrics_stats 
		WHERE metric_type = 'workflow' AND metric_name = 'execution_count' AND labels IS NULL
		ORDER BY timestamp DESC LIMIT 1
	`)
	row.Scan(&metrics.ExecutionCount)

	row = s.db.QueryRow(`
		SELECT value FROM metrics_stats 
		WHERE metric_type = 'workflow' AND metric_name = 'success_rate' AND labels IS NULL
		ORDER BY timestamp DESC LIMIT 1
	`)
	row.Scan(&metrics.SuccessRate)

	row = s.db.QueryRow(`
		SELECT value FROM metrics_stats 
		WHERE metric_type = 'workflow' AND metric_name = 'avg_duration_sec' AND labels IS NULL
		ORDER BY timestamp DESC LIMIT 1
	`)
	var avgDuration float64
	row.Scan(&avgDuration)
	metrics.AvgDurationSec = int64(avgDuration)

	// Get top workflows
	rows, err := s.db.Query(`
		SELECT w.id, w.name, COUNT(e.id) as count
		FROM workflows w
		LEFT JOIN workflow_executions e ON w.id = e.workflow_id
		AND e.started_at > datetime('now', '-24 hours')
		GROUP BY w.id
		ORDER BY count DESC
		LIMIT 5
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var workflowID, workflowName string
		var count int64
		if err := rows.Scan(&workflowID, &workflowName, &count); err != nil {
			continue
		}

		// Get success rate for this workflow
		var successCount int64
		s.db.QueryRow(`
			SELECT COUNT(*) FROM workflow_executions 
			WHERE workflow_id = ? AND status = 'completed'
			AND started_at > datetime('now', '-24 hours')
		`, workflowID).Scan(&successCount)

		successRate := float64(0)
		if count > 0 {
			successRate = float64(successCount) / float64(count)
		}

		metrics.TopWorkflows = append(metrics.TopWorkflows, WorkflowStat{
			WorkflowID:     workflowID,
			WorkflowName:   workflowName,
			ExecutionCount: count,
			SuccessRate:    successRate,
		})
	}

	return rows.Err()
}

// getSystemMetrics gets system metrics for dashboard.
func (s *Service) getSystemMetrics(metrics *SystemMetrics) error {
	// Get latest metrics from database
	row := s.db.QueryRow(`
		SELECT value FROM metrics_stats 
		WHERE metric_type = 'system' AND metric_name = 'memory_mb'
		ORDER BY timestamp DESC LIMIT 1
	`)
	var memoryMB float64
	row.Scan(&memoryMB)
	metrics.MemoryMB = int64(memoryMB)

	row = s.db.QueryRow(`
		SELECT value FROM metrics_stats 
		WHERE metric_type = 'system' AND metric_name = 'goroutines'
		ORDER BY timestamp DESC LIMIT 1
	`)
	var goroutines float64
	row.Scan(&goroutines)
	metrics.Goroutines = int(goroutines)

	row = s.db.QueryRow(`
		SELECT value FROM metrics_stats 
		WHERE metric_type = 'system' AND metric_name = 'uptime_sec'
		ORDER BY timestamp DESC LIMIT 1
	`)
	var uptime float64
	row.Scan(&uptime)
	metrics.UptimeSec = int64(uptime)

	return nil
}

// GetRecentActivity returns recent activity items.
func (s *Service) GetRecentActivity(limit int) ([]RecentActivity, error) {
	var activities []RecentActivity

	// Get recent workflow executions
	rows, err := s.db.Query(`
		SELECT id, workflow_id, status, started_at
		FROM workflow_executions
		ORDER BY started_at DESC
		LIMIT ?
	`, limit/3)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var activity RecentActivity
		var workflowID string
		if err := rows.Scan(&activity.ID, &workflowID, &activity.Status, &activity.Timestamp); err != nil {
			continue
		}
		activity.Type = "workflow"
		activity.Action = "execute"
		activity.Description = fmt.Sprintf("Workflow %s executed", workflowID)
		activities = append(activities, activity)
	}

	if err := rows.Err(); err != nil {
		s.logger.Warn("error iterating workflow rows", zap.Error(err))
	}

	// Get recent audit logs
	rows2, err := s.db.Query(`
		SELECT id, category, action, resource_id, status, timestamp
		FROM audit_logs
		ORDER BY timestamp DESC
		LIMIT ?
	`, limit/3)
	if err != nil {
		return activities, nil
	}
	defer rows2.Close()

	for rows2.Next() {
		var activity RecentActivity
		var category, resourceID string
		if err := rows2.Scan(&activity.ID, &category, &activity.Action, &resourceID, &activity.Status, &activity.Timestamp); err != nil {
			continue
		}
		activity.Type = category
		activity.Description = fmt.Sprintf("%s %s on %s", category, activity.Action, resourceID)
		activities = append(activities, activity)
	}

	if err := rows2.Err(); err != nil {
		s.logger.Warn("error iterating audit rows", zap.Error(err))
	}

	return activities, nil
}

// CleanupOldMetrics removes old metrics data.
func (s *Service) CleanupOldMetrics(retentionDays int) error {
	_, err := s.db.Exec(`
		DELETE FROM metrics_stats 
		WHERE timestamp < datetime('now', '-' || ? || ' days')
	`, retentionDays)
	return err
}
