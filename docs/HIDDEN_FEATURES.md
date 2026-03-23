# 隐藏 / 已移除功能记录

> 最后更新：2026-03-23

本文档记录从导航菜单中移除或代码已删除的功能模块，便于后续维护时快速了解背景。

---

## 已彻底删除（代码不存在）

### 触发器（Triggers）

- **删除时间**：2026-03 本轮迭代
- **原因**：与"定时任务"模块功能高度重叠（cron 类型触发器），webhook/message 类型使用率极低
- **删除范围**：
  - `internal/trigger/` — 整个目录已删除（Manager、Engine、types）
  - `internal/server/handlers/triggers.go` — Handler 已删除
  - `internal/server/server.go` — 触发器路由已移除（`/webhook/:id`、`/api/messages`）
  - `cmd/gopaw/main.go` — `triggerMgr`、`triggerEngine` 初始化已移除
  - `web/src/pages/Triggers.vue` — 页面已删除
  - `web/src/router/index.ts` — `/triggers` 路由已移除

---

## 已从导航隐藏（代码保留）

### Agent 消息（Agent Messages）

- **隐藏时间**：2026-03 本轮迭代
- **原因**：Agent 间通信系统（task/response/notify/query），主要作为内部基础设施使用，用户直接使用率极低；依赖触发器的 message 类型触发器已随触发器一起删除
- **现状**：后端路由、数据库表、消息管理器均保留，作为多 Agent 协作的底层通信机制
- **恢复方式**：在 `web/src/layouts/IconSidebar.vue` 的 `automationItems` 中添加 `{ to: '/agent-messages', label: 'Agent消息', icon: MessageCircle }`，并在 router/index.ts 中添加对应路由

### 工作流（Workflows）

- **隐藏时间**：2026-03 本轮迭代
- **原因**：功能不完善——执行引擎是 fire-and-forget 模式，步骤间输出传递实际不可用；与"流程编排"功能定位重叠
- **现状**：后端代码（`internal/workflow/`）完整保留，前端页面（`web/src/pages/Workflows.vue`）保留，路由保留
- **核心缺陷**：`runner.go` 的 `executeTask/executeNotify/executeQuery` 均通过 `msgMgr.SendTask()` fire-and-forget 发出消息后即返回，不等待 Agent 回复，导致步骤间 `{{steps.step1.output}}` 变量传递形同虚设
- **恢复方式**：在 `web/src/layouts/IconSidebar.vue` 的 `automationItems` 中添加 `{ to: '/workflows', label: '工作流', icon: GitBranch }`

---

## 菜单名称变更记录

| 原名 | 现名 | 变更时间 |
|------|------|---------|
| 编排器 | 流程编排 | 2026-03 |

---

## 说明

- **隐藏 ≠ 删除**：隐藏的功能后端 API 仍然可访问，只是没有导航入口
- 如需恢复某项功能，参考"恢复方式"说明，通常只需改侧边栏 + router
