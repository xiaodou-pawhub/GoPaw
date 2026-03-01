// Package tools provides the built-in Tool implementations for GoPaw.
package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&HTTPGetTool{})
}

const maxHTTPResponseSize = 2 << 20 // 2 MB

// HTTPGetTool performs HTTP GET requests and returns the response body.
type HTTPGetTool struct{}

func (t *HTTPGetTool) Name() string { return "http_get" }

func (t *HTTPGetTool) Description() string {
	return "Perform an HTTP GET request to a URL and return the response body as text. Supports custom headers."
}

func (t *HTTPGetTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"url": {
				Type:        "string",
				Description: "The URL to fetch.",
			},
			"headers": {
				Type:        "object",
				Description: "Optional map of HTTP header key-value pairs to include in the request.",
			},
		},
		Required: []string{"url"},
	}
}

func (t *HTTPGetTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return "", fmt.Errorf("http_get: 'url' argument is required")
	}

	httpCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("http_get: build request: %w", err)
	}

	// Apply custom headers if provided.
	if headersRaw, ok := args["headers"]; ok {
		if headersMap, ok := headersRaw.(map[string]interface{}); ok {
			for k, v := range headersMap {
				if vs, ok := v.(string); ok {
					req.Header.Set(k, vs)
				}
			}
		}
	}

	// Set a descriptive User-Agent by default.
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "GoPaw/0.1 (+https://github.com/gopaw/gopaw)")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http_get: request to %q: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxHTTPResponseSize))
	if err != nil {
		return "", fmt.Errorf("http_get: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http_get: server returned status %d for %q", resp.StatusCode, url)
	}

	result := fmt.Sprintf("HTTP %d\nURL: %s\n\n%s", resp.StatusCode, url, string(body))
	return result, nil
}
