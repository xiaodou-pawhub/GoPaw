package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gopaw/gopaw/internal/llm"
)

// HookPreReasoning is called before each LLM call.
// It receives the current message slice and returns a (possibly modified) slice.
// Returning an error aborts the agent loop immediately.
type HookPreReasoning func(ctx context.Context, messages []llm.ChatMessage) ([]llm.ChatMessage, error)

// HookPostTool is called after each tool execution.
// toolName is the tool that ran, output is its string result, execErr is non-nil on failure.
// Post-tool hooks are fire-and-forget: their return value is ignored.
type HookPostTool func(ctx context.Context, toolName, output string, execErr error)

// Hooks bundles all optional callbacks injected into the agent loop.
// Multiple hooks of the same type are executed in registration order.
type Hooks struct {
	PreReasoning []HookPreReasoning
	PostTool     []HookPostTool
}

// runPreReasoning executes all pre-reasoning hooks in order, threading messages through.
func (h *Hooks) runPreReasoning(ctx context.Context, messages []llm.ChatMessage) ([]llm.ChatMessage, error) {
	for _, fn := range h.PreReasoning {
		var err error
		messages, err = fn(ctx, messages)
		if err != nil {
			return nil, fmt.Errorf("pre-reasoning hook: %w", err)
		}
	}
	return messages, nil
}

// runPostTool executes all post-tool hooks. Panics are recovered and silently dropped.
func (h *Hooks) runPostTool(ctx context.Context, toolName, output string, execErr error) {
	for _, fn := range h.PostTool {
		func() {
			defer func() { recover() }() //nolint:errcheck
			fn(ctx, toolName, output, execErr)
		}()
	}
}

// ── Built-in hooks ──────────────────────────────────────────────────────────

// AutoJournalHook returns a PostTool hook that appends a brief record of each
// tool execution to today's daily memory note (notes/YYYYMM/YYYYMMDD.md).
// Memory management tools (prefix "memory_") are skipped to prevent feedback loops.
func AutoJournalHook(notesDir string) HookPostTool {
	return func(_ context.Context, toolName, output string, execErr error) {
		if strings.HasPrefix(toolName, "memory_") {
			return
		}

		now := time.Now()
		monthDir := filepath.Join(notesDir, now.Format("200601"))
		if err := os.MkdirAll(monthDir, 0o755); err != nil {
			return
		}
		dayFile := filepath.Join(monthDir, now.Format("20060102")+".md")

		status := "ok"
		if execErr != nil {
			status = "err"
		}

		preview := truncate(output, 200)
		entry := fmt.Sprintf("- [%s] `%s` [%s] %s\n", now.Format("15:04"), toolName, status, preview)

		// Write a date heading if the file does not yet exist.
		flags := os.O_APPEND | os.O_WRONLY
		if _, err := os.Stat(dayFile); os.IsNotExist(err) {
			flags = os.O_CREATE | os.O_WRONLY
			entry = fmt.Sprintf("# %s\n\n", now.Format("2006-01-02")) + entry
		}

		f, err := os.OpenFile(dayFile, flags|os.O_CREATE, 0o644)
		if err != nil {
			return
		}
		defer f.Close()
		_, _ = f.WriteString(entry)
	}
}

// InjectCurrentTime is a pre-reasoning hook that appends the current date and
// time to the system message on every LLM call. This ensures the model always
// has accurate temporal context without baking a fixed timestamp into AGENT.md.
func InjectCurrentTime() HookPreReasoning {
	return func(_ context.Context, messages []llm.ChatMessage) ([]llm.ChatMessage, error) {
		now := time.Now().Format("2006-01-02 15:04 MST")
		suffix := fmt.Sprintf("\n\n[当前时间: %s]", now)

		result := make([]llm.ChatMessage, len(messages))
		copy(result, messages)
		for i, m := range result {
			if m.Role == llm.RoleSystem {
				result[i].Content = m.Content + suffix
				break
			}
		}
		return result, nil
	}
}
