# [feat] Workspace 目录统一 + 插件迁移 + 对话日志

**报告日期**: 2026-03-03
**开发者**: 小M（AI 助手）
**关联任务**: M-1、M-2、M-3、M-4（ai/handoff-xiaom.md）
**涉及文件数**: 20+ 个

---

## 功能概述

本次开发完成了四个任务：

1. **M-1 Workspace 目录统一**：引入 `~/.gopaw` 统一根目录，所有运行时数据（数据库、日志、Agent 文件）从中派生
2. **M-2 移除 plugins.enabled 配置**：简化插件启动逻辑，配置了 API Key 即自动激活
3. **M-3 plugins/ → internal/ 迁移**：将插件代码迁移到 `internal/` 目录，符合 Go 项目惯例
4. **M-4 对话日志系统**：新增 NDJSON 格式的对话事件日志，便于事后分析

---

## 实现说明

### M-1：Workspace 目录统一

#### 核心逻辑

创建 `internal/workspace/workspace.go`，提供统一的路径解析：

```go
type Paths struct {
    Root          string // ~/.gopaw
    DBFile        string // ~/.gopaw/gopaw.db
    AgentMDFile   string // ~/.gopaw/agent/AGENT.md
    PersonaMDFile string // ~/.gopaw/agent/PERSONA.md
    ContextMDFile string // ~/.gopaw/agent/CONTEXT.md
    MemoryMDFile  string // ~/.gopaw/agent/MEMORY.md
    LogFile       string // ~/.gopaw/logs/gopaw.log
    ConvLogFile   string // ~/.gopaw/logs/conversations.jsonl
    SkillsDir     string // ~/.gopaw/skills/
}

func Resolve(dir string) (*Paths, error) {
    expanded := expandTilde(dir)
    abs, err := filepath.Abs(expanded)
    // ...
}
```

**审查要点**：
- ✅ `expandTilde` 支持 `~` 展开到用户主目录
- ✅ `EnsureDirs` 创建所有必需子目录
- ⚠️ 迁移时需考虑旧数据路径（当前未做自动迁移）

#### 配置变更

移除 `storage.path`，新增 `workspace.dir`：

```yaml
# config.yaml（新）
workspace:
  dir: ~/.gopaw

# config.yaml（旧，已移除）
storage:
  type: sqlite
  path: data/gopaw.db
```

### M-2：移除 plugins.enabled 配置

#### 核心逻辑

修改 `channel/manager.go` 的 `Start` 方法，自动遍历所有已注册插件：

```go
func (m *Manager) Start(ctx context.Context, pluginCfgs map[string]json.RawMessage) error {
    m.ctx = ctx
    for _, p := range m.registry.All() {
        name := p.Name()
        cfg := pluginCfgs[name]
        if cfg == nil {
            cfg = json.RawMessage("{}")
        }
        // Init 失败则跳过，打 Warning
        if err := p.Init(cfg); err != nil {
            m.logger.Warn("channel plugin init failed, skipping",
                zap.String("plugin", name), zap.Error(err))
            continue
        }
        // ...
    }
    return nil
}
```

**审查要点**：
- ✅ 不再依赖 `enabled` 列表，简化配置
- ✅ Init 失败时打印 Warning 并跳过，不会阻止其他插件启动
- ⚠️ 需确保各插件的 `Init()` 在缺少必要配置时返回明确错误

### M-3：plugins/ → internal/ 迁移

#### 目录映射

| 源路径 | 目标路径 |
|--------|---------|
| `plugins/channels/feishu/` | `internal/platform/feishu/` |
| `plugins/channels/dingtalk/` | `internal/platform/dingtalk/` |
| `plugins/channels/webhook/` | `internal/platform/webhook/` |
| `plugins/channels/console/` | `internal/platform/console/` |
| `plugins/tools/` | `internal/tools/` |

#### Import 路径替换

