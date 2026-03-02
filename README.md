# GoPaw

**轻量级、插件化的个人 AI 助理工作台（Go 语言实现）**

GoPaw 从 CoPaw（Python 版）吸取经验重新设计，核心目标：极轻量（< 150MB）、真插件化、服务器友好、Docker 一键启动。

## 核心特性

- **ReAct Agent**：Reasoning + Acting 推理循环，支持工具调用
- **多频道接入**：飞书、钉钉、Web Console、Webhook
- **三层 Skill 体系**：Prompt Skill（零代码）/ Config Skill（低代码）/ Code Skill（完整代码）
- **内置工具集**：文件读写、Shell、Web 搜索（Tavily）、HTTP
- **定时任务**：Cron 表达式调度，支持活跃时间窗口
- **持久化记忆**：SQLite + FTS5 全文搜索，支持上下文压缩
- **配置热加载**：修改 config.yaml 无需重启

## 快速开始

### 开发模式（推荐开发者）

```bash
# 需要 Go 1.22+、Node.js 18+、pnpm
git clone https://github.com/gopaw/gopaw.git && cd gopaw
go mod download && make web-install
go run ./cmd/gopaw init
make dev
# 前端 http://localhost:5173（HMR 热更新）
# API  http://localhost:8088
```

### 生产模式（单二进制部署）

```bash
make build          # 前端构建（压缩）+ 嵌入 Go 二进制
./gopaw init        # 生成 config.yaml
./gopaw start       # 单进程启动，访问 http://localhost:8088
```

### Docker

```bash
cp config.yaml.example config.yaml
docker compose up -d
open http://localhost:8088
```

> 首次启动后，通过 Web UI → 设置 → LLM 提供商 配置 API Key，无需修改配置文件。

## CLI 命令

| 命令 | 说明 |
|------|------|
| `gopaw init` | 生成默认 config.yaml |
| `gopaw start [--config path]` | 启动服务 |
| `gopaw version` | 显示版本信息 |

## 目录结构

```
gopaw/
├── cmd/gopaw/         # 程序入口
├── internal/          # 核心业务逻辑（不对外暴露）
│   ├── agent/         # ReAct Agent 引擎
│   ├── memory/        # 记忆系统（SQLite + FTS5）
│   ├── channel/       # 频道管理
│   ├── skill/         # Skill 加载器
│   ├── tool/          # Tool 注册表
│   ├── llm/           # LLM 客户端
│   ├── scheduler/     # Cron 调度器
│   ├── server/        # HTTP/WebSocket 服务
│   └── config/        # 配置管理
├── pkg/               # 公共接口（插件开发者使用）
│   ├── plugin/        # ChannelPlugin / Tool / Skill 接口
│   └── types/         # 统一消息类型
├── plugins/           # 内置插件实现
│   ├── channels/      # feishu / dingtalk / console / webhook
│   └── tools/         # file_read / file_write / shell / web_search / http_get
└── skills/            # 用户自定义 Skill 目录
```

## 配置说明

详见 `config.yaml.example`，每个字段均有注释。关键配置项：

| 配置项 | 说明 | 热加载 |
|--------|------|--------|
| `llm.api_key` | LLM API Key | 是 |
| `llm.model` | 使用的模型 | 是 |
| `agent.system_prompt` | 系统提示 | 是 |
| `plugins.enabled` | 启用的频道插件 | 否（需重启） |
| `server.port` | 监听端口 | 否（需重启） |

## REST API

| 路由 | 方法 | 说明 |
|------|------|------|
| `/api/agent/chat` | POST | 发送消息 |
| `/api/agent/chat/stream` | GET | SSE 流式响应 |
| `/api/agent/sessions` | GET | 列出所有会话 |
| `/api/skills` | GET | 列出所有技能 |
| `/api/channels/health` | GET | 频道健康状态 |
| `/api/cron` | GET/POST | 定时任务管理 |
| `/api/system/version` | GET | 版本信息 |
| `/health` | GET | 健康检查 |
| `/ws` | WS | WebSocket 双向通信 |

## 插件开发

### 开发频道插件

实现 `pkg/plugin.ChannelPlugin` 接口，并在 `init()` 中注册：

```go
package myplugin

import "github.com/gopaw/gopaw/internal/channel"

type MyPlugin struct { ... }

func (p *MyPlugin) Name() string { return "my_channel" }
// ... 实现所有接口方法

func init() {
    channel.Register(&MyPlugin{})
}
```

### 开发工具插件

实现 `pkg/plugin.Tool` 接口，并在 `init()` 中注册：

```go
package mytools

import "github.com/gopaw/gopaw/internal/tool"

type MyTool struct{}

func (t *MyTool) Name() string { return "my_tool" }
// ... 实现所有接口方法

func init() {
    tool.Register(&MyTool{})
}
```

### 创建 Skill

在 `skills/` 目录下创建子目录，添加 `manifest.yaml` 和 `prompt.md`：

```yaml
# skills/my_skill/manifest.yaml
name: my_skill
version: 1.0.0
display_name: 我的技能
level: 1
```

```markdown
<!-- skills/my_skill/prompt.md -->
## 我的技能能力描述
当用户询问...时，你应该...
```

## 构建说明

```bash
make dev            # 开发模式：Vite HMR + Go 后端（不 embed 前端）
make build          # 生产模式：前端压缩 + 嵌入 Go 二进制 → 单文件
make test           # 运行所有测试
make docker-build   # 构建 Docker 镜像
make lint           # 代码检查
```

## 系统要求

- **生产部署**：Docker 或 Go 1.22+ 运行时
- **最低内存**：50MB（不含 LLM API 响应）
- **存储**：SQLite 数据库（默认 `data/gopaw.db`）
- **网络**：能访问 LLM API（OpenAI 或兼容接口）

## License

Apache 2.0
