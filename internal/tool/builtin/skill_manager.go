package builtin

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/gopaw/gopaw/internal/skill"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

var (
	skillMgrMu sync.RWMutex
	skillMgr   *skill.Manager
)

// SetSkillManager 注入技能管理器，在 main.go 中初始化后调用。
func SetSkillManager(m *skill.Manager) {
	skillMgrMu.Lock()
	defer skillMgrMu.Unlock()
	skillMgr = m
}

func getSkillManager() *skill.Manager {
	skillMgrMu.RLock()
	defer skillMgrMu.RUnlock()
	return skillMgr
}

func init() {
	tool.Register(&SkillManagerTool{})
}

// SkillManagerTool 允许 Agent 在对话中查看和重新加载技能。
type SkillManagerTool struct{}

func (t *SkillManagerTool) Name() string { return "skill_manager" }

func (t *SkillManagerTool) Description() string {
	return "Manage GoPaw skills at runtime. " +
		"Use 'list' to show all loaded skills and their status. " +
		"Use 'reload' to re-scan the skills directory and load any newly added skills without restarting."
}

func (t *SkillManagerTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"action": {
				Type:        "string",
				Description: "Action to perform: 'list' to show all skills, 'reload' to rescan and reload skills from disk.",
				Enum:        []string{"list", "reload"},
			},
		},
		Required: []string{"action"},
	}
}

func (t *SkillManagerTool) Execute(_ context.Context, args map[string]interface{}) *plugin.ToolResult {
	mgr := getSkillManager()
	if mgr == nil {
		return plugin.ErrorResult("skill manager not available")
	}

	action, _ := args["action"].(string)

	switch action {
	case "list":
		return t.list(mgr)
	case "reload":
		return t.reload(mgr)
	default:
		return plugin.ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}

func (t *SkillManagerTool) list(mgr *skill.Manager) *plugin.ToolResult {
	entries := mgr.Registry().All()
	if len(entries) == 0 {
		return plugin.NewToolResult("当前没有已加载的技能。技能目录为工作区下的 skills/ 文件夹。")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("已加载 %d 个技能：\n\n", len(entries)))
	for _, e := range entries {
		status := "✓ 已启用"
		if !e.Enabled {
			status = "✗ 已禁用"
		}
		sb.WriteString(fmt.Sprintf("- **%s** (`%s`) %s\n",
			e.Manifest.DisplayName, e.Manifest.Name, status))
		if e.Manifest.Description != "" {
			sb.WriteString(fmt.Sprintf("  %s\n", e.Manifest.Description))
		}
	}
	return plugin.NewToolResult(sb.String())
}

func (t *SkillManagerTool) reload(mgr *skill.Manager) *plugin.ToolResult {
	if err := mgr.Reload(); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("技能重新加载失败：%v", err))
	}
	entries := mgr.Registry().All()
	return plugin.NewToolResult(fmt.Sprintf("技能已重新加载，当前共 %d 个技能。", len(entries)))
}
