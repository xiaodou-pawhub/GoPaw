// Package llm provides a unified interface for communicating with language model providers.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	maxRetries    = 3
	retryBaseMs   = 500
)

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// openAIChatRequest is the wire format sent to the OpenAI Chat Completions endpoint.
type openAIChatRequest struct {
	Model         string            `json:"model"`
	Messages      []openAIMessage   `json:"messages"`
	Tools         []ToolDefinition  `json:"tools,omitempty"`
	MaxTokens     int               `json:"max_tokens,omitempty"`
	Temperature   float32           `json:"temperature,omitempty"`
	Stream        bool              `json:"stream"`
	StreamOptions *streamOptions    `json:"stream_options,omitempty"`
}

// streamOptions requests additional data in the streaming response (e.g. token usage).
// Supported by Dashscope and OpenAI-compatible APIs.
type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type openAIMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message      openAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

// streamToolCallDelta is the per-delta fragment of a tool call in a streaming response.
// The index field identifies which tool call in the batch this fragment belongs to.
type streamToolCallDelta struct {
	Index    int    `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAIStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string                `json:"content"`
			ToolCalls []streamToolCallDelta `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

// OpenAIClient implements Client against any OpenAI-compatible endpoint.
type OpenAIClient struct {
	providerID string
	baseURL    string
	apiKey     string
	model      string
	maxTokens  int
	timeout    time.Duration
	httpClient *http.Client
	logger     *zap.Logger
}

// NewOpenAIClient creates an OpenAIClient.
func NewOpenAIClient(providerID, baseURL, apiKey, model string, maxTokens, timeoutSec int, logger *zap.Logger) *OpenAIClient {
	return &OpenAIClient{
		providerID: providerID,
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		maxTokens:  maxTokens,
		timeout:    time.Duration(timeoutSec) * time.Second,
		httpClient: &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		logger:     logger,
	}
}

// ModelName returns the configured model identifier.
func (c *OpenAIClient) ModelName() string { return c.model }

// Chat performs a blocking, non-streaming chat completion with automatic retries.
func (c *OpenAIClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	payload := c.buildPayload(req, false)

	var resp *openAIResponse
	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			wait := time.Duration(math.Pow(2, float64(attempt-1))*float64(retryBaseMs)) * time.Millisecond
			c.logger.Warn("llm: retrying after error",
				zap.Int("attempt", attempt), zap.Duration("wait", wait), zap.Error(lastErr))
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(wait):
			}
		}

		resp, lastErr = c.doChat(ctx, payload)
		if lastErr == nil && resp != nil && resp.Error == nil {
			break
		}
		// 如果是业务错误（如 API 密钥错误），不重试
		if resp != nil && resp.Error != nil && (resp.Error.Type == "invalid_request_error" || resp.Error.Type == "authentication_error") {
			lastErr = fmt.Errorf("api error [%s]: %s", resp.Error.Type, resp.Error.Message)
			break
		}
	}

	if lastErr != nil {
		// 记录失败并判断是否为持久错误
		isPersistent := strings.Contains(lastErr.Error(), "authentication") || 
						strings.Contains(lastErr.Error(), "401") ||
						strings.Contains(lastErr.Error(), "quota")
		GlobalHealthTracker.RecordFailure(c.providerID, lastErr, isPersistent)
		return nil, fmt.Errorf("llm: chat: %w", lastErr)
	}

	if resp.Error != nil {
		err := fmt.Errorf("llm: api error [%s]: %s", resp.Error.Type, resp.Error.Message)
		GlobalHealthTracker.RecordFailure(c.providerID, err, false)
		return nil, err
	}

	// 记录成功
	GlobalHealthTracker.RecordSuccess(c.providerID)

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("llm: empty choices in response")
	}

	choice := resp.Choices[0]
	out := &ChatResponse{
		Message: ChatMessage{
			Role:      Role(choice.Message.Role),
			Content:   choice.Message.Content,
			ToolCalls: choice.Message.ToolCalls,
		},
		FinishReason: choice.FinishReason,
		Usage: TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
	return out, nil
}

