package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockTool is a mock tool for testing.
type MockTool struct {
	name        string
	description string
	params      plugin.ToolParameters
	executeFunc func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) Parameters() plugin.ToolParameters {
	return m.params
}

func (m *MockTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, args)
	}
	return plugin.ErrorResult("execute not implemented")
}

func setupTestExecutor(t *testing.T) (*Executor, *Registry) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)
	executor := NewExecutor(registry, logger)
	return executor, registry
}

// TestExecute_Success tests successful tool execution.
func TestExecute_Success(t *testing.T) {
	exec, registry := setupTestExecutor(t)
	ctx := context.Background()

	mockTool := &MockTool{
		name:        "test_tool",
		description: "A test tool",
		params: plugin.ToolParameters{
			Type: "object",
			Properties: map[string]plugin.ToolProperty{
				"input": {Type: "string", Description: "Input parameter"},
			},
		},
		executeFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			return plugin.NewToolResult("success result")
		},
	}
	registry.Register(mockTool)

	argsJSON := `{"input": "test"}`
	result, err := exec.Execute(ctx, "test_tool", argsJSON, "test", "chat", "session", "user")

	require.NoError(t, err)
	assert.Equal(t, "success result", result)
}

// TestExecute_ToolNotFound tests execution of non-existent tool.
func TestExecute_ToolNotFound(t *testing.T) {
	exec, _ := setupTestExecutor(t)
	ctx := context.Background()

	result, err := exec.Execute(ctx, "nonexistent", `{}`, "test", "chat", "session", "user")

	require.NoError(t, err) // Returns error message as string, not error
	assert.Contains(t, result, "not found")
}

// TestExecute_InvalidJSON tests execution with invalid JSON arguments.
func TestExecute_InvalidJSON(t *testing.T) {
	exec, registry := setupTestExecutor(t)
	ctx := context.Background()

	mockTool := &MockTool{
		name:        "test_tool",
		executeFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			return plugin.NewToolResult("ok")
		},
	}
	registry.Register(mockTool)

	result, err := exec.Execute(ctx, "test_tool", `invalid json`, "test", "chat", "session", "user")

	require.NoError(t, err)
	assert.Contains(t, result, "failed to parse arguments")
}

// TestExecute_ToolError tests handling of tool execution errors.
func TestExecute_ToolError(t *testing.T) {
	exec, registry := setupTestExecutor(t)
	ctx := context.Background()

	mockTool := &MockTool{
		name: "error_tool",
		executeFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			return plugin.ErrorResult("tool execution failed")
		},
	}
	registry.Register(mockTool)

	result, err := exec.Execute(ctx, "error_tool", `{}`, "test", "chat", "session", "user")

	require.NoError(t, err)
	assert.Contains(t, result, "tool execution failed")
}

// TestExecute_Timeout tests tool execution timeout.
func TestExecute_Timeout(t *testing.T) {
	exec, registry := setupTestExecutor(t)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	slowTool := &MockTool{
		name: "slow_tool",
		executeFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			select {
			case <-time.After(500 * time.Millisecond):
				return plugin.NewToolResult("completed")
			case <-ctx.Done():
				return plugin.ErrorResult("context deadline exceeded")
			}
		},
	}
	registry.Register(slowTool)

	result, err := exec.Execute(ctx, "slow_tool", `{}`, "test", "chat", "session", "user")

	require.NoError(t, err)
	assert.Contains(t, result, "deadline exceeded")
}

// TestExecute_ContextCancellation tests context cancellation.
func TestExecute_ContextCancellation(t *testing.T) {
	exec, registry := setupTestExecutor(t)
	ctx, cancel := context.WithCancel(context.Background())

	cancelTool := &MockTool{
		name: "cancel_tool",
		executeFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			// Cancel context after starting
			go func() {
				time.Sleep(50 * time.Millisecond)
				cancel()
			}()

			select {
			case <-time.After(500 * time.Millisecond):
				return plugin.NewToolResult("completed")
			case <-ctx.Done():
				return plugin.ErrorResult("context canceled")
			}
		},
	}
	registry.Register(cancelTool)

	result, err := exec.Execute(ctx, "cancel_tool", `{}`, "test", "chat", "session", "user")

	require.NoError(t, err)
	assert.Contains(t, result, "canceled")
}

// TestExecute_Concurrent tests concurrent tool execution.
func TestExecute_Concurrent(t *testing.T) {
	exec, registry := setupTestExecutor(t)
	ctx := context.Background()

	// Register multiple tools
	for i := 0; i < 5; i++ {
		tool := &MockTool{
			name: fmt.Sprintf("tool_%d", i),
			executeFunc: func(id int) func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
				return func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
					return plugin.NewToolResult(fmt.Sprintf("result_%d", id))
				}
			}(i),
		}
		registry.Register(tool)
	}

	// Execute concurrently
	results := make(chan string, 5)

	for i := 0; i < 5; i++ {
		go func(id int) {
			result, _ := exec.Execute(ctx, fmt.Sprintf("tool_%d", id), `{}`, "test", "chat", "session", "user")
			results <- result
		}(i)
	}

	// Collect results
	var collected []string
	for i := 0; i < 5; i++ {
		select {
		case result := <-results:
			collected = append(collected, result)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for results")
		}
	}

	assert.Len(t, collected, 5)
	for i := 0; i < 5; i++ {
		assert.Contains(t, collected, fmt.Sprintf("result_%d", i))
	}
}

