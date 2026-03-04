package llm

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// ProviderEntry describes one LLM provider in the fallback chain.
type ProviderEntry struct {
	ID         string
	Name       string
	BaseURL    string
	APIKey     string
	Model      string
	MaxTokens  int
	TimeoutSec int
}

// GetChainFunc is called on every LLM request to obtain the current ordered
// provider list. The first entry is the primary; subsequent entries are fallbacks.
type GetChainFunc func() ([]ProviderEntry, error)

// FallbackClient is an LLM client that tries providers in priority order.
// If a provider returns an error (network, rate-limit, server error…) the next
// provider in the chain is attempted transparently. Context cancellation
// short-circuits the chain immediately.
type FallbackClient struct {
	getChain GetChainFunc
	logger   *zap.Logger
}

// NewFallbackClient creates a FallbackClient backed by the given chain resolver.
func NewFallbackClient(fn GetChainFunc, logger *zap.Logger) *FallbackClient {
	return &FallbackClient{getChain: fn, logger: logger}
}

// ModelName returns the primary (active) provider's model, or "not-configured".
func (c *FallbackClient) ModelName() string {
	providers, err := c.getChain()
	if err != nil || len(providers) == 0 {
		return "not-configured"
	}
	return providers[0].Model
}

// Chat performs a blocking chat completion, trying each provider in order on failure.
func (c *FallbackClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	providers, err := c.getChain()
	if err != nil {
		return nil, fmt.Errorf("llm: resolve providers: %w", err)
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("llm: no providers configured — visit Web UI → Settings → LLM Providers")
	}

	var lastErr error
	for i, p := range providers {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Health Check: 如果是非主模型且在冷却中，跳过
		if i > 0 && !GlobalHealthTracker.IsAvailable(p.ID) {
			status, _, _ := GlobalHealthTracker.GetStatus(p.ID)
			c.logger.Debug("llm: skipping unavailable fallback provider",
				zap.String("provider", p.Name), zap.String("status", string(status)))
			continue
		}

		client := NewOpenAIClient(p.ID, p.BaseURL, p.APIKey, p.Model, p.MaxTokens, p.TimeoutSec, c.logger)
		resp, err := client.Chat(ctx, req)
		if err == nil {
			if i > 0 {
				c.logger.Info("llm: fallback provider succeeded",
					zap.String("provider", p.Name),
					zap.Int("attempt", i+1),
				)
			}
			return resp, nil
		}
		c.logger.Warn("llm: provider failed, trying next",
			zap.String("provider", p.Name),
			zap.Int("attempt", i+1),
			zap.Int("total", len(providers)),
			zap.Error(err),
		)
		lastErr = err
	}
	return nil, fmt.Errorf("llm: all %d provider(s) failed; last error: %w", len(providers), lastErr)
}

// Stream sends a streaming chat request, trying each provider in order on failure.
func (c *FallbackClient) Stream(ctx context.Context, req ChatRequest) (<-chan StreamDelta, error) {
	providers, err := c.getChain()
	if err != nil {
		return nil, fmt.Errorf("llm: resolve providers: %w", err)
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("llm: no providers configured")
	}

	var lastErr error
	for i, p := range providers {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		// Health Check
		if i > 0 && !GlobalHealthTracker.IsAvailable(p.ID) {
			continue
		}

		client := NewOpenAIClient(p.ID, p.BaseURL, p.APIKey, p.Model, p.MaxTokens, p.TimeoutSec, c.logger)
		ch, err := client.Stream(ctx, req)
		if err == nil {
			if i > 0 {
				c.logger.Info("llm: fallback provider stream succeeded",
					zap.String("provider", p.Name),
					zap.Int("attempt", i+1),
				)
			}
			return ch, nil
		}
		c.logger.Warn("llm: provider stream failed, trying next",
			zap.String("provider", p.Name),
			zap.Int("attempt", i+1),
			zap.Error(err),
		)
		lastErr = err
	}
	return nil, fmt.Errorf("llm: all %d provider(s) failed; last error: %w", len(providers), lastErr)
}