```go
// 旧
_ "github.com/gopaw/gopaw/plugins/channels/feishu"
_ "github.com/gopaw/gopaw/plugins/tools"

// 新
_ "github.com/gopaw/gopaw/internal/platform/feishu"
_ "github.com/gopaw/gopaw/internal/tools"
```

### M-4：对话日志系统

#### 核心逻辑

创建 `internal/convlog/convlog.go`，写入 NDJSON 格式日志：

```go
type Event struct {
    Ts        time.Time       `json:"ts"`
    SessionID string          `json:"session_id"`
    Event     string          `json:"event"`
    Content   string          `json:"content,omitempty"`
    Tool      string          `json:"tool,omitempty"`
    Input     json.RawMessage `json:"input,omitempty"`
    Output    string          `json:"output,omitempty"`
    Error     *string         `json:"error,omitempty"`
    Tokens    *TokenUsage     `json:"tokens,omitempty"`
}
```

事件类型：`user_message`、`tool_call`、`tool_result`、`agent_reply`

**审查要点**：
- ✅ 使用 `sync.Mutex` 保证并发安全
- ✅ 日志文件按行追加，每行一个 JSON 对象
- ⚠️ 日志文件可能无限增长，后续需考虑轮转机制

---

## 文件变更清单

### 新增文件

| 文件 | 说明 |
|------|------|
| `internal/workspace/workspace.go` | Workspace 路径解析 |
| `internal/convlog/convlog.go` | 对话日志记录器 |
| `internal/server/handlers/workspace.go` | Workspace 文件 API |
| `internal/platform/console/plugin.go` | Console 频道插件（迁移） |
| `internal/platform/dingtalk/plugin.go` | 钉钉频道插件（迁移） |
| `internal/platform/feishu/plugin.go` | 飞书频道插件（迁移） |
| `internal/platform/webhook/plugin.go` | Webhook 频道插件（迁移） |
| `internal/tools/*.go` | 内置工具（迁移） |

### 修改文件

| 文件 | 变更说明 |
|------|---------|
| `internal/config/config.go` | 新增 `WorkspaceConfig`，移除 `StorageConfig` |
| `internal/config/config_test.go` | 更新测试用例匹配新配置结构 |
| `internal/agent/agent.go` | 注入 `convlog.Logger`，记录对话事件 |
| `internal/channel/manager.go` | 简化 `Start` 方法，自动发现插件 |
| `internal/server/server.go` | 注册 `/api/workspace/*` 路由 |
| `internal/server/handlers/config.go` | 返回 `workspace` 配置替代 `storage` |
| `cmd/gopaw/main.go` | 初始化 workspace、convlog，更新插件 import 路径 |
| `config.yaml.example` | 移除旧配置，新增 workspace.dir |
| `internal/convlog/convlog.go` | 修复路径解析 panic 风险 |
| `README.md` | 更新目录结构、配置说明、系统要求 |
| `GoPaw_Design.md` | 更新目录结构、config.yaml 示例、热加载表格 |

### 删除文件

| 文件 | 说明 |
|------|------|
| `plugins/` 整个目录 | 已迁移到 `internal/` |
| `reports/2026-03/*.md` | 清理旧报告（git 历史可追溯） |
| `scripts/*.sh` | 清理临时脚本 |

**变更统计**：新增约 600 行 / 修改约 200 行 / 删除约 400 行

---

## 接口 / API 变更

### 新增接口

| 接口 | 方法 | 说明 |
|------|------|------|
| `/api/workspace/agent` | GET/PUT | 读写 AGENT.md |
| `/api/workspace/persona` | GET/PUT | 读写 PERSONA.md |
| `/api/workspace/context` | GET/PUT | 读写 CONTEXT.md |
| `/api/workspace/memory` | GET/PUT | 读写 MEMORY.md |

### 修改接口

| 接口 | 变更 |
|------|------|
| `GET /api/config` | 返回 `workspace` 替代 `storage` |

