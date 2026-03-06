// Package llm provides a unified interface for communicating with language model providers.
package llm

import "context"

// Role represents the author of a chat message.
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

// ChatMessage is one turn in a multi-turn conversation.
type ChatMessage struct {
	Role       Role        `json:"role"`
	Content    string      `json:"content"`
	// ToolCalls is populated when the assistant requests tool invocations.
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	// ToolCallID is set on role=tool messages to match the assistant's request.
	ToolCallID string      `json:"tool_call_id,omitempty"`
	// Name is an optional identifier for the tool that produced this message.
	Name       string      `json:"name,omitempty"`
}

// ToolCall represents a single function-call request issued by the LLM.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"` // always "function"
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction carries the function name and its JSON-encoded arguments.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolDefinition describes a tool the LLM may call, following the OpenAI schema.
type ToolDefinition struct {
	Type     string           `json:"type"` // "function"
	Function FunctionDef      `json:"function"`
}

// FunctionDef is the nested function description inside a ToolDefinition.
type FunctionDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"` // JSON Schema object
}

// ChatRequest bundles the inputs for a single LLM call.
type ChatRequest struct {
	Messages    []ChatMessage    `json:"messages"`
	Tools       []ToolDefinition `json:"tools,omitempty"`
	// MaxTokens overrides the provider default when non-zero.
	MaxTokens   int              `json:"max_tokens,omitempty"`
	// Temperature controls output randomness (0-2).
	Temperature float32          `json:"temperature,omitempty"`
}

// ChatResponse is the normalised output from a non-streaming LLM call.
type ChatResponse struct {
	Message      ChatMessage
	// FinishReason explains why generation stopped ("stop", "tool_calls", "length" …).
	FinishReason string
	// Usage reports token consumption.
	Usage        TokenUsage
}

// TokenUsage reports token counts for billing and compression heuristics.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// StreamDelta is one incremental chunk received during streaming.
type StreamDelta struct {
	// Content is the text fragment (may be empty when tool calls arrive).
	Content      string
	// ToolCalls contains partial or complete tool-call data.
	ToolCalls    []ToolCall
	// FinishReason is non-empty on the last chunk.
	FinishReason string
	// Error is set if the stream encountered a fatal error mid-flight.
	Error        error
}

// Client is the unified interface for all LLM providers.
type Client interface {
	// Chat performs a blocking, non-streaming chat completion.
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// Stream sends a chat request and delivers incremental deltas via the returned channel.
	Stream(ctx context.Context, req ChatRequest) (<-chan StreamDelta, error)

	// ModelName returns the configured model identifier.
	ModelName() string
}

// Embedder describes a provider capable of generating vector embeddings for text.
type Embedder interface {
	Embed(ctx context.Context, input string) ([]float32, error)
}

