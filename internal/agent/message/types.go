// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package message

import (
	"crypto/rand"
	"encoding/json"
	"time"
)

// MessageType represents the type of agent message.
type MessageType string

const (
	// Task - request another agent to perform a task
	MessageTypeTask MessageType = "task"
	// Response - response to a task or query
	MessageTypeResponse MessageType = "response"
	// Notify - notification without expecting response
	MessageTypeNotify MessageType = "notify"
	// Query - query information from another agent
	MessageTypeQuery MessageType = "query"
	// Result - result of processing
	MessageTypeResult MessageType = "result"
)

// MessageStatus represents the status of a message.
type MessageStatus string

const (
	// Pending - message sent but not yet processed
	MessageStatusPending MessageStatus = "pending"
	// Processing - message is being processed
	MessageStatusProcessing MessageStatus = "processing"
	// Completed - message processed successfully
	MessageStatusCompleted MessageStatus = "completed"
	// Failed - message processing failed
	MessageStatusFailed MessageStatus = "failed"
)

// Message represents a message between agents.
type Message struct {
	ID          string          `json:"id"`
	Type        MessageType     `json:"type"`
	FromAgent   string          `json:"from_agent"`
	ToAgent     string          `json:"to_agent"`
	Content     string          `json:"content"`
	Payload     json.RawMessage `json:"payload"` // JSON data
	ParentID    string          `json:"parent_id"` // For reply threading
	Status      MessageStatus   `json:"status"`
	Error       string          `json:"error,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	ProcessedAt *time.Time      `json:"processed_at,omitempty"`
}

// Payload represents the data payload of a message.
type Payload map[string]interface{}

// TaskPayload for task messages.
type TaskPayload struct {
	TaskID      string                 `json:"task_id"`
	Description string                 `json:"description"`
	Priority    string                 `json:"priority"` // low, normal, high, urgent
	Deadline    *time.Time             `json:"deadline,omitempty"`
	Data        map[string]interface{} `json:"data"`
}

// ResponsePayload for response messages.
type ResponsePayload struct {
	InReplyTo string                 `json:"in_reply_to"` // Original message ID
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data"`
}

// NotifyPayload for notification messages.
type NotifyPayload struct {
	Event   string                 `json:"event"`
	Details map[string]interface{} `json:"details"`
}

// QueryPayload for query messages.
type QueryPayload struct {
	QueryID  string                 `json:"query_id"`
	Question string                 `json:"question"`
	Context  map[string]interface{} `json:"context"`
}

// ResultPayload for result messages.
type ResultPayload struct {
	TaskID  string                 `json:"task_id"`
	Output  string                 `json:"output"`
	Metrics map[string]interface{} `json:"metrics"`
}

// Conversation represents a conversation thread between agents.
type Conversation struct {
	ID          string    `json:"id"`
	AgentIDs    []string  `json:"agent_ids"` // Participants
	Title       string    `json:"title"`
	MessageCount int      `json:"message_count"`
	LastMessageAt *time.Time `json:"last_message_at,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MessageStats represents statistics for agent messages.
type MessageStats struct {
	TotalSent     int `json:"total_sent"`
	TotalReceived int `json:"total_received"`
	PendingCount  int `json:"pending_count"`
	FailedCount   int `json:"failed_count"`
}

// MarshalPayload marshals a payload to JSON.
func MarshalPayload(payload interface{}) (json.RawMessage, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(data), nil
}

// UnmarshalPayload unmarshals JSON to a specific payload type.
func UnmarshalPayload(data json.RawMessage, v interface{}) error {
	return json.Unmarshal(data, v)
}

// NewMessage creates a new message.
func NewMessage(msgType MessageType, fromAgent, toAgent, content string, payload interface{}) (*Message, error) {
	id := generateMessageID()

	var payloadData json.RawMessage
	if payload != nil {
		data, err := MarshalPayload(payload)
		if err != nil {
			return nil, err
		}
		payloadData = data
	}

	now := time.Now().UTC()
	return &Message{
		ID:        id,
		Type:      msgType,
		FromAgent: fromAgent,
		ToAgent:   toAgent,
		Content:   content,
		Payload:   payloadData,
		Status:    MessageStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// generateMessageID generates a unique message ID.
func generateMessageID() string {
	return "msg_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString generates a random string of given length.
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	rand.Read(result)
	for i := range result {
		result[i] = charset[result[i]%byte(len(charset))]
	}
	return string(result)
}
