// Package convlog writes per-conversation NDJSON event logs.
// Each line is a JSON event, appended to ~/.gopaw/logs/conversations.jsonl.
// Events are also printed to the console logger in a simplified one-line format.
package convlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unicode/utf8"

	"go.uber.org/zap"
)

// Event types for conversation logging.
const (
	EventUserMessage = "user_message"
	EventToolCall    = "tool_call"
	EventToolResult  = "tool_result"
	EventAgentReply  = "agent_reply"
)

// maxConsoleContent is the max rune length of content shown in console output.
const maxConsoleContent = 80

// Event represents a single conversation event.
type Event struct {
	Ts        time.Time       `json:"ts"`
	SessionID string          `json:"session_id"`
	Event     string          `json:"event"`
	Content   string          `json:"content,omitempty"`
	// tool_call / tool_result fields
	Tool   string          `json:"tool,omitempty"`
	Input  json.RawMessage `json:"input,omitempty"`
	Output string          `json:"output,omitempty"`
	Error  *string         `json:"error,omitempty"`
	// agent_reply fields
	Tokens *TokenUsage `json:"tokens,omitempty"`
}

// TokenUsage tracks LLM token consumption.
type TokenUsage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
}

// Logger appends NDJSON events to a file and prints simplified lines to the console.
type Logger struct {
	mu     sync.Mutex
	file   *os.File
	logger *zap.Logger // console logger; may be nil (disables console output)
}

// New opens (or creates) the log file for appending.
// Pass a non-nil zap.Logger to also emit simplified events to the console.
func New(path string, logger *zap.Logger) (*Logger, error) {
	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &Logger{file: f, logger: logger}, nil
}

// Log appends one event to the JSONL file. Thread-safe.
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

// logConsole prints a simplified one-line event summary to the console.
func (l *Logger) logConsole(e Event) {
	if l.logger == nil {
		return
	}
	sid := shortID(e.SessionID)
	switch e.Event {
	case EventUserMessage:
		l.logger.Info("conv: user",
			zap.String("session", sid),
			zap.String("content", clip(e.Content)),
		)
	case EventAgentReply:
		l.logger.Info("conv: agent",
			zap.String("session", sid),
			zap.String("content", clip(e.Content)),
		)
	case EventToolCall:
		l.logger.Info("conv: tool→call",
			zap.String("session", sid),
			zap.String("tool", e.Tool),
			zap.String("input", clip(string(e.Input))),
		)
	case EventToolResult:
		errStr := ""
		if e.Error != nil {
			errStr = *e.Error
		}
		l.logger.Info("conv: tool←result",
			zap.String("session", sid),
			zap.String("tool", e.Tool),
			zap.String("output", clip(e.Output)),
			zap.String("error", errStr),
		)
	}
}

// Close closes the underlying file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

// LogUserMessage logs a user message event.
func (l *Logger) LogUserMessage(sessionID, content string) error {
	e := Event{SessionID: sessionID, Event: EventUserMessage, Content: content}
	l.logConsole(e)
	return l.Log(e)
}

// LogToolCall logs a tool call event.
func (l *Logger) LogToolCall(sessionID, tool string, input json.RawMessage) error {
	e := Event{SessionID: sessionID, Event: EventToolCall, Tool: tool, Input: input}
	l.logConsole(e)
	return l.Log(e)
}

// LogToolResult logs a tool result event.
func (l *Logger) LogToolResult(sessionID, tool, output string, errMsg *string) error {
	e := Event{SessionID: sessionID, Event: EventToolResult, Tool: tool, Output: output, Error: errMsg}
	l.logConsole(e)
	return l.Log(e)
}

// LogAgentReply logs an agent reply event with optional token usage.
func (l *Logger) LogAgentReply(sessionID, content string, tokens *TokenUsage) error {
	e := Event{SessionID: sessionID, Event: EventAgentReply, Content: content, Tokens: tokens}
	l.logConsole(e)
	return l.Log(e)
}

// shortID returns the first 8 characters of a UUID-style session ID.
func shortID(id string) string {
	runes := []rune(id)
	if len(runes) <= 8 {
		return id
	}
	return string(runes[:8])
}

// clip truncates s to maxConsoleContent runes and appends "…" if cut.
func clip(s string) string {
	if utf8.RuneCountInString(s) <= maxConsoleContent {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxConsoleContent]) + "…"
}
