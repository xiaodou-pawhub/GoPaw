package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&FileManageTool{})
}

type FileManageTool struct{}

func (t *FileManageTool) Name() string { return "file_manage" }

func (t *FileManageTool) Description() string {
	return "Manage files and directories in the workspace. " +
		"Actions: 'list' (list directory), 'move' (rename/move), 'delete' (remove file/dir), " +
		"'stat' (get file info), 'mkdir' (create directory). " +
		"All operations are sandboxed to the workspace. Use '.' or '~' for root. " +
		"Destructive actions (move, delete) require user confirmation."
}

func (t *FileManageTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"action": {
				Type:        "string",
				Description: "The action to perform.",
				Enum:        []string{"list", "move", "delete", "stat", "mkdir"},
			},
			"path": {
				Type:        "string",
				Description: "The path to the file or directory. Can be relative to workspace or starting with '~'.",
			},
			"new_path": {
				Type:        "string",
				Description: "Target destination path (only for 'move').",
			},
			"recursive": {
				Type:        "boolean",
				Description: "Enable recursive deletion for non-empty directories.",
			},
		},
		Required: []string{"action", "path"},
	}
}

// GuardedTool implementation: Force approval for destructive operations with Rich Metadata.
func (t *FileManageTool) RequireApproval(args map[string]interface{}) bool {
	action, _ := args["action"].(string)
	if action == "move" || action == "delete" {
		return true
	}
	return false
}

func (t *FileManageTool) Execute(_ context.Context, args map[string]interface{}) *plugin.ToolResult {
	action, _ := args["action"].(string)
	relPath, _ := args["path"].(string)
	newRelPath, _ := args["new_path"].(string)
	recursive, _ := args["recursive"].(bool)

	root := getWorkspaceRoot()
	if root == "" {
		return plugin.ErrorResult("workspace root not configured")
	}

	// 1. Resolve Path with Smart Routing (~ or .)
	absPath, err := t.resolveSafePath(root, relPath)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("security block: %v", err))
	}

	switch action {
	case "list":
		return t.handleList(absPath, relPath)

	case "stat":
		return t.handleStat(absPath, relPath)

	case "mkdir":
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return plugin.ErrorResult(fmt.Sprintf("failed to create directory: %v", err))
		}
		return plugin.NewToolResult(fmt.Sprintf("Directory created: %s", relPath))

	case "move":
		if newRelPath == "" {
			return plugin.ErrorResult("new_path is required for move")
		}
		absNewPath, err := t.resolveSafePath(root, newRelPath)
		if err != nil {
			return plugin.ErrorResult(fmt.Sprintf("security block (new_path): %v", err))
		}
		
		// Ensure destination parent exists (Go feature: auto-mkdir for moves)
		if err := os.MkdirAll(filepath.Dir(absNewPath), 0755); err != nil {
			return plugin.ErrorResult(fmt.Sprintf("failed to prepare destination: %v", err))
		}

		if err := os.Rename(absPath, absNewPath); err != nil {
			return plugin.ErrorResult(fmt.Sprintf("failed to move: %v", err))
		}
		return plugin.NewToolResult(fmt.Sprintf("SUCCESS: Moved %s to %s", relPath, newRelPath))

	case "delete":
		// Safety check: Don't allow deleting the root!
		if absPath == root {
			return plugin.ErrorResult("critical: deletion of workspace root is prohibited")
		}

		var err error
		if recursive {
			err = os.RemoveAll(absPath)
		} else {
			err = os.Remove(absPath)
		}
		
		if err != nil {
			return plugin.ErrorResult(fmt.Sprintf("failed to delete: %v (hint: use recursive=true for non-empty dirs)", err))
		}
		return plugin.NewToolResult(fmt.Sprintf("SUCCESS: Deleted %s", relPath))

	default:
		return plugin.ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}

// ── Action Handlers ────────────────────────────────────────────────────────

func (t *FileManageTool) handleList(absPath, relPath string) *plugin.ToolResult {
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to read directory: %v", err))
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "### Contents of %s\n\n", relPath)
	fmt.Fprintf(&sb, "| Type | Name | Size | Modified |\n")
	fmt.Fprintf(&sb, "| :--- | :--- | :--- | :--- |\n")

	for _, e := range entries {
		info, _ := e.Info()
		typeIcon := "📄"
		if e.IsDir() {
			typeIcon = "📁"
		}
		fmt.Fprintf(&sb, "| %s | %s | %s | %s |\n", 
			typeIcon, e.Name(), formatSize(info.Size()), info.ModTime().Format("2006-01-02 15:04"))
	}
	
	if len(entries) == 0 {
		sb.WriteString("\n*(Directory is empty)*")
	}

	return plugin.NewToolResult(sb.String())
}

func (t *FileManageTool) handleStat(absPath, relPath string) *plugin.ToolResult {
	info, err := os.Stat(absPath)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to access: %v", err))
	}

	res := fmt.Sprintf("### File Metadata: %s\n\n", relPath)
	res += fmt.Sprintf("- **Type**: %s\n", map[bool]string{true: "Directory", false: "File"}[info.IsDir()])
	res += fmt.Sprintf("- **Size**: %s (%d bytes)\n", formatSize(info.Size()), info.Size())
	res += fmt.Sprintf("- **Permissions**: %v\n", info.Mode())
	res += fmt.Sprintf("- **Last Modified**: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))

	return plugin.NewToolResult(res)
}

// ── Helpers ────────────────────────────────────────────────────────────────

func (t *FileManageTool) resolveSafePath(root, rel string) (string, error) {
	// Handle special prefixes
	if strings.HasPrefix(rel, "~/") {
		rel = strings.TrimPrefix(rel, "~/")
	} else if rel == "~" || rel == "" {
		rel = "."
	}

	// Clean path and join with root
	abs := filepath.Join(root, filepath.Clean(rel))
	
	// Absolute path validation (Strict Sandbox)
	absRoot, _ := filepath.Abs(root)
	absTarget, _ := filepath.Abs(abs)

	if !strings.HasPrefix(absTarget, absRoot) {
		return "", fmt.Errorf("access denied: path %s attempts to escape workspace", rel)
	}
	
	return absTarget, nil
}

func formatSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
