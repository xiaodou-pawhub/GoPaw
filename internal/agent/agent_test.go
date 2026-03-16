package agent

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/gopaw/gopaw/internal/agent/mocks"
	"github.com/gopaw/gopaw/internal/llm"
	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/internal/skill"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupTestAgent creates a ReActAgent with mock dependencies.
func setupTestAgent(t *testing.T) (*ReActAgent, *mocks.MockLLMClient, *tool.Registry) {
	logger := zap.NewNop()
	mockLLM := &mocks.MockLLMClient{}
	registry := tool.NewRegistry(logger)
	skillMgr := skill.NewManager("", logger)
	memMgr := memory.NewManager(":memory:", logger)
	ltmStore, _ := memory.NewLTMStore(":memory:")

	agent := New(
		mockLLM,
		registry,
		skillMgr,
		memMgr,
		Config{
			DefaultPrompt: "You are a helpful assistant.",
			MaxSteps:      10,
			LTMStore:      ltmStore,
		},
		logger,
	)

	return agent, mockLLM, registry
}

// TestReActLoop_SingleStep tests a simple single-step response.
func TestReActLoop_SingleStep(t *testing.T) {
	agent, mockLLM, _ := setupTestAgent(t)
	ctx := context.Background()

	// Mock LLM to return final answer immediately
	mockLLM.ChatFunc = func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Message: llm.ChatMessage{
				Role:    llm.RoleAssistant,
				Content: "Hello! How can I help you today?",
			},
			FinishReason: "stop",
			Usage: llm.Usage{
				PromptTokens:     100,
				CompletionTokens: 20,
			},
		}, nil
	}

	req := &types.Request{
		SessionID: "test-session",
		UserID:    "test-user",
		Channel:   "test",
		ChatID:    "test-chat",
		Content:   "Hi",
	}

	resp, err := agent.Process(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "Hello! How can I help you today?", resp.Content)
	assert.Equal(t, types.MsgTypeText, resp.MsgType)
}

// TestReActLoop_MultiStep tests multi-step reasoning with tool calls.
func TestReActLoop_MultiStep(t *testing.T) {
	agent, mockLLM, registry := setupTestAgent(t)
	ctx := context.Background()

	// Register a mock tool
	mockTool := &mocks.MockTool{
		NameVal:        "calculator",
		DescriptionVal: "Calculate mathematical expressions",
		ParamsVal: plugin.ToolParameters{
			Type: "object",
			Properties: map[string]plugin.ToolProperty{
				"expression": {Type: "string", Description: "Math expression"},
			},
		},
		ExecuteFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			return plugin.NewToolResult("42")
		},
	}
	registry.Register(mockTool)

	callCount := 0
	mockLLM.ChatFunc = func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
		callCount++

		switch callCount {
		case 1:
			// First call: request tool execution
			return &llm.ChatResponse{
				Message: llm.ChatMessage{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: llm.FunctionCall{
								Name:      "calculator",
								Arguments: `{"expression": "6*7"}`,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			}, nil
		case 2:
			// Second call: final answer
			return &llm.ChatResponse{
				Message: llm.ChatMessage{
					Role:    llm.RoleAssistant,
					Content: "The result is 42.",
				},
				FinishReason: "stop",
			}, nil
		default:
			return nil, fmt.Errorf("unexpected call %d", callCount)
		}
	}

	req := &types.Request{
		SessionID: "test-session",
		UserID:    "test-user",
		Channel:   "test",
		ChatID:    "test-chat",
		Content:   "What is 6 times 7?",
	}

	resp, err := agent.Process(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "The result is 42.", resp.Content)
	assert.Equal(t, 2, callCount, "Expected 2 LLM calls")
}

// TestReActLoop_MaxStepsReached tests that the agent stops at max steps.
func TestReActLoop_MaxStepsReached(t *testing.T) {
	agent, mockLLM, registry := setupTestAgent(t)
	ctx := context.Background()

	// Register a tool that keeps being called
	mockTool := &mocks.MockTool{
		NameVal:        "infinite_tool",
		DescriptionVal: "Never ends",
		ExecuteFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			return plugin.NewToolResult("result")
		},
	}
	registry.Register(mockTool)

	// Always return tool call (never final answer)
	mockLLM.ChatFunc = func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Message: llm.ChatMessage{
				Role: llm.RoleAssistant,
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "infinite_tool",
							Arguments: `{}`,
						},
					},
				},
			},
			FinishReason: "tool_calls",
		}, nil
	}

	req := &types.Request{
		SessionID: "test-session",
		UserID:    "test-user",
		Channel:   "test",
		ChatID:    "test-chat",
		Content:   "Infinite loop test",
	}

	_, err := agent.Process(ctx, req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "max steps reached")
}

