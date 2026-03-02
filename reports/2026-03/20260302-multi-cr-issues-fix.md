# [fix] Code Review 问题整改

**报告日期**: 2026-03-02
**开发者**: 小M (AI Assistant)
**关联任务**: 审核报告 20260302-review-feat-multi-channel-test-connection 等
**涉及文件数**: 5 个

---

## 问题描述

根据三份 Code Review 报告，存在以下需要整改的问题：

### P0 问题（不得合并）

| # | 问题 | 来源 |
|---|------|------|
| P0-1 | 飞书插件测试未迁移，旧测试引用已删除的 `VerificationToken` 和 `HandleEventRequest` | 飞书 Stream Mode |
| P0-2 | 会话初始化竞态：`loadSessions()` 和 `createNewSession()` 顺序问题 | Web CR 整改 |
| P0-3 | EventSource 生命周期未管理，切换会话时未关闭 SSE 连接 | Web CR 整改 |

### P1 问题（强烈建议）

| # | 问题 | 来源 |
|---|------|------|
| P1-1 | `/api/channels/:name/test` 返回 `details` 直接透传内部错误 | 频道测试 |
| P1-3 | 多插件 HTTP 超时策略不一致，部分使用 `http.DefaultClient` | 频道测试 |
| P1-4 | 飞书 `connected` 在连接建立前置 true，健康状态误报 | 飞书 Stream Mode |

---

## 实现说明

### P0-1: 飞书插件测试重写

移除旧 Webhook 测试，改为 Stream Mode 适配：

```go
// 新测试覆盖 Stream Mode 场景
func TestPlugin_Test(t *testing.T) {
    // Test 1: 未配置时测试失败
    p := &Plugin{}
    p.Init([]byte("{}"))
    result := p.Test(context.Background())
    if result.Success {
        t.Fatal("expected test to fail for unconfigured plugin")
    }

    // Test 2: 配置有效时测试 token 获取
    p2 := &Plugin{}
    p2.Init(json.RawMessage(`{"app_id":"test","app_secret":"test"}`))
    // ... token 获取测试
}
```

**审查要点**：
- ✅ 移除了对 `VerificationToken` 和 `HandleEventRequest` 的引用
- ✅ 新增 `TestPlugin_Test` 覆盖 Test() 方法
- ✅ 新增 `TestPlugin_Stop` 覆盖生命周期
- ✅ 新增 `TestPlugin_ConcurrentHealth` 覆盖并发安全
- ✅ 新增 `TestPlugin_TokenExpiry` 覆盖 token 过期逻辑

### P0-2 & P0-3: 前端会话和 SSE 生命周期

```typescript
// 维护 EventSource 引用
let currentEventSource: EventSource | null = null

function closeEventSource() {
  if (currentEventSource) {
    currentEventSource.close()
    currentEventSource = null
  }
  isStreaming.value = false
  isThinking.value = false
}

onMounted(async () => {
  // 先加载会话列表，等待完成
  await loadSessions()
  
  // 如果有历史会话，选择第一个；否则创建新会话
  if (sessions.value.length > 0) {
    selectSession(sessions.value[0].id)
  } else {
    createNewSession()
  }
})

onUnmounted(() => {
  // 组件卸载时关闭 EventSource
  closeEventSource()
})
```

**审查要点**：
- ✅ `onMounted` 改为 `async`，先 `await loadSessions()` 再判断
- ✅ 维护 `currentEventSource` 引用
- ✅ `selectSession` 和 `createNewSession` 开始时调用 `closeEventSource()`
- ✅ 新增 `onUnmounted` 清理 EventSource

### P1-1: 错误脱敏

```go
func (h *ChannelsHandler) Test(c *gin.Context) {
    // ...
    
    // Details 包含敏感错误信息，只写日志不返回前端
    if result.Details != "" {
        h.logger.Warn("channel test failed",
            zap.String("channel", name),
            zap.String("message", result.Message),
            zap.String("details", result.Details),
        )
    }

    c.JSON(http.StatusOK, gin.H{
        "success": result.Success,
        "message": result.Message,
        // 不返回 details
    })
}
```

**审查要点**：
- ✅ 移除了 `details` 字段的前端返回
- ✅ 敏感信息仅写服务端日志

### P1-3: 统一 HTTP 超时策略

```go
// dingtalk/plugin.go
const (
    defaultTimeout = 10 * time.Second
)

type Plugin struct {
    // ...
    httpClient *http.Client
}

func (p *Plugin) Init(cfg json.RawMessage) error {
    // ...
    p.httpClient = &http.Client{Timeout: defaultTimeout}
    // ...
}
```

**审查要点**：
- ✅ 钉钉插件统一使用带 10s 超时的 `httpClient`
- ✅ 与飞书插件保持一致的超时策略

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `plugins/channels/feishu/plugin_test.go` | 重写 | 适配 Stream Mode，9 个测试用例 |
| `plugins/channels/feishu/plugin.go` | 修改 | 优化日志和连接状态 |
| `plugins/channels/dingtalk/plugin.go` | 修改 | 统一 httpClient + 超时策略 |
| `internal/server/handlers/channels.go` | 修改 | details 脱敏，只写日志 |
| `web/src/pages/Chat.vue` | 修改 | 修复会话竞态 + EventSource 生命周期 |

**变更统计**：约 +120/-60 行

---

## 自检结果

```bash
# 后端编译
go build ./plugins/channels/... ./internal/server/...  ✅ 通过

# 飞书插件测试（含 race 检测）
go test -race ./plugins/channels/feishu/...  ✅ 通过（9 个用例）

# 前端编译
npx vite build  ✅ 通过
```

---

## 遗留事项

无

---

## 审查清单

### 代码逻辑
- [ ] 飞书测试是否完整覆盖 Stream Mode 场景
- [ ] EventSource 在所有退出路径是否正确关闭
- [ ] 会话初始化竞态是否彻底解决

### 并发 & 错误处理
- [ ] 钉钉 httpClient 是否统一使用
- [ ] details 敏感信息是否完全隔离

### 功能验证
- [ ] 切换会话时 SSE 是否正确断开
- [ ] 页面卸载时资源是否正确释放

### 测试覆盖
- [ ] 飞书插件测试是否通过 race 检测
- [ ] 测试用例是否覆盖主要场景