func (c *OpenAIClient) doChat(ctx context.Context, payload openAIChatRequest) (*openAIResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Log the full request for debugging
	c.logger.Debug("LLM Request Details",
		zap.String("method", http.MethodPost),
		zap.String("url", c.baseURL+"/chat/completions"),
		zap.String("model", c.model),
		zap.String("api_key_prefix", c.apiKey[:min(8, len(c.apiKey))]+"..."),
		zap.Any("payload", payload),
		zap.String("raw_body", string(body)),
	)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	// Log request headers
	c.logger.Debug("HTTP Request Headers",
		zap.String("Content-Type", httpReq.Header.Get("Content-Type")),
		zap.String("Authorization", httpReq.Header.Get("Authorization")),
	)

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer httpResp.Body.Close()

	// Log response status
	c.logger.Debug("HTTP Response",
		zap.Int("status_code", httpResp.StatusCode),
		zap.String("status", httpResp.Status),
	)

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Log raw response body
	c.logger.Debug("Raw Response Body",
		zap.String("body", string(respBody)),
	)

	var result openAIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	return &result, nil
}

// Stream sends a streaming chat request and delivers incremental deltas.
func (c *OpenAIClient) Stream(ctx context.Context, req ChatRequest) (<-chan StreamDelta, error) {
	payload := c.buildPayload(req, true)
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("llm: stream: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("llm: stream: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("llm: stream: http: %w", err)
	}
	if httpResp.StatusCode != http.StatusOK {
		httpResp.Body.Close()
		return nil, fmt.Errorf("llm: stream: http status %d", httpResp.StatusCode)
	}

	ch := make(chan StreamDelta, 32)
	go func() {
		defer close(ch)
		defer httpResp.Body.Close()

		streamStart := time.Now()
		firstDelta := true

		// toolCallAcc accumulates incremental tool_call fragments by index.
		// OpenAI streams tool_calls as: first delta has id+name, subsequent deltas append arguments.
		toolCallAcc := map[int]*ToolCall{}

		scanner := bufio.NewScanner(httpResp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}
			var chunk openAIStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				ch <- StreamDelta{Error: fmt.Errorf("llm: stream: parse chunk: %w", err)}
				return
			}
			if len(chunk.Choices) == 0 {
				continue
			}
			choice := chunk.Choices[0]

			// Content delta → emit immediately for smooth streaming.
			if choice.Delta.Content != "" {
				if firstDelta {
					firstDelta = false
					c.logger.Debug("llm: stream first delta received",
						zap.String("model", c.model),
						zap.Duration("ttft", time.Since(streamStart)),
					)
				}
				ch <- StreamDelta{Content: choice.Delta.Content}
			}

			// Tool call delta → accumulate by index.
			for _, tcd := range choice.Delta.ToolCalls {
				if tc, ok := toolCallAcc[tcd.Index]; ok {
					tc.Function.Arguments += tcd.Function.Arguments
					if tcd.ID != "" {
						tc.ID = tcd.ID
					}
					if tcd.Function.Name != "" {
						tc.Function.Name = tcd.Function.Name
					}
				} else {
					toolCallAcc[tcd.Index] = &ToolCall{
						ID:   tcd.ID,
						Type: tcd.Type,
						Function: ToolCallFunction{
							Name:      tcd.Function.Name,
							Arguments: tcd.Function.Arguments,
						},
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			ch <- StreamDelta{Error: fmt.Errorf("llm: stream: scanner: %w", err)}
			return
		}

		// Emit accumulated tool calls as a single final delta, in index order.
		if len(toolCallAcc) > 0 {
			calls := make([]ToolCall, len(toolCallAcc))
			for idx, tc := range toolCallAcc {
				if idx < len(calls) {
					calls[idx] = *tc
				}
			}
			ch <- StreamDelta{ToolCalls: calls, FinishReason: "tool_calls"}
		}
	}()
	return ch, nil
}

func (c *OpenAIClient) buildPayload(req ChatRequest, stream bool) openAIChatRequest {
	msgs := make([]openAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = openAIMessage{
			Role:       string(m.Role),
			Content:    m.Content,
			ToolCalls:  m.ToolCalls,
			ToolCallID: m.ToolCallID,
			Name:       m.Name,
		}
	}
	maxTok := c.maxTokens
	if req.MaxTokens > 0 {
		maxTok = req.MaxTokens
	}
	r := openAIChatRequest{
		Model:     c.model,
		Messages:  msgs,
		Tools:     req.Tools,
		MaxTokens: maxTok,
		Stream:    stream,
	}
	if stream {
		r.StreamOptions = &streamOptions{IncludeUsage: true}
	}
	return r
}
