// Package tools provides the built-in Tool implementations for GoPaw.
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&WebSearchTool{})
}

const tavilySearchURL = "https://api.tavily.com/search"

// WebSearchTool performs web searches using the Tavily API.
type WebSearchTool struct{}

func (t *WebSearchTool) Name() string { return "web_search" }

func (t *WebSearchTool) Description() string {
	return "Search the web for up-to-date information. Returns a list of relevant results with titles, URLs and snippets."
}

func (t *WebSearchTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"query": {
				Type:        "string",
				Description: "The search query string.",
			},
			"limit": {
				Type:        "integer",
				Description: "Maximum number of results to return (default: 5, max: 10).",
			},
		},
		Required: []string{"query"},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("web_search: 'query' argument is required")
	}

	limit := 5
	if v, ok := args["limit"]; ok {
		switch lv := v.(type) {
		case float64:
			limit = int(lv)
		case int:
			limit = lv
		}
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	apiKey := os.Getenv("TAVILY_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("web_search: TAVILY_API_KEY environment variable is not set")
	}

	payload := map[string]interface{}{
		"api_key":          apiKey,
		"query":            query,
		"max_results":      limit,
		"include_answer":   false,
		"search_depth":     "basic",
	}
	body, _ := json.Marshal(payload)

	httpCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(httpCtx, http.MethodPost, tavilySearchURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("web_search: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("web_search: http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("web_search: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("web_search: api error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("web_search: parse response: %w", err)
	}

	if len(result.Results) == 0 {
		return "No results found.", nil
	}

	var sb strings.Builder
	for i, r := range result.Results {
		sb.WriteString(fmt.Sprintf("%d. **%s**\n   URL: %s\n   %s\n\n", i+1, r.Title, r.URL, r.Content))
	}
	return sb.String(), nil
}
