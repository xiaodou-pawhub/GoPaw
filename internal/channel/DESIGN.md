# Channel 模块设计文档

## 职责

Channel 模块负责：
- 维护频道插件注册表（Registry）
- 启动所有已启用的频道插件，统一收集消息
- 将来自不同频道的消息汇聚到单一 channel，供 Agent 消费
- 将 Agent 的响应路由到对应频道的插件进行发送
- 提供健康状态聚合接口

Channel 模块**不负责**：
- 消息的业务处理（由 Agent 负责）
- 插件本身的协议实现（由各 plugins/channels/* 负责）

## 架构图

```mermaid
flowchart TB
    subgraph 频道插件
        FS[feishu plugin]
        DT[dingtalk plugin]
        WH[webhook plugin]
        CN[console plugin]
    end

    subgraph internal/channel
        REG[Registry]
        MGR[Manager]
        AGG[(aggregated chan\ncap=1000)]
    end

    FS -->|Receive()| MGR
    DT -->|Receive()| MGR
    WH -->|Receive()| MGR
    CN -->|Receive()| MGR

    MGR -->|fanIn goroutine| AGG
    AGG -->|Messages()| Agent

    Agent -->|Send(msg)| MGR
    MGR -->|plugin.Send()| FS
    MGR -->|plugin.Send()| DT
```

## 核心接口

```go
type Manager struct { ... }

func NewManager(registry *Registry, logger *zap.Logger) *Manager
func (m *Manager) Start(ctx context.Context, enabledNames []string, pluginCfgs map[string]json.RawMessage) error
func (m *Manager) Messages() <-chan *types.Message
func (m *Manager) Send(msg *types.Message) error
func (m *Manager) Stop()
func (m *Manager) Health() map[string]plugin.HealthStatus

type Registry struct { ... }
func Register(p plugin.ChannelPlugin)  // 全局注册（供 init()）
func Global() *Registry
```

## 关键设计决策

1. **缓冲聚合 channel（容量 1000）**：防止慢速 Agent 导致高流量频道积压。超过缓冲上限时丢弃消息并记录警告，保持系统稳定。
2. **fanIn 模式**：每个插件对应一个 goroutine，将其 Receive() channel 转发到 aggregated，避免 Agent 轮询多个 channel。
3. **Init → Start 两阶段启动**：Init 解析配置（可以快速失败），Start 建立网络连接（异步），与其他模块的生命周期解耦。

## 依赖关系

- **依赖**：`pkg/plugin`（ChannelPlugin 接口）、`pkg/types`（Message 类型）、`go.uber.org/zap`
- **被依赖**：`internal/agent`（消息来源和发送目标）、`internal/server/handlers`（channels API）

## 验收标准

- [ ] console 频道启动后能收发消息
- [ ] 未注册的插件名在 Start() 时记录警告，不导致整体启动失败
- [ ] 所有启用的频道消息都能通过 Messages() 收到
- [ ] Send() 能正确路由到对应频道插件
- [ ] Stop() 正常关闭所有插件

## 配置项

```yaml
plugins:
  enabled:
    - console
    - feishu

plugin:
  feishu:
    app_id: ${FEISHU_APP_ID}
    app_secret: ${FEISHU_APP_SECRET}
```
