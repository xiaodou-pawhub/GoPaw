package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&MemoryWriteTool{})
}

// MemoryWriteTool 将内容写入 memory/ 目录下的记忆文件。
type MemoryWriteTool struct{}

func (t *MemoryWriteTool) Name() string { return "memory_write" }

func (t *MemoryWriteTool) Description() string {
	return "Write or append content to a memory file in the memory directory. " +
		"Use this to remember important facts, user preferences, or project information across sessions. " +
		"Default file is MEMORY.md (the core memory index). " +
		"Supports sub-files like 'preferences.md' or 'projects/gopaw.md'."
}

func (t *MemoryWriteTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"content": {
				Type:        "string",
				Description: "The content to write. Use markdown format.",
			},
			"file": {
				Type:        "string",
				Description: "Target file relative to memory dir, e.g. 'MEMORY.md', 'preferences.md', 'projects/gopaw.md'. Defaults to 'MEMORY.md'.",
			},
			"mode": {
				Type:        "string",
				Description: "Write mode: 'append' (default) adds content to end of file, 'replace' overwrites the entire file.",
			},
		},
		Required: []string{"content"},
	}
}

func (t *MemoryWriteTool) Execute(_ context.Context, args map[string]interface{}) (string, error) {
	memDir := getMemoryDir()
	if memDir == "" {
		return "", fmt.Errorf("memory_write: memory directory not configured")
	}

	content, ok := args["content"].(string)
	if !ok || strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("memory_write: 'content' argument is required")
	}

	file := "MEMORY.md"
	if f, ok := args["file"].(string); ok && f != "" {
		file = f
	}

	mode := "append"
	if m, ok := args["mode"].(string); ok && m == "replace" {
		mode = "replace"
	}

	// 安全校验：禁止路径穿越
	target := filepath.Join(memDir, file)
	if !strings.HasPrefix(target, memDir) {
		return "", fmt.Errorf("memory_write: invalid file path %q", file)
	}

	// 创建父目录
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return "", fmt.Errorf("memory_write: create dirs: %w", err)
	}

	// 写入前备份（仅对 MEMORY.md 做快照）
	if file == "MEMORY.md" {
		if err := snapshotMemoryMD(memDir, target); err != nil {
			// 备份失败不阻断写入，忽略错误
			_ = err
		}
	}

	if mode == "replace" {
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return "", fmt.Errorf("memory_write: write %q: %w", file, err)
		}
		return fmt.Sprintf("已覆盖写入 %s（%d 字节）", file, len(content)), nil
	}

	// append 模式：追加到文件末尾
	f, err := os.OpenFile(target, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("memory_write: open %q: %w", file, err)
	}
	defer f.Close()

	// 追加前确保换行分隔
	entry := "\n" + strings.TrimRight(content, "\n") + "\n"
	if _, err := f.WriteString(entry); err != nil {
		return "", fmt.Errorf("memory_write: append to %q: %w", file, err)
	}

	return fmt.Sprintf("已追加 %d 字节到 %s", len(entry), file), nil
}

// snapshotMemoryMD 在写入前将 MEMORY.md 备份到 .snapshot/ 目录。
func snapshotMemoryMD(memDir, target string) error {
	data, err := os.ReadFile(target)
	if err != nil {
		return nil // 文件不存在时跳过
	}
	snapDir := filepath.Join(memDir, ".snapshot")
	if err := os.MkdirAll(snapDir, 0o755); err != nil {
		return err
	}
	snapName := "MEMORY_" + time.Now().Format("20060102_150405") + ".md"
	return os.WriteFile(filepath.Join(snapDir, snapName), data, 0o644)
}
