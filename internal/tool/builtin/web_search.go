package builtin

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

type WebSearchTool struct{}

func (t *WebSearchTool) Name() string { return "web_search" }

func (t *WebSearchTool) Description() string {
	return "Search the web for information. Automatically selects the best available provider " +
		"(Brave, Tavily, or Serper). Returns structured summaries."
}

func (t *WebSearchTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"query": {
				Type:        "string",
				Description: "The search query.",
			},
		},
		Required: []string{"query"},
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	query, _ := args["query"].(string)
	if query == "" {
		return plugin.ErrorResult("query cannot be empty")
	}

	// 1. Try Brave Search (Preferred)
	if key := os.Getenv("BRAVE_API_KEY"); key != "" {
		return t.searchBrave(ctx, query, key)
	}

	// 2. Try Tavily (AI Optimized)
	if key := os.Getenv("TAVILY_API_KEY"); key != "" {
		return t.searchTavily(ctx, query, key)
	}

	// 3. Try Serper (Google Wrapper)
	if key := os.Getenv("SERPER_API_KEY"); key != "" {
		return t.searchSerper(ctx, query, key)
	}

	// 4. Try Baidu Qianfan (Chinese Optimized)
	if ak := os.Getenv("BAIDU_QIANFAN_AK"); ak != "" {
		if sk := os.Getenv("BAIDU_QIANFAN_SK"); sk != "" {
			return t.searchBaidu(ctx, query, ak, sk)
		}
	}

	// No keys found
	return plugin.ErrorResult(
		"No web search API keys configured. Please set one of the following environment variables:\n" +
			"- BRAVE_API_KEY (Get free at https://api.search.brave.com/app/keys)\n" +
			"- TAVILY_API_KEY (Get free at https://tavily.com)\n" +
			"- SERPER_API_KEY (Get free at https://serper.dev)\n" +
			"- BAIDU_QIANFAN_AK & BAIDU_QIANFAN_SK (Get free at https://console.bce.baidu.com)",
	)
}

// --- Providers ---

func (t *WebSearchTool) searchBaidu(ctx context.Context, query, ak, sk string) *plugin.ToolResult {
	// 1. Get Access Token
	tokenURL := fmt.Sprintf("https://aip.baidubce.com/oauth/2.0/token?grant_type=client_credentials&client_id=%s&client_secret=%s", ak, sk)
	reqToken, _ := http.NewRequestWithContext(ctx, "POST", tokenURL, nil)
	
	client := &http.Client{Timeout: 10 * time.Second}
	respToken, err := client.Do(reqToken)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("baidu token request failed: %v", err))
	}
	defer respToken.Body.Close()

	var tokenRes struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(respToken.Body).Decode(&tokenRes); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("baidu token decode failed: %v", err))
	}

	// 2. Call Search Plugin (Using Qianfan Knowledge Search API as an example)
	// Note: The specific endpoint depends on the exact plugin enabled. 
	// Here we use a generic Knowledge/Search interface or AppBuilder generic search.
	// For standard web search, we might fallback to Serper if Baidu doesn't expose a raw web search API easily.
	// Assuming usage of "Baidu Search" plugin via AppBuilder logic or similar.
	
	// Since raw Baidu Web Search API is not publicly standard like Bing/Google,
	// we will placeholder this to prompt user if they strictly want Baidu.
	// A more stable approach for Baidu results is via Serper (which supports gl=cn).
	
	return plugin.ErrorResult("Baidu Qianfan integration requires a specific AppID. For Chinese results, configured Serper with 'gl=cn' is recommended.") 
}

func (t *WebSearchTool) searchBrave(ctx context.Context, query, key string) *plugin.ToolResult {
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://api.search.brave.com/res/v1/web/search?q="+query, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", key)

	resp, err := client.Do(req)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("brave search failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return plugin.ErrorResult(fmt.Sprintf("brave api error (%d): %s", resp.StatusCode, string(body)))
	}

	var res struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				Description string `json:"description"`
				URL         string `json:"url"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("brave decode failed: %v", err))
	}

	var sb strings.Builder
	sb.WriteString("### Search Results (Brave)\n\n")
	for i, item := range res.Web.Results {
		if i >= 5 {
			break
		}
		fmt.Fprintf(&sb, "**%d. [%s](%s)**\n%s\n\n", i+1, item.Title, item.URL, item.Description)
	}
	return plugin.NewToolResult(sb.String())
}

func (t *WebSearchTool) searchTavily(ctx context.Context, query, key string) *plugin.ToolResult {
	payload := map[string]interface{}{
		"query":       query,
		"search_depth": "basic",
		"max_results":  5,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.tavily.com/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", key) // Tavily uses api-key header or body param

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("tavily search failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return plugin.ErrorResult(fmt.Sprintf("tavily api error (%d)", resp.StatusCode))
	}

	var res struct {
		Results []struct {
			Title   string `json:"title"`
			Content string `json:"content"`
			URL     string `json:"url"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("tavily decode failed: %v", err))
	}

	var sb strings.Builder
	sb.WriteString("### Search Results (Tavily)\n\n")
	for i, item := range res.Results {
		fmt.Fprintf(&sb, "**%d. [%s](%s)**\n%s\n\n", i+1, item.Title, item.URL, item.Content)
	}
	return plugin.NewToolResult(sb.String())
}

func (t *WebSearchTool) searchSerper(ctx context.Context, query, key string) *plugin.ToolResult {
	payload := map[string]interface{}{
		"q": query,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(ctx, "POST", "https://google.serper.dev/search", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", key)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("serper search failed: %v", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return plugin.ErrorResult(fmt.Sprintf("serper api error (%d)", resp.StatusCode))
	}

	var res struct {
		Organic []struct {
			Title   string `json:"title"`
			Snippet string `json:"snippet"`
			Link    string `json:"link"`
		} `json:"organic"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("serper decode failed: %v", err))
	}

	var sb strings.Builder
	sb.WriteString("### Search Results (Serper/Google)\n\n")
	for i, item := range res.Organic {
		if i >= 5 {
			break
		}
		fmt.Fprintf(&sb, "**%d. [%s](%s)**\n%s\n\n", i+1, item.Title, item.Link, item.Snippet)
	}
	return plugin.NewToolResult(sb.String())
}
