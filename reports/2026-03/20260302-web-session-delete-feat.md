# [feat] 会话删除功能实现报告

**报告日期**: 2026-03-02
**开发者**: 小G (AI Assistant)
**关联任务**: N-8
**涉及文件数**: 3 个

---

## 功能概述

为用户提供物理删除历史会话的能力。实现了从前端入口、二次确认风险提示、后端 API 调用到数据库级联清理的全链路闭环。

---

## 实现说明

### 核心逻辑

1. **后端 API**: 在 `internal/server/handlers/agent.go` 中新增 `DeleteSession` 方法，调用 `memory.Manager.Clear` 清理 SQLite 中的消息与会话记录，并同步移除内存中的会话对象。
2. **前端交互**: 
   - 在会话列表项采用悬浮（Hover）触发删除按钮的设计，保持界面简洁。
   - 集成 `naive-ui` 的 `useDialog` 进行风险确认。
   - 删除当前正在对话的会话时，自动调用 `closeCurrentSSE()` 并重置 UI。

```go
// 后端删除实现片段
func (h *AgentHandler) DeleteSession(c *gin.Context) {
    sessionID := c.Param("id")
    h.mem.Clear(sessionID)
    h.agent.Sessions().Delete(sessionID)
    c.JSON(http.StatusOK, gin.H{"ok": true, "deleted_id": sessionID})
}
```

**审查要点**：
- ✅ 确保了内存与持久化存储的一致性清理。
- ✅ 实现了 SSE 连接的主动切断，防止残留回调干扰。

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/server/handlers/agent.go` | 修改 | 新增 DeleteSession 处理器 |
| `internal/server/server.go` | 修改 | 注册 DELETE 路由 |
| `web/src/pages/Chat.vue` | 修改 | 增加删除按钮及交互逻辑 |

---

## 验收标准完成情况

- [x] 后端 API 响应正确。
- [x] 前端列表实时刷新。
- [x] 删除当前会话逻辑严密（含 SSE 关闭）。

---

## 自检结果

```bash
go build ./...      ✅ 通过
pnpm run build      ✅ 通过
```
