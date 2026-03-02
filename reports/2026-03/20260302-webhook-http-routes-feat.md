# [feat] Webhook HTTP 路由接入

**报告日期**: 2026-03-02
**开发者**: 小M (AI 助手)
**关联任务**: M-1 / handoff-xiaom.md
**涉及文件数**: 3 个

---

## 功能概述

为 Webhook 频道注册 HTTP 路由，使外部系统可以：
1. POST `/webhook/:token` 推送消息给 Agent
2. GET `/webhook/:token/messages` 轮询拉取 Agent 回复

让 Webhook 频道从"代码已实现"变为"真正可用"。

---

## 实现说明

### 架构设计

```
外部系统 ──POST──► /webhook/:token ──► WebhookHandler.Receive() ──► Plugin.HandleReceive()
                                                                          │
                                                                          ▼
                                                                    inbound channel
                                                                          │
                                                                          ▼
                                                                       Agent
                                                                          │
                                                                          ▼
外部系统 ◄──GET─── /webhook/:token/messages ◄── WebhookHandler.Poll() ◄─ Plugin.HandlePoll()
```

### Step 1：Manager.GetPlugin 方法

```go
// GetPlugin returns the plugin with the given name.
// This is a wrapper around the registry's Get method for use by HTTP handlers.
func (m *Manager) GetPlugin(name string) (plugin.ChannelPlugin, error) {
    return m.registry.Get(name)
}
```

**审查要点**：
- ✅ 简单代理，内部调用 `m.registry.Get(name)`
- ✅ 返回类型是 `plugin.ChannelPlugin` 接口
- ⚠️ Handler 层需类型断言到具体 `*webhook.Plugin`

### Step 2：WebhookHandler

```go
// WebhookHandler handles Webhook channel HTTP requests.
type WebhookHandler struct {
    channelMgr *channel.Manager
}

// Receive handles POST /webhook/:token — external systems push messages to Agent.
func (h *WebhookHandler) Receive(c *gin.Context) {
    token := c.Param("token")
    p, err := h.channelMgr.GetPlugin("webhook")
    if err != nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhook channel not available"})
        return
    }
    wp, ok := p.(*webhook.Plugin)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid plugin type"})
        return
    }
    wp.HandleReceive(c.Writer, c.Request, token)
}

// Poll handles GET /webhook/:token/messages — external systems poll for Agent responses.
func (h *WebhookHandler) Poll(c *gin.Context) {
    token := c.Param("token")
    p, err := h.channelMgr.GetPlugin("webhook")
    if err != nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "webhook channel not available"})
        return
    }
    wp, ok := p.(*webhook.Plugin)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid plugin type"})
        return
    }
    wp.HandlePoll(c.Writer, c.Request, token)
}
```

**审查要点**：
- ✅ 复用 Plugin 已有的 `HandleReceive` 和 `HandlePoll` 方法
- ✅ 类型断言失败返回 500
- ✅ 插件不可用返回 503（Service Unavailable）

### Step 3：路由注册

```go
// Webhook channel routes (no /api prefix, as agreed with external systems).
webhookH := handlers.NewWebhookHandler(channelMgr)
s.engine.POST("/webhook/:token", webhookH.Receive)
s.engine.GET("/webhook/:token/messages", webhookH.Poll)
```

**审查要点**：
- ✅ 路由不带 `/api` 前缀（与外部系统约定）
- ✅ 注册在 SPA handler 之前，避免被 fallback 捕获

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/channel/manager.go` | 修改 | 添加 `GetPlugin()` 方法 |
| `internal/server/handlers/webhook.go` | 新增 | Webhook HTTP 处理器 |
| `internal/server/server.go` | 修改 | 注册 Webhook 路由 |

**变更统计**：新增 52 行 / 修改 5 行 / 删除 0 行

---

## 接口 / API 变更

| 接口 | 变更类型 | 是否兼容 | 说明 |
|------|---------|---------|------|
| `POST /webhook/:token` | 新增 | — | 外部系统推送消息 |
| `GET /webhook/:token/messages` | 新增 | — | 外部系统轮询回复 |
| `Manager.GetPlugin(name string)` | 新增 | — | HTTP 层获取插件实例 |

---

## 自检结果

```bash
go build ./...      ✅ 通过
go vet ./...        ✅ 通过
go test -race ./... ✅ 通过
```

---

## 验收标准完成情况

对照 `handoff-xiaom.md` M-1 验收标准：

- [x] `go build ./...` 编译通过
- [x] `POST /webhook/{token}` 正确路由到 `HandleReceive()`
- [x] `GET /webhook/{token}/messages` 正确路由到 `HandlePoll()`
- [x] Token 不匹配时返回 401（逻辑在 plugin 内，已有处理）

---

## 遗留事项

无。

---

## 审查清单

### 代码逻辑

- [ ] `GetPlugin()` 是否正确代理 registry.Get()
- [ ] 类型断言是否安全（ok pattern）
- [ ] 错误响应码是否合适（503 vs 500）

### 路由注册

- [ ] 路由路径是否符合约定（无 /api 前缀）
- [ ] 是否在 SPA handler 之前注册

### 功能验证

- [ ] 正确 token 可以推送消息
- [ ] 错误 token 返回 401
- [ ] 轮询可以获取已推送消息

### 测试覆盖

- [ ] 手动测试路由可达
- [ ] `go test -race` 无 data race
