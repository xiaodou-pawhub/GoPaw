package builtin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&WebFetchTool{})
}

type WebFetchTool struct{}

func (t *WebFetchTool) Name() string { return "web_fetch" }

func (t *WebFetchTool) Description() string {
	return "Fetch the content of a web page and convert it to clean Markdown. Preferred for reading documentation or articles."
}

func (t *WebFetchTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"url": {
				Type:        "string",
				Description: "The URL to fetch.",
			},
			"raw": {
				Type:        "boolean",
				Description: "If true, returns the raw HTML/text without Markdown conversion.",
			},
		},
		Required: []string{"url"},
	}
}

func (t *WebFetchTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	url, _ := args["url"].(string)
	rawMode, _ := args["raw"].(bool)

	targetURL := url
	// Use Jina Reader for automatic HTML-to-Markdown conversion if not in raw mode.
	if !rawMode && !strings.HasPrefix(url, "https://r.jina.ai/") {
		targetURL = "https://r.jina.ai/" + url
	}

	client := &http.Client{Timeout: 45 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	req.Header.Set("User-Agent", "GoPaw/0.1.0 Agent")
	
	// Optional: Header for Jina to get better results
	// req.Header.Set("X-With-Generated-Alt", "true")

	resp, err := client.Do(req)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("fetch failed: %v", err))
	}
	defer resp.Body.Close()

	// 2MB limit for web content
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to read response: %v", err))
	}

	if resp.StatusCode >= 400 {
		// If Jina failed, fallback to raw fetch once
		if !rawMode && strings.HasPrefix(targetURL, "https://r.jina.ai/") {
			return t.Execute(ctx, map[string]interface{}{"url": url, "raw": true})
		}
		return plugin.ErrorResult(fmt.Sprintf("server returned status %d", resp.StatusCode))
	}

	return plugin.NewToolResult(string(body))
}
