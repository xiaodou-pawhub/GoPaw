// Package workspace resolves and provides all runtime data paths under the workspace directory.
package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Paths holds resolved absolute paths for all workspace data.
type Paths struct {
	Root             string // workspace root, e.g. ~/.gopaw
	DBFile           string // ~/.gopaw/gopaw.db
	MemoriesDBFile   string // ~/.gopaw/memories.db  (long-term structured memory)
	AgentMDFile      string // ~/.gopaw/agent/AGENT.md
	PersonaMDFile    string // ~/.gopaw/agent/PERSONA.md
	ContextMDFile    string // ~/.gopaw/agent/CONTEXT.md
	MemoryDir        string // ~/.gopaw/memory/
	MemoryMDFile     string // ~/.gopaw/memory/MEMORY.md
	MemoryArchDir    string // ~/.gopaw/memory/archive/
	MemorySnapDir    string // ~/.gopaw/memory/.snapshot/
	MemoryNotesDir   string // ~/.gopaw/memory/notes/
	LogFile          string // ~/.gopaw/logs/gopaw.log
	ConvLogFile      string // ~/.gopaw/logs/conversations.jsonl
	SkillsDir        string // ~/.gopaw/skills/
}

// Resolve expands ~ and returns a Paths struct with all derived paths.
func Resolve(dir string) (*Paths, error) {
	expanded := expandTilde(dir)
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return nil, err
	}
	memDir := filepath.Join(abs, "memory")
	return &Paths{
		Root:           abs,
		DBFile:         filepath.Join(abs, "gopaw.db"),
		MemoriesDBFile: filepath.Join(abs, "memories.db"),
		AgentMDFile:    filepath.Join(abs, "agent", "AGENT.md"),
		PersonaMDFile:  filepath.Join(abs, "agent", "PERSONA.md"),
		ContextMDFile:  filepath.Join(abs, "agent", "CONTEXT.md"),
		MemoryDir:      memDir,
		MemoryMDFile:   filepath.Join(memDir, "MEMORY.md"),
		MemoryArchDir:  filepath.Join(memDir, "archive"),
		MemorySnapDir:  filepath.Join(memDir, ".snapshot"),
		MemoryNotesDir: filepath.Join(memDir, "notes"),
		LogFile:        filepath.Join(abs, "logs", "gopaw.log"),
		ConvLogFile:    filepath.Join(abs, "logs", "conversations.jsonl"),
		SkillsDir:      filepath.Join(abs, "skills"),
	}, nil
}

// EnsureDirs creates all required subdirectories under the workspace.
func EnsureDirs(p *Paths) error {
	dirs := []string{
		p.Root,
		filepath.Dir(p.AgentMDFile), // ~/.gopaw/agent/
		p.MemoryDir,                 // ~/.gopaw/memory/
		p.MemoryArchDir,             // ~/.gopaw/memory/archive/
		p.MemorySnapDir,             // ~/.gopaw/memory/.snapshot/
		p.MemoryNotesDir,            // ~/.gopaw/memory/notes/
		filepath.Dir(p.LogFile),     // ~/.gopaw/logs/
		p.SkillsDir,                 // ~/.gopaw/skills/
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	// 初始化 MEMORY.md 模板（仅在文件不存在时创建）
	if _, err := os.Stat(p.MemoryMDFile); os.IsNotExist(err) {
		if err := initMemoryMD(p.MemoryMDFile); err != nil {
			return err
		}
	}
	return nil
}

// initMemoryMD 创建初始 MEMORY.md 模板文件。
func initMemoryMD(path string) error {
	content := "# Memory\n> 最后更新：" + time.Now().Format("2006-01-02") + "\n\n" +
		"## 核心事实\n" +
		"<!-- 在这里记录重要事实，例如：用户偏好、项目信息等 -->\n\n" +
		"## 近期摘要\n" +
		"<!-- 由系统自动追加的对话摘要 -->\n"
	return os.WriteFile(path, []byte(content), 0o644)
}

// expandTilde replaces ~/ with the user's home directory.
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
