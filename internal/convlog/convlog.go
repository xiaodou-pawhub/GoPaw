// Package convlog writes per-conversation NDJSON event logs.
// Each line is a JSON event, appended to ~/.gopaw/logs/conversations.jsonl.
package convlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Event types for conversation logging.
const (
	EventUserMessage = "user_message"
	EventToolCall    = "tool_call"
	EventToolResult  = "tool_result"
	EventAgentReply  = "agent_reply"
)

// Event represents a single conversation event.
type Event struct {
	Ts        time.Time       `json:"ts"`
	SessionID string          `json:"session_id"`
	Event     string          `json:"event"`
	Content   string          `json:"content,omitempty"`
	Tool      string          `json:"tool,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Output    string          `json:"output,omitempty"`
	Error     *string         `json:"error,omitempty"`
	Tokens    *TokenUsage     `json:"tokens,omitempty"`
}

// TokenUsage tracks LLM token consumption.
type TokenUsage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
}

// Logger appends NDJSON events to a file.
type Logger struct {
	mu   sync.Mutex
	file *os.File
}

// New opens (or creates) the log file for appending.
func New(path string) (*Logger, error) {
	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &Logger{file: f}, nil
}

// Log appends one event. Thread-safe.
func (l *Logger) Log(e Event) error {
	if e.Ts.IsZero() {
		e.Ts = time.Now().UTC()
	}
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err = l.file.Write(data)
	return err
}

// Close closes the underlying file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

// LogUserMessage logs a user message event.
func (l *Logger) LogUserMessage(sessionID, content string) error {
	return l.Log(Event{
		SessionID: sessionID,
		Event:     EventUserMessage,
		Content:   content,
	})
}

// LogToolCall logs a tool call event.
func (l *Logger) LogToolCall(sessionID, tool string, input json.RawMessage) error {
	return l.Log(Event{
		SessionID: sessionID,
		Event:     EventToolCall,
		Tool:      tool,
		Input:     input,
	})
}

// LogToolResult logs a tool result event.
func (l *Logger) LogToolResult(sessionID, tool, output string, errMsg *string) error {
	return l.Log(Event{
		SessionID: sessionID,
		Event:     EventToolResult,
		Tool:      tool,
		Output:    output,
		Error:     errMsg,
	})
}

// LogAgentReply logs an agent reply event with optional token usage.
func (l *Logger) LogAgentReply(sessionID, content string, tokens *TokenUsage) error {
	return l.Log(Event{
		SessionID: sessionID,
		Event:     EventAgentReply,
		Content:   content,
		Tokens:    tokens,
	})
}
