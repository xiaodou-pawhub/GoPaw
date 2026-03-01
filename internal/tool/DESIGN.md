# Tool 模块设计文档

## 职责

Tool 模块维护 Agent 可调用的工具注册表，并提供统一的执行入口：
- 提供线程安全的 Registry（增删查）
- 维护全局注册表，供各 `init()` 函数自动注册内置工具
- Executor 封装参数解析和错误处理，统一调用日志

Tool 模块**不负责**：
- 工具本身的业务逻辑（在 `plugins/tools/` 中实现）
- 工具的权限控制（v0.3 引入）

## 架构图

```mermaid
flowchart LR
    subgraph plugins/tools
        FR[file_read.go]
        FW[file_write.go]
        SH[shell.go]
        WS[web_search.go]
    end

    subgraph internal/tool
        REG[Registry]
        EXE[Executor]
    end

    FR -->|init: Register| REG
    FW -->|init: Register| REG
    SH -->|init: Register| REG
    WS -->|init: Register| REG

    Agent -->|Execute(name, args)| EXE
    EXE -->|Get(name)| REG
    EXE -->|Execute(ctx, args)| plugins/tools
```

## 核心接口

```go
type Registry struct { ... }

func NewRegistry() *Registry
func Register(t plugin.Tool)           // 全局注册表快捷方式
func Global() *Registry                // 获取全局注册表

func (r *Registry) Register(t plugin.Tool)
func (r *Registry) Unregister(name string)
func (r *Registry) Get(name string) (plugin.Tool, error)
func (r *Registry) All() []plugin.Tool
func (r *Registry) Names() []string

type Executor struct { ... }

func NewExecutor(registry *Registry, logger *zap.Logger) *Executor
func (e *Executor) Execute(ctx context.Context, toolName, argsJSON string) (string, error)
```

## 关键设计决策

1. **全局注册表 + init()**：内置工具在自己的 package init() 中调用 `tool.Register()`，无需修改任何核心代码即可添加新工具。
2. **JSON 参数解析在 Executor**：Agent 从 LLM 拿到 JSON 字符串参数，Executor 负责解析，工具实现者不需要关心序列化。
3. **读写锁**：Registry 使用 `sync.RWMutex`，允许并发读取（多个会话同时调用工具），写入（启动时注册）串行化。

## 依赖关系

- **依赖**：`pkg/plugin`（Tool 接口）、`go.uber.org/zap`
- **被依赖**：`internal/agent`（Executor）、`plugins/tools/*`（Register）

## 验收标准

- [ ] 内置工具通过 `init()` 自动注册，主程序无需手动调用
- [ ] 未注册的工具名返回清晰错误，不 panic
- [ ] 参数 JSON 解析失败时返回包含工具名的错误信息
- [ ] Registry 在高并发下无数据竞争（go race detector 无报告）