---

## 自检结果

```bash
go build ./...      ✅ 通过
go vet ./...        ✅ 通过
go test -race ./... ✅ 通过（config_test.go 已更新）
```

---

## Review 反馈修复

### P0-1：config.yaml.example 未同步更新

**问题**：config.yaml.example 仍保留旧的 `storage` 和 `plugins.enabled` 配置。

**修复**：
- 移除 `storage.type` 和 `storage.path` 配置
- 移除 `plugins.enabled` 列表
- 新增 `workspace.dir` 配置（默认 `~/.gopaw`）

### P0-2：convlog.go 路径解析存在 panic 风险

**问题**：`convlog.go:48` 使用字符串切片 `path[:len(path)-len("/conversations.jsonl")]` 解析父目录，若路径不包含预期后缀会导致 panic。

**修复**：使用 `filepath.Dir(path)` 替代字符串切片操作。

```go
// 修复前
if err := os.MkdirAll(path[:len(path)-len("/conversations.jsonl")], 0o755); err != nil {

// 修复后
if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
```

### P1：README.md / GoPaw_Design.md 文档需更新

**修复**：
- 更新 README.md 目录结构（移除 `plugins/`，新增 `internal/platform/`、`internal/tools/`、`internal/workspace/`、`internal/convlog/`）
- 更新 README.md 配置说明（移除 `plugins.enabled`，新增 `workspace.dir`）
- 更新 README.md 系统要求（存储路径改为 `~/.gopaw/gopaw.db`）
- 更新 GoPaw_Design.md 目录结构
- 更新 GoPaw_Design.md config.yaml 示例
- 更新 GoPaw_Design.md 热加载汇总表格

---

## 验收标准完成情况

### M-1：Workspace 目录统一

- [x] `config.yaml` 增加 `workspace.dir` 字段
- [x] `internal/config/config.go` 增加 `WorkspaceConfig`
- [x] 新增 `internal/workspace/workspace.go`
- [x] 启动代码使用 `workspace.Paths`
- [x] 新增 Workspace 文件 API handlers
- [x] 编译通过

### M-2：移除 plugins.enabled

- [x] 简化 `channel.Manager.Start()` 方法
- [x] 插件自动发现并启动
- [x] 编译通过

### M-3：plugins/ → internal/ 迁移

- [x] 文件迁移完成
- [x] Import 路径替换完成
- [x] 删除旧 `plugins/` 目录
- [x] 编译通过

### M-4：对话日志系统

- [x] 新增 `internal/convlog/convlog.go`
- [x] Agent 中注入 Logger
- [x] 记录 `user_message`、`tool_call`、`tool_result`、`agent_reply` 事件
- [x] 编译通过

---

## 遗留事项

1. **旧数据迁移**：用户已有的 `data/gopaw.db` 需手动迁移到 `~/.gopaw/gopaw.db`
2. **对话日志轮转**：`conversations.jsonl` 文件可能无限增长，需后续增加轮转机制
3. **前端适配**：新增的 `/api/workspace/*` 接口需前端调用（供小G 的 G-1/G-2 任务使用）

---

## 审查清单

### 代码逻辑

- [ ] `workspace.Resolve()` 路径解析逻辑是否正确
- [ ] `convlog.Logger` 并发写入是否安全
- [ ] 插件迁移后功能是否正常

### 并发 & 错误处理

- [ ] `convlog.Logger` 使用 `sync.Mutex` 保护文件写入
- [ ] 错误是否向上传递，未被静默吞掉

### 功能验证

- [ ] 启动后 `~/.gopaw/` 目录结构是否自动创建
- [ ] 对话日志是否正确写入 `~/.gopaw/logs/conversations.jsonl`
- [ ] 插件迁移后是否正常工作

### 测试覆盖

- [ ] `internal/config/config_test.go` 测试用例已更新
- [ ] `go test -race` 是否通过
