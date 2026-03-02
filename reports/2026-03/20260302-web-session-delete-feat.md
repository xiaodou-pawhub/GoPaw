# [feat] 会话删除功能实现报告

**报告日期**: 2026-03-02
**开发者**: 小G (AI Assistant)
**关联任务**: N-8
**涉及文件数**: 3 个

---

## 功能概述

为用户提供删除历史会话的能力。实现了从前端入口、二次确认、后端 API 调用到数据库级联删除的全链路功能。

---

## 实现说明

### 核心逻辑

1. **后端 API**: 在 `AgentHandler` 中新增 `DeleteSession` 方法，调用 `memory.Manager.Clear` 清理 SQLite 数据，并同步更新内存中的 `SessionManager`。
2. **前端交互**: 
   - 采用列表项悬浮（Hover）显示删除图标的设计，减少视觉干扰。
   - 集成 `naive-ui` 的 `useDialog` 进行风险提示。
   - 删除当前正在对话的会话时，自动重置聊天区域。

```go
// 后端删除核心实现
func (h *AgentHandler) DeleteSession(c *gin.Context) {
    sessionID := c.Param("id")
    h.mem.Clear(sessionID) // 清理数据库
    h.agent.Sessions().Delete(sessionID) // 清理内存
    c.JSON(http.StatusOK, gin.H{"ok": true})
}
```

**审查要点**：
- ✅ 实现了内存与数据库的双重清理。
- ✅ 前端增加了防误点二次确认逻辑。

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/server/handlers/agent.go` | 修改 | 新增 DeleteSession 处理器 |
| `internal/server/server.go` | 修改 | 注册 DELETE 路由 |
| `web/src/pages/Chat.vue` | 修改 | 增加删除按钮及交互逻辑 |

---

## 验收标准完成情况

- [x] 后端 API 支持 DELETE 方法。
- [x] 前端列表实时反映删除结果。
- [x] 删除当前会话不导致界面崩溃。

---

## 自检结果

```bash
go build ./...      ✅ 通过
pnpm run build      ✅ 通过
```
