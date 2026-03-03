// Package workspace resolves and provides all runtime data paths under the workspace directory.
package workspace

import (
	"os"
	"path/filepath"
	"strings"
)

// Paths holds resolved absolute paths for all workspace data.
type Paths struct {
	Root          string // workspace root, e.g. ~/.gopaw
	DBFile        string // ~/.gopaw/gopaw.db
	AgentMDFile   string // ~/.gopaw/agent/AGENT.md
	PersonaMDFile string // ~/.gopaw/agent/PERSONA.md
	ContextMDFile string // ~/.gopaw/agent/CONTEXT.md
	MemoryMDFile  string // ~/.gopaw/agent/MEMORY.md
	LogFile       string // ~/.gopaw/logs/gopaw.log
	ConvLogFile   string // ~/.gopaw/logs/conversations.jsonl
	SkillsDir     string // ~/.gopaw/skills/
}

// Resolve expands ~ and returns a Paths struct with all derived paths.
func Resolve(dir string) (*Paths, error) {
	expanded := expandTilde(dir)
	abs, err := filepath.Abs(expanded)
	if err != nil {
		return nil, err
	}
	return &Paths{
		Root:          abs,
		DBFile:        filepath.Join(abs, "gopaw.db"),
		AgentMDFile:   filepath.Join(abs, "agent", "AGENT.md"),
		PersonaMDFile: filepath.Join(abs, "agent", "PERSONA.md"),
		ContextMDFile: filepath.Join(abs, "agent", "CONTEXT.md"),
		MemoryMDFile:  filepath.Join(abs, "agent", "MEMORY.md"),
		LogFile:       filepath.Join(abs, "logs", "gopaw.log"),
		ConvLogFile:   filepath.Join(abs, "logs", "conversations.jsonl"),
		SkillsDir:     filepath.Join(abs, "skills"),
	}, nil
}

// EnsureDirs creates all required subdirectories under the workspace.
func EnsureDirs(p *Paths) error {
	dirs := []string{
		p.Root,
		filepath.Dir(p.AgentMDFile), // ~/.gopaw/agent/
		filepath.Dir(p.LogFile),     // ~/.gopaw/logs/
		p.SkillsDir,                 // ~/.gopaw/skills/
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
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
