# [feat] 飞书通信模式重构 (Webhook -> Stream Mode)

**报告日期**: 2026-03-02
**开发者**: 小M (AI Assistant)
**关联任务**: 飞书长连接支持
**涉及文件数**: 1 个 (plugins/channels/feishu/plugin.go)

---

## 功能概述

将飞书频道的通信模式从传统的 **Webhook (HTTP 回调)** 彻底切换为 **Stream Mode (WebSocket 长连接)**。

**核心改进：**
- **零配置穿透**：不再需要公网 IP 或内网穿透工具，机器人即可在内网接收飞书消息。
- **安全性提升**：通信由 GoPaw 主动发起，不再暴露 HTTP 端点，减少了潜在的攻击面。
- **架构简化**：移除了复杂的验签、加解密和 Challenge 响应逻辑。

---

## 实现说明

### 核心逻辑

引入飞书官方 `oapi-sdk-go/v3`，利用 `larkws` 建立持久连接。

1. **客户端初始化**：使用 `app_id` 和 `app_secret` 初始化 `larkws.Client`。
2. **事件订阅**：通过 `larkevent.NewEventDispatcher` 监听 `P2MessageReceiveV1` 事件。
3. **异步连接**：在 `Start()` 生命周期中开启独立 goroutine 维护 WebSocket 连接。
4. **统一分发**：将收到的消息解包后，通过 `inbound` channel 注入 GoPaw 核心引擎。

```go
// 启动长连接核心代码
p.wsClient = larkws.NewClient(p.cfg.AppID, p.cfg.AppSecret,
    larkws.WithEventHandler(eventHandler),
)
go func() {
    err := p.wsClient.Start(context.Background())
    // ... 错误处理
}()
```

**审查要点**：
- ✅ 长连接在后台 goroutine 运行，不阻塞主进程启动。
- ✅ 消息发送 (`Send`) 仍保留 REST 模式，符合飞书官方架构设计。
- ⚠️ 目前 SDK 内部自动处理重连，但需观察极度不稳定的网络环境下的心跳表现。

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `plugins/channels/feishu/plugin.go` | 修改 | 全面重写接收逻辑，集成 larkws |

---

## 验收标准完成情况

- [x] 成功集成飞书官方长连接 SDK。
- [x] 实现 WebSocket 消息接收并转换为 GoPaw 消息模型。
- [x] 保留 OpenAPI 消息发送能力。
- [x] 移除不再需要的 Webhook 验签逻辑。

---

## 自检结果

```bash
go build ./...      ✅ 通过
go vet ./...        ✅ 通过
```

---

## 遗留事项

- 需在 Web UI 增加飞书长连接的实时连接状态（当前仅显示“长连接运行中”）。
- 后续可考虑增加 WebSocket 连接断开后的自动重试日志报警。

---

## 审查清单

### 代码逻辑
- [ ] 后台 goroutine 的 Context 生命周期管理。
- [ ] SDK 引入后二进制体积的增量评估（预计增加 5-10MB）。

### 功能验证
- [ ] 在无公网环境下的实际收发测试。
- [ ] 飞书后台长连接开关状态的确认。
