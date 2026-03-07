package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&BrowserControlTool{})
}

type BrowserControlTool struct {
	store   plugin.MediaStore
	session string
}

func (t *BrowserControlTool) Name() string { return "browser_control" }

func (t *BrowserControlTool) Description() string {
	return "Control a headless web browser to interact with websites. " +
		"Actions: 'navigate' (open URL), 'click' (click element), 'type' (input text), " +
		"'scroll' (scroll down), 'screenshot' (capture view), 'extract' (get page content), " +
		"'get_cookies' (get session cookies for http_client synergy). " +
		"This tool persists session data, so once logged in, you stay logged in."
}

func (t *BrowserControlTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"action": {
				Type:        "string",
				Description: "The action to perform.",
				Enum:        []string{"navigate", "click", "type", "scroll", "screenshot", "extract", "get_cookies"},
			},
			"url": {
				Type:        "string",
				Description: "URL for 'navigate' action.",
			},
			"selector": {
				Type:        "string",
				Description: "CSS selector for 'click' or 'type' actions.",
			},
			"text": {
				Type:        "string",
				Description: "Text content for the 'type' action.",
			},
			"wait_seconds": {
				Type:        "integer",
				Description: "Seconds to wait after the action for the page to stabilize. Default 2.",
			},
		},
		Required: []string{"action"},
	}
}

func (t *BrowserControlTool) SetMediaStore(s plugin.MediaStore) {
	t.store = s
}

func (t *BrowserControlTool) SetContext(channel, chatID, session, user string) {
	t.session = session
}

func (t *BrowserControlTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	action, _ := args["action"].(string)
	targetURL, _ := args["url"].(string)
	selector, _ := args["selector"].(string)
	text, _ := args["text"].(string)
	waitSec, _ := args["wait_seconds"].(float64)

	if waitSec <= 0 {
		waitSec = 2
	}

	// 1. Setup allocator: remote Chrome (Docker sidecar) or local Chrome (development)
	var allocCtx context.Context
	var allocCancel context.CancelFunc
	if chromeURL := os.Getenv("CHROME_URL"); chromeURL != "" {
		// Docker mode: connect to headless-shell sidecar via CDP WebSocket
		allocCtx, allocCancel = chromedp.NewRemoteAllocator(context.Background(), chromeURL)
	} else {
		// Dev mode: launch a local Chrome process with a persistent user data dir
		home, _ := os.UserHomeDir()
		userDataDir := filepath.Join(home, ".gopaw", "browser_data")
		_ = os.MkdirAll(userDataDir, 0755)
		opts := append(chromedp.DefaultExecAllocatorOptions[:],
			chromedp.UserDataDir(userDataDir),
			chromedp.Flag("headless", true),
			chromedp.Flag("disable-gpu", true),
			chromedp.Flag("no-sandbox", true),
		)
		allocCtx, allocCancel = chromedp.NewExecAllocator(context.Background(), opts...)
	}
	defer allocCancel()

	// 2. Create browser context
	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	executeCtx, executeCancel := context.WithTimeout(browserCtx, 60*time.Second)
	defer executeCancel()

	// 3. Define Actions
	var tasks chromedp.Tasks
	var resultMsg string
	var extraData string
	var cookies []*network.Cookie

	// Set viewport to a standard desktop resolution to avoid layout breakage
	tasks = append(tasks, chromedp.EmulateViewport(1280, 800))

	switch action {
	case "navigate":
		if targetURL == "" {
			return plugin.ErrorResult("URL is required for navigate")
		}
		// WaitReady ensures the DOM is interactive before proceeding
		tasks = append(tasks,
			chromedp.Navigate(targetURL),
			chromedp.WaitReady("body", chromedp.ByQuery),
		)
		resultMsg = fmt.Sprintf("Navigated to %s", targetURL)
	case "click":
		if selector == "" {
			return plugin.ErrorResult("selector is required for click")
		}
		tasks = append(tasks, chromedp.WaitVisible(selector), chromedp.Click(selector))
		resultMsg = fmt.Sprintf("Clicked on %s", selector)
	case "type":
		if selector == "" || text == "" {
			return plugin.ErrorResult("selector and text are required for type")
		}
		tasks = append(tasks, chromedp.WaitVisible(selector), chromedp.SendKeys(selector, text))
		resultMsg = fmt.Sprintf("Typed text into %s", selector)
	case "scroll":
		tasks = append(tasks, chromedp.Evaluate(`window.scrollBy(0, 500)`, nil))
		resultMsg = "Scrolled down"
	case "extract":
		var html string
		var innerText string
		tasks = append(tasks,
			chromedp.OuterHTML("html", &html),
			chromedp.Evaluate(`document.body.innerText`, &innerText),
		)
		resultMsg = "Content extracted"
		// Capture first 5000 chars of text to avoid context bloat
		if len(innerText) > 5000 {
			innerText = innerText[:5000] + "..."
		}
		extraData = fmt.Sprintf("\n\n--- Extracted Text ---\n%s", innerText)
	case "get_cookies":
		tasks = append(tasks, chromedp.ActionFunc(func(ctx context.Context) error {
			var err error
			cookies, err = network.GetCookies().Do(ctx)
			return err
		}))
		resultMsg = "Cookies retrieved"
	case "screenshot":
		resultMsg = "Screenshot captured"
	default:
		return plugin.ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}

	// 4. Feedback screenshot
	var buf []byte
	tasks = append(tasks, chromedp.Sleep(time.Duration(waitSec)*time.Second))
	tasks = append(tasks, chromedp.CaptureScreenshot(&buf))

	// 5. Run Chromedp
	if err := chromedp.Run(executeCtx, tasks); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("browser error: %v", err))
	}

	// Format Cookies if requested
	if action == "get_cookies" {
		var cookieParts []string
		for _, c := range cookies {
			cookieParts = append(cookieParts, fmt.Sprintf("%s=%s", c.Name, c.Value))
		}
		extraData = fmt.Sprintf("\n\n--- Session Cookies ---\nUse this in http_client headers as 'Cookie':\n%s", strings.Join(cookieParts, "; "))
	}

	// 6. Save Screenshot to Store
	tmpPath := t.store.TempPath(".png")
	_ = os.WriteFile(tmpPath, buf, 0644)

	ref, err := t.store.Store(tmpPath, plugin.MediaMeta{
		Filename:    fmt.Sprintf("browser_%s.png", action),
		ContentType: "image/png",
		Source:      "tool:browser_control",
	}, t.session)

	userOutput := fmt.Sprintf("🌐 浏览器操作完成：%s", resultMsg)
	llmOutput := fmt.Sprintf("%s. Result screenshot: %s%s", resultMsg, ref, extraData)

	if err != nil {
		_ = os.Remove(tmpPath)
		return plugin.NewToolResult(llmOutput + " (but failed to save screenshot)")
	}

	return &plugin.ToolResult{
		LLMOutput:  llmOutput,
		UserOutput: userOutput,
		Media:      []string{ref},
	}
}
