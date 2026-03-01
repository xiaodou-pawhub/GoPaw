# Webhook 频道插件设计文档

## 职责

Webhook 频道是 GoPaw 对外开放的通用接入桥梁：
- 接收任意第三方系统发来的 POST 消息
- 支持两种响应模式：主动推送（callback_url）和外部轮询（GET /messages）
- token 用于鉴权，防止未授权调用

## 架构图

```mermaid
flowchart LR
    EXT[外部系统] -->|POST /webhook/{token}| RCV[HandleReceive]
    RCV --> IB[(inbound channel)]
    IB --> AG[Agent]
    AG --> PL[Plugin.Send]

    PL -->|callback_url 已配置| CB[POST callback_url]
    PL -->|轮询模式| QUEUE[(outbound map)]
    EXT -->|GET /webhook/{token}/messages| POLL[HandlePoll]
    POLL --> QUEUE
```

## 核心接口

```go
func (p *Plugin) HandleReceive(w, r, token string)  // POST /webhook/{token}
func (p *Plugin) HandlePoll(w, r, token string)      // GET /webhook/{token}/messages
```

## 关键设计决策

1. **轮询 vs 回调**：通过 `callback_url` 是否配置自动切换模式，无需代码改动。
2. **内存队列**：轮询模式的响应存储在内存中，读取后清空，适合低频场景；高频场景推荐 callback_url。
3. **token 鉴权**：所有请求都验证 token，防止滥用。

## 依赖关系

- **依赖**：`internal/channel`（注册表）、`pkg/types`
- **被依赖**：`internal/server`（注册 HTTP 路由 `/webhook/:token`）

## 验收标准

- [ ] POST /webhook/{token} 能将消息传递给 Agent
- [ ] callback_url 配置时 Agent 响应被 POST 到外部 URL
- [ ] callback_url 未配置时响应通过 GET /webhook/{token}/messages 可以轮询到
- [ ] token 错误返回 401

## 配置项

```yaml
plugin:
  webhook:
    token: ${WEBHOOK_TOKEN}
    callback_url: ""   # 留空则外部轮询
```