// TestReActLoop_ToolCallError tests handling of tool execution errors.
func TestReActLoop_ToolCallError(t *testing.T) {
	agent, mockLLM, registry := setupTestAgent(t)
	ctx := context.Background()

	// Register a tool that returns error
	mockTool := &mocks.MockTool{
		NameVal:        "error_tool",
		DescriptionVal: "Always fails",
		ExecuteFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			return plugin.ErrorResult("tool execution failed")
		},
	}
	registry.Register(mockTool)

	callCount := 0
	mockLLM.ChatFunc = func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
		callCount++

		if callCount == 1 {
			return &llm.ChatResponse{
				Message: llm.ChatMessage{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: llm.FunctionCall{
								Name:      "error_tool",
								Arguments: `{}`,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			}, nil
		}

		// After tool error, agent should handle gracefully
		return &llm.ChatResponse{
			Message: llm.ChatMessage{
				Role:    llm.RoleAssistant,
				Content: "I encountered an error but can still respond.",
			},
			FinishReason: "stop",
		}, nil
	}

	req := &types.Request{
		SessionID: "test-session",
		UserID:    "test-user",
		Channel:   "test",
		ChatID:    "test-chat",
		Content:   "Test error handling",
	}

	resp, err := agent.Process(ctx, req)

	require.NoError(t, err)
	assert.Equal(t, "I encountered an error but can still respond.", resp.Content)
}

// TestReActLoop_LLMError tests handling of LLM call errors.
func TestReActLoop_LLMError(t *testing.T) {
	agent, mockLLM, _ := setupTestAgent(t)
	ctx := context.Background()

	mockLLM.ChatFunc = func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
		return nil, errors.New("LLM service unavailable")
	}

	req := &types.Request{
		SessionID: "test-session",
		UserID:    "test-user",
		Channel:   "test",
		ChatID:    "test-chat",
		Content:   "Test",
	}

	_, err := agent.Process(ctx, req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "llm call")
}

// TestContextWindowGuard_HardLimit tests context window hard limit enforcement.
func TestContextWindowGuard_HardLimit(t *testing.T) {
	agent, mockLLM, _ := setupTestAgent(t)
	ctx := context.Background()

	mockLLM.ChatFunc = func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Message: llm.ChatMessage{
				Role:    llm.RoleAssistant,
				Content: "Response",
			},
			FinishReason: "stop",
			Usage: llm.Usage{
				PromptTokens: contextHardLimitTokens + 1000, // Exceed hard limit
			},
		}, nil
	}

	req := &types.Request{
		SessionID: "test-session",
		UserID:    "test-user",
		Channel:   "test",
		ChatID:    "test-chat",
		Content:   "Test",
	}

	_, err := agent.Process(ctx, req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context window exhausted")
}

// TestLoopDetector_PreventsInfiniteLoop tests loop detection mechanism.
func TestLoopDetector_PreventsInfiniteLoop(t *testing.T) {
	agent, mockLLM, registry := setupTestAgent(t)
	ctx := context.Background()

	// Register a tool
	mockTool := &mocks.MockTool{
		NameVal:        "repeater",
		DescriptionVal: "Repeats call",
		ExecuteFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			return plugin.NewToolResult("repeat")
		},
	}
	registry.Register(mockTool)

	callCount := 0
	mockLLM.ChatFunc = func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
		callCount++

		// Always call the same tool with same arguments (loop pattern)
		return &llm.ChatResponse{
			Message: llm.ChatMessage{
				Role: llm.RoleAssistant,
				ToolCalls: []llm.ToolCall{
					{
						ID:   fmt.Sprintf("call_%d", callCount),
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "repeater",
							Arguments: `{"action": "same"}`,
						},
					},
				},
			},
			FinishReason: "tool_calls",
		}, nil
	}

	req := &types.Request{
		SessionID: "test-session",
		UserID:    "test-user",
		Channel:   "test",
		ChatID:    "test-chat",
		Content:   "Test loop",
	}

	_, err := agent.Process(ctx, req)

	// Should detect loop and return error
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loop")
}

// TestToolExecution_Parallel tests parallel tool execution.
func TestToolExecution_Parallel(t *testing.T) {
	agent, mockLLM, registry := setupTestAgent(t)
	ctx := context.Background()

	// Register multiple tools
	tool1 := &mocks.MockTool{
		NameVal: "tool1",
		ExecuteFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			return plugin.NewToolResult("result1")
		},
	}
	tool2 := &mocks.MockTool{
		NameVal: "tool2",
		ExecuteFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			return plugin.NewToolResult("result2")
		},
	}
	registry.Register(tool1)
	registry.Register(tool2)

	mockLLM.ChatFunc = func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Message: llm.ChatMessage{
				Role: llm.RoleAssistant,
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "tool1",
							Arguments: `{}`,
						},
					},
					{
						ID:   "call_2",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "tool2",
							Arguments: `{}`,
						},
					},
				},
			},
			FinishReason: "tool_calls",
		}, nil
	}

	req := &types.Request{
		SessionID: "test-session",
		UserID:    "test-user",
		Channel:   "test",
		ChatID:    "test-chat",
		Content:   "Test parallel",
	}

	// Should complete without error
	_, err := agent.Process(ctx, req)
	require.NoError(t, err)
}

