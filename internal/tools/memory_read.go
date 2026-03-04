package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

// memoryDir 是 memory/ 目录的全局路径，由 SetMemoryDir 设置。
var (
	memoryDirMu sync.RWMutex
	memoryDir   string
)

// SetMemoryDir 设置记忆目录路径，应在 main.go 中 workspace 解析后调用。
func SetMemoryDir(dir string) {
	memoryDirMu.Lock()
	defer memoryDirMu.Unlock()
	memoryDir = dir
}

// getMemoryDir 返回当前记忆目录路径。
func getMemoryDir() string {
	memoryDirMu.RLock()
	defer memoryDirMu.RUnlock()
	return memoryDir
}

func init() {
	tool.Register(&MemoryReadTool{})
}

// MemoryReadTool 读取 memory/ 目录下的记忆文件。
type MemoryReadTool struct{}

func (t *MemoryReadTool) Name() string { return "memory_read" }

func (t *MemoryReadTool) Description() string {
	return "Read a memory file or list all memory files. " +
		"Use file='list' to see all available memory files. " +
		"Use file='MEMORY.md' (default) to read the core memory index. " +
		"Use file='archive/2026-03.md' to read archived conversation summaries."
}

func (t *MemoryReadTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"file": {
				Type:        "string",
				Description: "File to read relative to memory dir, e.g. 'MEMORY.md', 'preferences.md'. Use 'list' to list all files. Defaults to 'MEMORY.md'.",
			},
		},
		Required: []string{},
	}
}

func (t *MemoryReadTool) Execute(_ context.Context, args map[string]interface{}) (string, error) {
	memDir := getMemoryDir()
	if memDir == "" {
		return "", fmt.Errorf("memory_read: memory directory not configured")
	}

	file := "MEMORY.md"
	if f, ok := args["file"].(string); ok && f != "" {
		file = f
	}

	// 列出所有记忆文件
	if file == "list" {
		return listMemoryFiles(memDir)
	}

	// 安全校验：禁止路径穿越
	target := filepath.Join(memDir, file)
	if !strings.HasPrefix(target, memDir) {
		return "", fmt.Errorf("memory_read: invalid file path %q", file)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("文件 %q 不存在", file), nil
		}
		return "", fmt.Errorf("memory_read: read %q: %w", file, err)
	}

	return string(data), nil
}

// listMemoryFiles 列出 memory/ 目录下所有可读文件（忽略 .snapshot/）。
func listMemoryFiles(memDir string) (string, error) {
	var files []string
	err := filepath.Walk(memDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// 跳过 .snapshot 目录
		if info.IsDir() && info.Name() == ".snapshot" {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			rel, _ := filepath.Rel(memDir, path)
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("memory_read: list files: %w", err)
	}
	if len(files) == 0 {
		return "memory/ 目录下暂无记忆文件", nil
	}
	return "memory/ 目录下的文件：\n- " + strings.Join(files, "\n- "), nil
}
