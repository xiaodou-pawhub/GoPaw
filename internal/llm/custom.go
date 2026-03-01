// Package llm provides a unified interface for communicating with language model providers.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

// CustomClient implements Client for arbitrary LLM APIs.
// The caller supplies a request template (JSON with {{messages}} placeholder)
// and a dot-path that locates the text reply inside the response JSON.
type CustomClient struct {
	endpoint        string
	apiKey          string
	model           string
	requestTemplate string  // JSON template; {{messages}} is replaced with the marshalled message array
	responsePath    string  // dot-path into the JSON response, e.g. "data.choices.0.message.content"
	httpClient      *http.Client
	logger          *zap.Logger
}

// NewCustomClient creates a CustomClient.
func NewCustomClient(endpoint, apiKey, model, requestTemplate, responsePath string, timeoutSec int, logger *zap.Logger) *CustomClient {
	return &CustomClient{
		endpoint:        endpoint,
		apiKey:          apiKey,
		model:           model,
		requestTemplate: requestTemplate,
		responsePath:    responsePath,
		httpClient:      &http.Client{Timeout: time.Duration(timeoutSec) * time.Second},
		logger:          logger,
	}
}

// ModelName returns the configured model name.
func (c *CustomClient) ModelName() string { return c.model }

// Chat performs a blocking call to the custom LLM endpoint.
func (c *CustomClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	msgsJSON, err := json.Marshal(req.Messages)
	if err != nil {
		return nil, fmt.Errorf("custom llm: marshal messages: %w", err)
	}

	// Build request body from template.
	body := c.requestTemplate
	if body == "" {
		// Fall back to a simple OpenAI-like payload.
		defaultPayload := map[string]interface{}{
			"model":    c.model,
			"messages": req.Messages,
		}
		b, _ := json.Marshal(defaultPayload)
		body = string(b)
	} else {
		body = strings.ReplaceAll(body, "{{messages}}", string(msgsJSON))
		body = strings.ReplaceAll(body, "{{model}}", c.model)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewBufferString(body))
	if err != nil {
		return nil, fmt.Errorf("custom llm: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("custom llm: http: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("custom llm: read response: %w", err)
	}

	var raw interface{}
	if err := json.Unmarshal(respBody, &raw); err != nil {
		return nil, fmt.Errorf("custom llm: parse response json: %w", err)
	}

	// Extract the text via the configured dot-path.
	text, err := extractPath(raw, c.responsePath)
	if err != nil {
		return nil, fmt.Errorf("custom llm: extract response path %q: %w", c.responsePath, err)
	}

	return &ChatResponse{
		Message:      ChatMessage{Role: RoleAssistant, Content: text},
		FinishReason: "stop",
	}, nil
}

// Stream is not implemented for the custom provider; it falls back to Chat.
func (c *CustomClient) Stream(ctx context.Context, req ChatRequest) (<-chan StreamDelta, error) {
	resp, err := c.Chat(ctx, req)
	if err != nil {
		return nil, err
	}
	ch := make(chan StreamDelta, 1)
	ch <- StreamDelta{Content: resp.Message.Content, FinishReason: "stop"}
	close(ch)
	return ch, nil
}

// extractPath walks a JSON value using a dot-separated path and returns the string at the leaf.
func extractPath(v interface{}, path string) (string, error) {
	if path == "" {
		if s, ok := v.(string); ok {
			return s, nil
		}
		b, _ := json.Marshal(v)
		return string(b), nil
	}
	parts := strings.SplitN(path, ".", 2)
	key := parts[0]
	rest := ""
	if len(parts) == 2 {
		rest = parts[1]
	}
	switch node := v.(type) {
	case map[string]interface{}:
		child, ok := node[key]
		if !ok {
			return "", fmt.Errorf("key %q not found", key)
		}
		return extractPath(child, rest)
	case []interface{}:
		// Allow numeric index in the path.
		var idx int
		if _, err := fmt.Sscanf(key, "%d", &idx); err != nil {
			return "", fmt.Errorf("expected numeric index, got %q", key)
		}
		if idx < 0 || idx >= len(node) {
			return "", fmt.Errorf("index %d out of range (len=%d)", idx, len(node))
		}
		return extractPath(node[idx], rest)
	default:
		if s, ok := v.(string); ok {
			return s, nil
		}
		return "", fmt.Errorf("cannot traverse %T with key %q", v, key)
	}
}
