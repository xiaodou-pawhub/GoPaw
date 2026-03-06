package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&HTTPClientTool{})
}

type HTTPClientTool struct{}

func (t *HTTPClientTool) Name() string { return "http_client" }

func (t *HTTPClientTool) Description() string {
	return "Access external real-time data or interact with web services. " +
		"WHEN TO USE: Use this when the user asks for live information (weather, stocks, crypto prices, news), " +
		"needs to trigger a webhook, or mentions a specific API. If you need to verify a piece of information " +
		"that might have changed since your training data, use this or web_search. " +
		"Non-GET requests require user approval."
}

func (t *HTTPClientTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"method": {
				Type:        "string",
				Description: "HTTP method (GET, POST, PUT, DELETE, PATCH).",
				Enum:        []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
			},
			"url": {
				Type:        "string",
				Description: "The full target URL (must start with http:// or https://).",
			},
			"headers": {
				Type:        "object",
				Description: "Key-value pairs for HTTP headers (e.g. {'Content-Type': 'application/json'}).",
			},
			"body": {
				Type:        "string",
				Description: "The request body string (for POST/PUT).",
			},
			"timeout": {
				Type:        "integer",
				Description: "Optional timeout in seconds (default 30).",
			},
		},
		Required: []string{"method", "url"},
	}
}

// GuardedTool implementation: Force approval for any state-changing requests.
func (t *HTTPClientTool) RequireApproval(args map[string]interface{}) bool {
	method, _ := args["method"].(string)
	method = strings.ToUpper(method)
	// GET is considered safe (read-only). Others require approval.
	return method != "GET" && method != ""
}

func (t *HTTPClientTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	method, _ := args["method"].(string)
	targetURL, _ := args["url"].(string)
	headers, _ := args["headers"].(map[string]interface{})
	bodyStr, _ := args["body"].(string)
	timeoutSec, _ := args["timeout"].(float64)

	if timeoutSec <= 0 {
		timeoutSec = 30
	}

	// 1. Basic URL validation
	u, err := url.Parse(targetURL)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("invalid URL: %v", err))
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return plugin.ErrorResult("URL must use http or https scheme")
	}

	// 2. SSRF Protection: Deny local/private IP addresses
	if err := t.isSafeURL(u); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("security block: %v", err))
	}

	// 3. Prepare Request
	var bodyReader io.Reader
	if bodyStr != "" {
		bodyReader = strings.NewReader(bodyStr)
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), targetURL, bodyReader)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to create request: %v", err))
	}

	// Set Headers
	for k, v := range headers {
		req.Header.Set(k, fmt.Sprintf("%v", v))
	}
	if _, ok := req.Header["User-Agent"]; !ok {
		req.Header.Set("User-Agent", "GoPaw-Agent/1.0")
	}

	// 4. Execution
	client := &http.Client{
		Timeout: time.Duration(timeoutSec) * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("request failed: %v", err))
	}
	defer resp.Body.Close()

	// 5. Process Response
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1024*1024)) // Limit to 1MB
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to read response: %v", err))
	}

	result := struct {
		StatusCode int                 `json:"status_code"`
		Headers    map[string][]string `json:"headers"`
		Body       string              `json:"body"`
	}{
		StatusCode: resp.StatusCode,
		Headers:    t.filterHeaders(resp.Header),
		Body:       string(respBody),
	}

	jsonRes, _ := json.MarshalIndent(result, "", "  ")
	return plugin.NewToolResult(string(jsonRes))
}

func (t *HTTPClientTool) filterHeaders(h http.Header) map[string][]string {
	// Remove sensitive or noisy headers from output
	out := make(map[string][]string)
	for k, v := range h {
		lowKey := strings.ToLower(k)
		if strings.Contains(lowKey, "cookie") || strings.Contains(lowKey, "set-cookie") || strings.Contains(lowKey, "auth") {
			continue
		}
		out[k] = v
	}
	return out
}

func (t *HTTPClientTool) isSafeURL(u *url.URL) error {
	host := u.Hostname()
	ips, err := net.LookupIP(host)
	if err != nil {
		// If we can't resolve, assume it's external or let the dialer fail
		return nil 
	}

	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsPrivate() {
			return fmt.Errorf("access to private network address %s is prohibited", ip.String())
		}
	}
	return nil
}
