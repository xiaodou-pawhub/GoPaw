package builtin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&WebPostTool{})
}

type WebPostTool struct{}

func (t *WebPostTool) Name() string { return "web_post" }

func (t *WebPostTool) Description() string {
	return "Send a POST request to a URL with a JSON payload."
}

func (t *WebPostTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"url": {
				Type:        "string",
				Description: "The target URL.",
			},
			"payload": {
				Type:        "string",
				Description: "The JSON string to send as the request body.",
			},
		},
		Required: []string{"url", "payload"},
	}
}

func (t *WebPostTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	url, _ := args["url"].(string)
	payload, _ := args["payload"].(string)

	client := &http.Client{Timeout: 30 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "GoPaw/0.1.0 Agent")

	resp, err := client.Do(req)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("post failed: %v", err))
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 400 {
		return plugin.ErrorResult(fmt.Sprintf("server returned status %d: %s", resp.StatusCode, string(body)))
	}

	return plugin.NewToolResult(string(body))
}
