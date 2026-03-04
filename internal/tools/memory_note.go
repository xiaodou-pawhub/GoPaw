package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

var (
	memoryNotesDirMu sync.RWMutex
	memoryNotesDir   string
)

// SetMemoryNotesDir 设置每日笔记目录（memory/notes/），应在 main.go 中调用。
func SetMemoryNotesDir(dir string) {
	memoryNotesDirMu.Lock()
	defer memoryNotesDirMu.Unlock()
	memoryNotesDir = dir
}

func getMemoryNotesDir() string {
	memoryNotesDirMu.RLock()
	defer memoryNotesDirMu.RUnlock()
	return memoryNotesDir
}

func init() {
	tool.Register(&MemoryNoteTool{})
}

// MemoryNoteTool appends a note to today's daily notes file.
type MemoryNoteTool struct{}

func (t *MemoryNoteTool) Name() string { return "memory_note" }

func (t *MemoryNoteTool) Description() string {
	return "Append a note to today's daily notes file (memory/notes/YYYYMM/YYYYMMDD.md). " +
		"Use this for ephemeral observations, reminders, or context specific to today. " +
		"Unlike memory_store (which persists forever), daily notes are automatically archived after 30 days."
}

func (t *MemoryNoteTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"content": {
				Type:        "string",
				Description: "The note content to append to today's daily notes.",
			},
		},
		Required: []string{"content"},
	}
}

func (t *MemoryNoteTool) Execute(ctx context.Context, params map[string]any) (string, error) {
	notesDir := getMemoryNotesDir()
	if notesDir == "" {
		return "", fmt.Errorf("memory_note: notes directory not initialized")
	}

	content, _ := params["content"].(string)
	if content == "" {
		return "", fmt.Errorf("memory_note: 'content' is required")
	}

	now := time.Now()
	// monthly sub-directory: YYYYMM
	monthDir := filepath.Join(notesDir, now.Format("200601"))
	if err := os.MkdirAll(monthDir, 0o755); err != nil {
		return "", fmt.Errorf("memory_note: create month dir: %w", err)
	}

	// daily file: YYYYMMDD.md
	dayFile := filepath.Join(monthDir, now.Format("20060102")+".md")

	var header string
	if _, err := os.Stat(dayFile); os.IsNotExist(err) {
		// New file: write date heading
		header = fmt.Sprintf("# %s\n\n", now.Format("2006-01-02"))
	}

	entry := fmt.Sprintf("- [%s] %s\n", now.Format("15:04"), content)

	f, err := os.OpenFile(dayFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("memory_note: open file: %w", err)
	}
	defer f.Close()

	if header != "" {
		if _, err := f.WriteString(header); err != nil {
			return "", fmt.Errorf("memory_note: write header: %w", err)
		}
	}
	if _, err := f.WriteString(entry); err != nil {
		return "", fmt.Errorf("memory_note: write entry: %w", err)
	}

	return fmt.Sprintf("Note appended to %s", now.Format("2006-01-02")), nil
}
