// Package llm provides a unified interface for communicating with language model providers.
package llm

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// GetProviderFunc is called on every LLM request to obtain the current active provider config.
// Returning an error means no provider is configured.
type GetProviderFunc func() (baseURL, apiKey, model string, maxTokens, timeoutSec int, err error)

// LiveClient is an LLM client that calls GetProviderFunc on each request,
// enabling dynamic provider switching via the Web UI without server restart.
type LiveClient struct {
	getProvider GetProviderFunc
	logger      *zap.Logger
}

// NewLiveClient creates a LiveClient backed by the given provider resolver function.
func NewLiveClient(fn GetProviderFunc, logger *zap.Logger) *LiveClient {
	return &LiveClient{getProvider: fn, logger: logger}
}

// ModelName returns the currently active model name, or "not-configured".
func (c *LiveClient) ModelName() string {
	_, _, model, _, _, err := c.getProvider()
	if err != nil {
		return "not-configured"
	}
	return model
}

// Chat performs a blocking, non-streaming chat completion using the currently active provider.
func (c *LiveClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	inner, err := c.resolve()
	if err != nil {
		return nil, err
	}
	return inner.Chat(ctx, req)
}

// Stream sends a streaming chat request using the currently active provider.
func (c *LiveClient) Stream(ctx context.Context, req ChatRequest) (<-chan StreamDelta, error) {
	inner, err := c.resolve()
	if err != nil {
		ch := make(chan StreamDelta, 1)
		ch <- StreamDelta{Error: err}
		close(ch)
		return ch, nil
	}
	return inner.Stream(ctx, req)
}

// resolve obtains the active provider and returns a concrete OpenAIClient.
func (c *LiveClient) resolve() (*OpenAIClient, error) {
	baseURL, apiKey, model, maxTokens, timeoutSec, err := c.getProvider()
	if err != nil {
		return nil, fmt.Errorf("LLM 未配置：%w", err)
	}
	return NewOpenAIClient(baseURL, apiKey, model, maxTokens, timeoutSec, c.logger), nil
}