// TestEmptyContent_Handling tests handling of empty LLM responses.
func TestEmptyContent_Handling(t *testing.T) {
	agent, mockLLM, _ := setupTestAgent(t)
	ctx := context.Background()

	mockLLM.ChatFunc = func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Message: llm.ChatMessage{
				Role:    llm.RoleAssistant,
				Content: "", // Empty content
			},
			FinishReason: "stop",
		}, nil
	}

	req := &types.Request{
		SessionID: "test-session",
		UserID:    "test-user",
		Channel:   "test",
		ChatID:    "test-chat",
		Content:   "Test",
	}

	_, err := agent.Process(ctx, req)

	// Should handle gracefully
	require.Error(t, err)
	assert.Contains(t, err.Error(), "max steps reached without any content")
}

// TestProcessStream_Basic tests the streaming process endpoint.
func TestProcessStream_Basic(t *testing.T) {
	agent, mockLLM, _ := setupTestAgent(t)
	ctx := context.Background()

	mockLLM.ChatFunc = func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Message: llm.ChatMessage{
				Role:    llm.RoleAssistant,
				Content: "Streaming response",
			},
			FinishReason: "stop",
		}, nil
	}

	req := &types.Request{
		SessionID: "test-session",
		UserID:    "test-user",
		Channel:   "test",
		ChatID:    "test-chat",
		Content:   "Test stream",
	}

	var progressEvents []ProgressEvent
	var finalContent string

	result, err := agent.ProcessStream(
		ctx,
		req,
		func(evt ProgressEvent) {
			progressEvents = append(progressEvents, evt)
		},
		func(delta string) {
			finalContent += delta
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "Streaming response", result)
	assert.Equal(t, "Streaming response", finalContent)
}

// BenchmarkProcess_SingleStep benchmarks single-step processing.
func BenchmarkProcess_SingleStep(b *testing.B) {
	logger := zap.NewNop()
	mockLLM := &mocks.MockLLMClient{}
	registry := tool.NewRegistry(logger)
	skillMgr := skill.NewManager("", logger)
	memMgr := memory.NewManager(":memory:", logger)
	ltmStore, _ := memory.NewLTMStore(":memory:")

	agent := New(
		mockLLM,
		registry,
		skillMgr,
		memMgr,
		Config{
			DefaultPrompt: "You are a helpful assistant.",
			MaxSteps:      10,
			LTMStore:      ltmStore,
		},
		logger,
	)

	ctx := context.Background()

	mockLLM.ChatFunc = func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
		return &llm.ChatResponse{
			Message: llm.ChatMessage{
				Role:    llm.RoleAssistant,
				Content: "Response",
			},
			FinishReason: "stop",
		}, nil
	}

	req := &types.Request{
		SessionID: "bench-session",
		UserID:    "bench-user",
		Channel:   "test",
		ChatID:    "test-chat",
		Content:   "Benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = agent.Process(ctx, req)
	}
}

// BenchmarkProcess_MultiStep benchmarks multi-step processing.
func BenchmarkProcess_MultiStep(b *testing.B) {
	logger := zap.NewNop()
	mockLLM := &mocks.MockLLMClient{}
	registry := tool.NewRegistry(logger)
	skillMgr := skill.NewManager("", logger)
	memMgr := memory.NewManager(":memory:", logger)
	ltmStore, _ := memory.NewLTMStore(":memory:")

	agent := New(
		mockLLM,
		registry,
		skillMgr,
		memMgr,
		Config{
			DefaultPrompt: "You are a helpful assistant.",
			MaxSteps:      10,
			LTMStore:      ltmStore,
		},
		logger,
	)

	ctx := context.Background()

	tool := &mocks.MockTool{
		NameVal: "bench_tool",
		ExecuteFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			return plugin.NewToolResult("result")
		},
	}
	registry.Register(tool)

	callCount := 0
	mockLLM.ChatFunc = func(ctx context.Context, req llm.ChatRequest) (*llm.ChatResponse, error) {
		callCount++
		if callCount%2 == 1 {
			return &llm.ChatResponse{
				Message: llm.ChatMessage{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: llm.FunctionCall{
								Name:      "bench_tool",
								Arguments: `{}`,
							},
						},
					},
				},
				FinishReason: "tool_calls",
			}, nil
		}
		return &llm.ChatResponse{
			Message: llm.ChatMessage{
				Role:    llm.RoleAssistant,
				Content: "Final",
			},
			FinishReason: "stop",
		}, nil
	}

	req := &types.Request{
		SessionID: "bench-session",
		UserID:    "bench-user",
		Channel:   "test",
		ChatID:    "test-chat",
		Content:   "Benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = agent.Process(ctx, req)
	}
}
