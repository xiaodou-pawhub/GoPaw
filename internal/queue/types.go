// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package queue

import (
	"encoding/json"
	"time"
)

// MessageStatus represents the status of a message.
type MessageStatus string

const (
	MessageStatusPending    MessageStatus = "pending"
	MessageStatusProcessing MessageStatus = "processing"
	MessageStatusCompleted  MessageStatus = "completed"
	MessageStatusFailed     MessageStatus = "failed"
	MessageStatusDelayed    MessageStatus = "delayed"
)

// Queue names.
const (
	QueueWorkflow = "workflow"
	QueueAgent    = "agent"
	QueueRetry    = "retry"
	QueueDelay    = "delay"
)

// Priority levels (0 is highest).
const (
	PriorityHigh   = 0
	PriorityNormal = 5
	PriorityLow    = 9
)

// Message represents a queue message.
type Message struct {
	ID          string                 `json:"id"`
	Queue       string                 `json:"queue"`
	Type        string                 `json:"type"`
	Payload     map[string]interface{} `json:"payload"`
	Priority    int                    `json:"priority"`
	Status      MessageStatus          `json:"status"`
	Attempts    int                    `json:"attempts"`
	MaxRetries  int                    `json:"max_retries"`
	DelayUntil  *time.Time             `json:"delay_until,omitempty"`
	ProcessedBy string                 `json:"processed_by,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ProcessedAt *time.Time             `json:"processed_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

// Stats represents queue statistics.
type Stats struct {
	Queue             string    `json:"queue"`
	PendingCount      int64     `json:"pending_count"`
	ProcessingCount   int64     `json:"processing_count"`
	CompletedCount    int64     `json:"completed_count"`
	FailedCount       int64     `json:"failed_count"`
	DelayedCount      int64     `json:"delayed_count"`
	TotalCount        int64     `json:"total_count"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Handler is the function type for processing messages.
type Handler func(msg *Message) error

// PublishOptions represents options for publishing a message.
type PublishOptions struct {
	Priority   int
	MaxRetries int
	Delay      time.Duration
}

// DefaultPublishOptions returns default publish options.
func DefaultPublishOptions() PublishOptions {
	return PublishOptions{
		Priority:   PriorityNormal,
		MaxRetries: 3,
	}
}

// MarshalPayload marshals payload to JSON.
func MarshalPayload(payload map[string]interface{}) ([]byte, error) {
	return json.Marshal(payload)
}

// UnmarshalPayload unmarshals JSON to payload.
func UnmarshalPayload(data []byte) (map[string]interface{}, error) {
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// IsRetryable checks if a message can be retried.
func (m *Message) IsRetryable() bool {
	return m.Attempts < m.MaxRetries && m.Status == MessageStatusFailed
}

// ShouldProcess checks if a message should be processed now.
func (m *Message) ShouldProcess() bool {
	if m.Status != MessageStatusPending && m.Status != MessageStatusDelayed {
		return false
	}
	if m.DelayUntil != nil && time.Now().Before(*m.DelayUntil) {
		return false
	}
	return true
}
