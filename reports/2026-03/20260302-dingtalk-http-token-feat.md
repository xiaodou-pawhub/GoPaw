# [feat] 钉钉 HTTP 路由 + Token 自动刷新

**报告日期**: 2026-03-02
**开发者**: 小M (AI 助手)
**关联任务**: M-2 / handoff-xiaom.md
**涉及文件数**: 3 个

---

## 功能概述

1. 注册钉钉事件接收路由 `POST /dingtalk/event`
2. 实现后台 Token 自动刷新（每 90 分钟），避免长时间运行后 401 错误
3. 实现队列满时返回 503（与 Webhook 一致）

---

## 实现说明

### 架构设计

```
钉钉服务器 ──POST──► /dingtalk/event ──► DingTalkHandler.Event()
                                                │
                                                ▼
                                        Plugin.HandleReceive()
                                                │
                                                ▼
                                        HMAC 签名验证 → inbound channel
                                                │
                                                ▼
                                            Agent
                                                │
                                                ▼
                                        Plugin.Send() → getToken()
                                                │
                                                ▼
                                        钉钉 API (带 Token)
```

### Step 1：Token 自动刷新

```go
type Plugin struct {
    // Token 缓存与并发保护
    tokenMu     sync.RWMutex
    cachedToken string
    tokenExpiry time.Time
    cancel      context.CancelFunc
}

// Start 启动时获取初始 Token 并启动刷新循环
func (p *Plugin) Start(ctx context.Context) error {
    ctx, p.cancel = context.WithCancel(ctx)
    if p.configured {
        if err := p.refreshAndCache(ctx); err != nil {
            p.logger.Warn("dingtalk: initial token fetch failed", zap.Error(err))
        }
        go p.tokenRefreshLoop(ctx)
    }
    return nil
}

// tokenRefreshLoop 每 90 分钟刷新一次
func (p *Plugin) tokenRefreshLoop(ctx context.Context) {
    ticker := time.NewTicker(90 * time.Minute)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := p.refreshAndCache(ctx); err != nil {
                p.logger.Error("dingtalk: token refresh failed", zap.Error(err))
            }
        }
    }
}

// getToken 获取缓存的 Token，过期时自动刷新
func (p *Plugin) getToken() (string, error) {
    p.tokenMu.RLock()
    if p.cachedToken != "" && time.Now().Before(p.tokenExpiry) {
        token := p.cachedToken
        p.tokenMu.RUnlock()
        return token, nil
    }
    p.tokenMu.RUnlock()

    // Token 过期，需要刷新
    if err := p.refreshAndCache(context.Background()); err != nil {
        return "", fmt.Errorf("refresh token: %w", err)
    }
    // ...
}
```

**审查要点**：
- ✅ 使用 RWMutex 保护 Token 缓存
- ✅ 提前 5 分钟过期，避免临界时刻失效
- ✅ context 控制 goroutine 生命周期
- ⚠️ `refreshAndCache` 在 Send 中调用时使用 `context.Background()`

### Step 2：HTTP 路由处理器

```go
// DingTalkHandler handles DingTalk channel HTTP requests.
type DingTalkHandler struct {
    channelMgr *channel.Manager
}

func (h *DingTalkHandler) Event(c *gin.Context) {
    p, err := h.channelMgr.GetActivePlugin("dingtalk")
    if err != nil {
        c.JSON(http.StatusServiceUnavailable, gin.H{"error": "dingtalk channel not available"})
        return
    }
    handler, ok := p.(HTTPHandler)
    if !ok {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "plugin does not support HTTP handling"})
        return
    }
    handler.HandleReceive(c.Writer, c.Request, "")
}
```

**审查要点**：
- ✅ 使用 `GetActivePlugin` 确保插件已启动
- ✅ 使用 `HTTPHandler` 接口解耦
- ✅ 复用 M-1 中定义的接口

### Step 3：HandleReceive 实现

```go
func (p *Plugin) HandleReceive(w http.ResponseWriter, r *http.Request, _ string) {
    // ... 签名验证、事件解析 ...

    select {
    case p.inbound <- msg:
        w.WriteHeader(http.StatusOK)
    default:
        // 队列满，返回 503
        p.logger.Warn("dingtalk: inbound queue full, rejecting message")
        w.WriteHeader(http.StatusServiceUnavailable)
    }
}
```

**审查要点**：
- ✅ 队列满时返回 503（与 Webhook 一致）
- ✅ HMAC-SHA256 签名验证保持不变

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `plugins/channels/dingtalk/plugin.go` | 修改 | Token 自动刷新 + HandleReceive 实现 |
| `internal/server/handlers/dingtalk.go` | 新增 | 钉钉 HTTP 处理器 |
| `internal/server/server.go` | 修改 | 注册路由 |

**变更统计**：新增 120 行 / 修改 30 行 / 删除 40 行

---

## 接口 / API 变更

| 接口 | 变更类型 | 是否兼容 | 说明 |
|------|---------|---------|------|
| `POST /dingtalk/event` | 新增 | — | 钉钉事件接收 |
| `Plugin.getToken()` | 新增 | — | 内部方法 |

---

## 自检结果

```bash
go build ./...      ✅ 通过
go vet ./...        ✅ 通过
go test -race ./... ✅ 通过
```

---

## 验收标准完成情况

对照 `handoff-xiaom.md` M-2 验收标准：

- [x] `go build ./...` 编译通过
- [x] `POST /dingtalk/event` 路由可访问
- [x] HMAC 签名验证正确（签名错误返回 401）
- [x] `go test -race` 无 data race

---

## 遗留事项

无。

---

## 审查清单

### 代码逻辑

- [ ] Token 缓存并发安全（RWMutex）
- [ ] goroutine 有退出机制（context.Done）
- [ ] 提前 5 分钟刷新避免临界失效

### 功能验证

- [ ] 正确签名可接收事件
- [ ] 错误签名返回 401
- [ ] Token 过期后自动刷新

### 测试覆盖

- [ ] `go test -race` 无 data race
