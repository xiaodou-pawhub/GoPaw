// Package tools provides the built-in Tool implementations for GoPaw.
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&HTTPPostTool{})
}

// httpClient is the shared HTTP client for POST requests.
// 中文：用于 POST 请求的共享 HTTP 客户端
// English: Shared HTTP client for POST requests
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

// validateURL checks if the URL is safe to request.
// 中文：校验 URL 是否安全可访问
// English: Validate if URL is safe to access
func validateURL(urlStr string) error {
	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// 中文：仅允许 http/https scheme
	// English: Only allow http/https schemes
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("only http/https schemes are allowed")
	}

	// 中文：检查 host 是否为空
	// English: Check if host is empty
	if u.Host == "" {
		return fmt.Errorf("host is required")
	}

	// TODO: 后续可增加私网地址拦截（按项目安全策略）
	// 当前版本先做基础校验

	return nil
}

// HTTPPostTool performs HTTP POST requests with JSON body and returns the response body as text.
type HTTPPostTool struct{}

func (t *HTTPPostTool) Name() string { return "http_post" }

func (t *HTTPPostTool) Description() string {
	return "Perform an HTTP POST request to a URL with optional JSON body and custom headers. " +
		"Useful for triggering webhooks, calling external APIs, or submitting data."
}

func (t *HTTPPostTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"url": {
				Type:        "string",
				Description: "The URL to send the POST request to.",
			},
			"body": {
				Type:        "object",
				Description: "Optional JSON object to send as the request body.",
			},
			"headers": {
				Type:        "object",
				Description: "Optional map of HTTP header key-value pairs to include in the request.",
			},
		},
		Required: []string{"url"},
	}
}

func (t *HTTPPostTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	// 中文：验证 URL 参数
	// English: Validate URL parameter
	url, ok := args["url"].(string)
	if !ok || url == "" {
		return "", fmt.Errorf("http_post: 'url' argument is required")
	}

	// 中文：URL 安全校验
	// English: URL safety validation
	if err := validateURL(url); err != nil {
		return "", fmt.Errorf("http_post: %w", err)
	}

	// 中文：序列化 body 为 JSON
	// English: Serialize body to JSON
	var bodyReader io.Reader
	if body, exists := args["body"]; exists && body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return "", fmt.Errorf("http_post: marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonData)
	}

	// 中文：创建请求
	// English: Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bodyReader)
	if err != nil {
		return "", fmt.Errorf("http_post: build request: %w", err)
	}

	// 中文：默认 Content-Type 为 application/json
	// English: Set default Content-Type
	req.Header.Set("Content-Type", "application/json")

	// 中文：应用自定义 headers
	// English: Apply custom headers
	if headersRaw, ok := args["headers"]; ok {
		if headersMap, ok := headersRaw.(map[string]interface{}); ok {
			for k, v := range headersMap {
				if vs, ok := v.(string); ok {
					req.Header.Set(k, vs)
				}
			}
		}
	}

	// 中文：设置默认 User-Agent
	// English: Set default User-Agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "GoPaw/0.1 (+https://github.com/gopaw/gopaw)")
	}

	// 中文：发送请求
	// English: Send request
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http_post: request to %q: %w", url, err)
	}
	defer resp.Body.Close()

	// 中文：读取响应体（限制大小）
	// English: Read response body (with size limit)
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxHTTPResponseSize))
	if err != nil {
		return "", fmt.Errorf("http_post: read response: %w", err)
	}

	// 中文：对 4xx/5xx 返回错误
	// English: Return error for 4xx/5xx status codes
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http_post: server returned status %d for %q", resp.StatusCode, url)
	}

	// 中文：格式化返回结果
	// English: Format result
	result := fmt.Sprintf("HTTP %d\nURL: %s\n\n%s", resp.StatusCode, url, string(bodyBytes))
	return result, nil
}
