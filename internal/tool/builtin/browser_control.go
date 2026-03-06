package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&BrowserControlTool{})
}

type BrowserControlTool struct {
	store      plugin.MediaStore
	session    string
	baseDir    string // Base workspace directory for browser data
}

func (t *BrowserControlTool) Name() string { return "browser_control" }

func (t *BrowserControlTool) Description() string {
	return "Control a headless web browser to interact with websites. " +
		"Actions: 'navigate' (open URL), 'click' (click element by selector), " +
		"'type' (input text into selector), 'scroll' (scroll down), 'screenshot' (capture view). " +
		"This tool persists session data (cookies/login), so once logged in, you stay logged in."
}

func (t *BrowserControlTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"action": {
				Type:        "string",
				Description: "The action to perform.",
				Enum:        []string{"navigate", "click", "type", "scroll", "screenshot"},
			},
			"url": {
				Type:        "string",
				Description: "URL for 'navigate' action.",
			},
			"selector": {
				Type:        "string",
				Description: "CSS selector for 'click' or 'type' actions (e.g. 'button.submit', '#username').",
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
	// We need the root workspace to store browser data. 
	// We'll infer it from current context if not explicitly set.
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

	// 1. Setup Persistent Data Directory
	// We use ~/.gopaw/browser_data to keep cookies/sessions
	home, _ := os.UserHomeDir()
	userDataDir := filepath.Join(home, ".gopaw", "browser_data")
	_ = os.MkdirAll(userDataDir, 0755)

	// 2. Browser Allocator with Persistence
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.UserDataDir(userDataDir),
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)
	defer browserCancel()

	executeCtx, executeCancel := context.WithTimeout(browserCtx, 60*time.Second)
	defer executeCancel()

	// 3. Define Actions
	var tasks chromedp.Tasks
	var resultMsg string

	switch action {
	case "navigate":
		if targetURL == "" { return plugin.ErrorResult("URL is required for navigate") }
		tasks = append(tasks, chromedp.Navigate(targetURL))
		resultMsg = fmt.Sprintf("Navigated to %s", targetURL)
	case "click":
		if selector == "" { return plugin.ErrorResult("selector is required for click") }
		tasks = append(tasks, chromedp.WaitVisible(selector), chromedp.Click(selector))
		resultMsg = fmt.Sprintf("Clicked on %s", selector)
	case "type":
		if selector == "" || text == "" { return plugin.ErrorResult("selector and text are required for type") }
		tasks = append(tasks, chromedp.WaitVisible(selector), chromedp.SendKeys(selector, text))
		resultMsg = fmt.Sprintf("Typed text into %s", selector)
	case "scroll":
		tasks = append(tasks, chromedp.Evaluate(`window.scrollBy(0, 500)`, nil))
		resultMsg = "Scrolled down"
	case "screenshot":
		resultMsg = "Screenshot captured"
	default:
		return plugin.ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}

	// 4. Always add a wait and a screenshot for feedback (except maybe pure navigation)
	var buf []byte
	tasks = append(tasks, chromedp.Sleep(time.Duration(waitSec)*time.Second))
	tasks = append(tasks, chromedp.CaptureScreenshot(&buf))

	// 5. Run Chromedp
	if err := chromedp.Run(executeCtx, tasks); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("browser error: %v", err))
	}

	// 6. Save Screenshot to Store
	tmpPath := t.store.TempPath(".png")
	_ = os.WriteFile(tmpPath, buf, 0644)
	
	ref, err := t.store.Store(tmpPath, plugin.MediaMeta{
		Filename:    fmt.Sprintf("browser_%s.png", action),
		ContentType: "image/png",
		Source:      "tool:browser_control",
	}, t.session)

	if err != nil {
		_ = os.Remove(tmpPath)
		return plugin.NewToolResult(resultMsg + " (but failed to save screenshot)")
	}

	return &plugin.ToolResult{
		LLMOutput:  fmt.Sprintf("%s. Result screenshot: %s", resultMsg, ref),
		UserOutput: fmt.Sprintf("🌐 浏览器操作完成：%s", resultMsg),
		Media:      []string{ref},
	}
}
