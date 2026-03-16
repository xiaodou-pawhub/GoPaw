// Package mocks provides mock implementations for agent testing.
package mocks

import (
	"context"
	"fmt"

	"github.com/gopaw/gopaw/internal/llm"
	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/internal/skill"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
	"go.uber.org/zap"
)

// MockLLMClient is a mock implementation of llm.Client.
type MockLLMClient struct {
	ChatFunc func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error)
}

func (m *MockLLMClient) Chat(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
	if m.ChatFunc != nil {
		return m.ChatFunc(ctx, req)
	}
	return nil, fmt.Errorf("ChatFunc not implemented")
}

// MockTool is a mock implementation of plugin.Tool.
type MockTool struct {
	NameVal        string
	DescriptionVal string
	ParamsVal      plugin.ToolParameters
	ExecuteFunc    func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult
}

func (m *MockTool) Name() string {
	return m.NameVal
}

func (m *MockTool) Description() string {
	return m.DescriptionVal
}

func (m *MockTool) Parameters() plugin.ToolParameters {
	return m.ParamsVal
}

func (m *MockTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, args)
	}
	return plugin.ErrorResult("ExecuteFunc not implemented")
}

// MockToolRegistry is a mock implementation of tool.Registry.
type MockToolRegistry struct {
	Tools []plugin.Tool
}

func (m *MockToolRegistry) All() []plugin.Tool {
	return m.Tools
}

func (m *MockToolRegistry) Get(name string) (plugin.Tool, bool) {
	for _, t := range m.Tools {
		if t.Name() == name {
			return t, true
		}
	}
	return nil, false
}

// MockSkillManager is a mock implementation of skill.Manager.
type MockSkillManager struct {
	Fragments []string
}

func (m *MockSkillManager) FragmentsForInput(input string) []string {
	return m.Fragments
}

// MockMemoryManager is a mock implementation of memory.Manager.
type MockMemoryManager struct {
	Context []llm.ChatMessage
	AddErr  error
}

func (m *MockMemoryManager) GetContext(sessionID string, limit int) ([]llm.ChatMessage, error) {
	return m.Context, nil
}

func (m *MockMemoryManager) Add(sessionID, userID, channel, userContent, agentContent string) error {
	return m.AddErr
}

func (m *MockMemoryManager) MaybeCompress(sessionID string) {}

// MockMemoryStore is a mock implementation of memory.LTMStore.
type MockMemoryStore struct {
	Memories []memory.Entry
}

func (m *MockMemoryStore) List(category memory.Category, limit int) ([]memory.Entry, error) {
	return m.Memories, nil
}

func (m *MockMemoryStore) Save(entry *memory.Entry) error {
	return nil
}

func (m *MockMemoryStore) Recall(query string, limit int) ([]memory.Entry, error) {
	return m.Memories, nil
}

// MockConvLog is a mock conversation logger.
type MockConvLog struct{}

func (m *MockConvLog) LogUserMessage(sessionID, content string) error    { return nil }
func (m *MockConvLog) LogAgentReply(sessionID string, content string, toolCalls interface{}) error {
	return nil
}
func (m *MockConvLog) LogToolCall(sessionID, toolName string, args interface{}) error { return nil }
func (m *MockConvLog) LogToolResult(sessionID, toolName, result string, err *string) error {
	return nil
}

// NewTestLogger creates a zap logger for testing.
func NewTestLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

// CreateTestAgent creates a ReActAgent with mock dependencies for testing.
func CreateTestAgent(
	llmClient llm.Client,
	toolRegistry *tool.Registry,
	skillManager *skill.Manager,
	memoryManager *memory.Manager,
	ltmStore *memory.LTMStore,
) *ReActAgent {
	// This is a helper that will be implemented in agent_test.go
	// to avoid circular imports
	return nil
}

// MockRequest creates a test request.
func MockRequest() *types.Request {
	return &types.Request{
		SessionID: "test-session",
		UserID:    "test-user",
		Channel:   "test",
		ChatID:    "test-chat",
		Content:   "Hello",
	}
}