// TestExecute_WithMetadata tests execution with metadata.
func TestExecute_WithMetadata(t *testing.T) {
	exec, registry := setupTestExecutor(t)
	ctx := context.Background()

	metaTool := &MockTool{
		name: "meta_tool",
		executeFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			return plugin.NewToolResult("ok")
		},
	}
	registry.Register(metaTool)

	result, err := exec.Execute(ctx, "meta_tool", `{}`, "my_channel", "my_chat", "my_session", "my_user")

	require.NoError(t, err)
	assert.Equal(t, "ok", result)
}

// TestExecute_LargeArgs tests execution with large arguments.
func TestExecute_LargeArgs(t *testing.T) {
	exec, registry := setupTestExecutor(t)
	ctx := context.Background()

	// Create large argument
	largeArgs := make(map[string]interface{})
	for i := 0; i < 1000; i++ {
		largeArgs[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d", i)
	}
	argsJSON, _ := json.Marshal(largeArgs)

	largeTool := &MockTool{
		name: "large_tool",
		executeFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			// Verify args are passed correctly
			if len(args) != len(largeArgs) {
				return plugin.ErrorResult(fmt.Sprintf("args count mismatch: got %d, expected %d", len(args), len(largeArgs)))
			}
			return plugin.NewToolResult("processed")
		},
	}
	registry.Register(largeTool)

	result, err := exec.Execute(ctx, "large_tool", string(argsJSON), "test", "chat", "session", "user")

	require.NoError(t, err)
	assert.Equal(t, "processed", result)
}

// TestExecute_EmptyArgs tests execution with empty arguments.
func TestExecute_EmptyArgs(t *testing.T) {
	exec, registry := setupTestExecutor(t)
	ctx := context.Background()

	emptyTool := &MockTool{
		name: "empty_tool",
		executeFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			// Should receive empty map
			if len(args) != 0 {
				return plugin.ErrorResult(fmt.Sprintf("expected empty args, got %d", len(args)))
			}
			return plugin.NewToolResult("success")
		},
	}
	registry.Register(emptyTool)

	result, err := exec.Execute(ctx, "empty_tool", "", "test", "chat", "session", "user")

	require.NoError(t, err)
	assert.Equal(t, "success", result)
}

// TestExecute_ComplexArgs tests execution with complex nested arguments.
func TestExecute_ComplexArgs(t *testing.T) {
	exec, registry := setupTestExecutor(t)
	ctx := context.Background()

	complexArgs := map[string]interface{}{
		"string":  "value",
		"number":  42,
		"boolean": true,
		"array":   []string{"a", "b", "c"},
		"nested": map[string]interface{}{
			"key": "nested_value",
		},
	}
	argsJSON, _ := json.Marshal(complexArgs)

	complexTool := &MockTool{
		name: "complex_tool",
		executeFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			// Verify all types are preserved
			if args["string"] != "value" {
				return plugin.ErrorResult("string mismatch")
			}
			if args["number"] != float64(42) { // JSON numbers are float64
				return plugin.ErrorResult("number mismatch")
			}
			if args["boolean"] != true {
				return plugin.ErrorResult("boolean mismatch")
			}
			return plugin.NewToolResult("success")
		},
	}
	registry.Register(complexTool)

	result, err := exec.Execute(ctx, "complex_tool", string(argsJSON), "test", "chat", "session", "user")

	require.NoError(t, err)
	assert.Equal(t, "success", result)
}

// BenchmarkExecute_Simple benchmarks simple tool execution.
func BenchmarkExecute_Simple(b *testing.B) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)
	executor := NewExecutor(registry, logger)
	ctx := context.Background()

	simpleTool := &MockTool{
		name: "bench_tool",
		executeFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
			return plugin.NewToolResult("result")
		},
	}
	registry.Register(simpleTool)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.Execute(ctx, "bench_tool", `{}`, "test", "chat", "session", "user")
	}
}

// BenchmarkExecute_Concurrent benchmarks concurrent tool execution.
func BenchmarkExecute_Concurrent(b *testing.B) {
	logger := zap.NewNop()
	registry := NewRegistry(logger)
	executor := NewExecutor(registry, logger)
	ctx := context.Background()

	// Register multiple tools
	for i := 0; i < 10; i++ {
		tool := &MockTool{
			name: fmt.Sprintf("tool_%d", i),
			executeFunc: func(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
				return plugin.NewToolResult("result")
			},
		}
		registry.Register(tool)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			toolName := fmt.Sprintf("tool_%d", i%10)
			_, _ = executor.Execute(ctx, toolName, `{}`, "test", "chat", "session", "user")
			i++
		}
	})
}
