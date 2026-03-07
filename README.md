<p align="center">
  <img src="assets/logo.png" width="80" alt="GoPaw Logo" />
  <h1 align="center">GoPaw</h1>
</p>

<p align="center">
  <a href="https://github.com/gopaw/gopaw/releases"><img src="https://img.shields.io/github/v/release/gopaw/gopaw?style=flat-square" alt="Release"></a>
  <a href="https://github.com/gopaw/gopaw/actions"><img src="https://img.shields.io/github/actions/workflow/status/gopaw/gopaw/ci.yml?branch=main&style=flat-square" alt="Build Status"></a>
  <a href="https://github.com/gopaw/gopaw/blob/main/LICENSE"><img src="https://img.shields.io/github/license/gopaw/gopaw?style=flat-square" alt="License"></a>
  <a href="https://github.com/gopaw/gopaw/releases"><img src="https://img.shields.io/github/downloads/gopaw/gopaw/latest/total?style=flat-square" alt="Downloads"></a>
  <a href="https://golang.org/doc/devel/release.html#go1.22"><img src="https://img.shields.io/github/go-mod/go-version/gopaw/gopaw?style=flat-square" alt="Go Version"></a>
</p>

<p align="center">
  <a href="README.md">🇨🇳 中文</a> · 
  <a href="README.en.md">🇺🇸 English</a> · 
  <a href="README.ja.md">🇯🇵 日本語</a>
</p>

---

## 🐾 轻量级 AI 助理工作台

**GoPaw** 是一个用 Go 语言实现的轻量级、插件化个人 AI 助理工作台。通过 ReAct 推理循环、多频道接入和三层技能体系，让你快速构建专属的 AI 助理。

### 核心优势

| 特性 | 说明 |
|------|------|
| 🚀 **极轻量** | 内存占用 < 150MB，单二进制文件，无浏览器依赖 |
| 🔌 **真插件化** | 频道、工具、技能均可插件化，按需加载 |
| 🖥️ **服务器友好** | 支持 Docker 一键部署，无需 GUI 环境 |
| 🎯 **低门槛** | 普通用户无需懂代码，开发者可自由扩展 |

---

## ✨ 核心特性

### 🧠 ReAct Agent

基于 ReAct（Reasoning + Acting）推理循环，支持：
- **Thought-Action-Observation** 循环推理
- **多工具调用** - 文件操作、Shell、Web 搜索、HTTP 请求
- **上下文感知** - 自动加载历史对话和记忆

### 📺 多频道接入

| 频道 | 说明 | 配置方式 |
|------|------|---------|
| **飞书** | 企业IM，支持群聊/私聊 | Web UI 配置 AppID/Secret |
| **钉钉** | 企业IM，支持群聊/私聊 | Web UI 配置 ClientID/Secret |
| **Web Console** | 内置网页控制台 | 启动即访问 http://localhost:8088 |
| **Webhook** | 标准 HTTP 接口 | 支持自定义回调 URL |

### 🎨 三层 Skill 体系

```
Level 1: Prompt Skill（零代码）
  └─ manifest.yaml + prompt.md
  └─ 通过提示词注入扩展能力

Level 2: Config Skill（低代码）
  └─ workflow.yaml 编排多步骤任务
  └─ 组合已有 Tools 完成复杂流程

Level 3: Code Skill（完整代码）
  └─ skill.go 实现自定义 Tool
  └─ 完整控制执行逻辑
```

### 🛠️ 内置工具集

| 工具 | 功能 | 示例 |
|------|------|------|
| `file_read` / `file_write` | 文件读写 | 读取配置文件、写入日志 |
| `shell_execute` | Shell 命令执行 | 运行脚本、系统管理 |
| `web_search` | 网络搜索（Tavily） | 查询实时信息 |
| `http_get` / `http_post` | HTTP 请求 | 调用外部 API |

### ⏰ 定时任务

- **Cron 表达式调度** - 支持秒级精度
- **活跃时间窗口** - 避免非打扰时段
- **隔离会话** - 不污染主对话历史

### 💾 持久化记忆

- **SQLite + FTS5** - 全文搜索支持
- **上下文压缩** - 自动摘要历史对话
- **长期记忆归档** - 定期压缩存储

### 🔧 配置热加载

- **config.yaml** - 修改后自动重载
- **AGENT.md** - 系统提示即时生效
- **技能管理** - Web UI 动态启用/禁用

---

## 🚀 快速开始

### Docker 部署（推荐）

```bash
# 1. 准备配置文件
cp config.yaml.example config.yaml

# 2. 启动服务
docker compose up -d

# 3. 访问 Web UI
open http://localhost:8088
```

> 💡 **提示**：首次启动后，通过 Web UI → 设置 → LLM 提供商 配置 API Key，无需修改配置文件。

### 本地开发

```bash
# 前置要求：Go 1.22+、Node.js 18+、pnpm
git clone https://github.com/gopaw/gopaw.git && cd gopaw

# 安装依赖
go mod download && make web-install

# 初始化配置
go run ./cmd/gopaw init

# 启动开发服务器
make dev
```

访问：
- **前端（HMR 热更新）**: http://localhost:5173
- **后端 API**: http://localhost:8088

### 生产模式（单二进制）

```bash
# 构建
make build

# 初始化并启动
./gopaw init
./gopaw start
```

---

## 📚 文档导航

| 文档 | 说明 |
|------|------|
| [部署指南](docker/DEPLOY.md) | Docker 部署、服务器配置、运维命令 |
| [技能开发](skills/SKILLS.md) | 创建自定义技能、Prompt 编写指南 |
| [插件规范](GoPaw_Design.md#10-插件规范) | 开发频道插件、工具插件 |
| [API 参考](#rest-api) | REST API、WebSocket 接口 |

---

## 🔧 CLI 命令

| 命令 | 说明 |
|------|------|
| `gopaw init` | 生成默认 config.yaml |
| `gopaw start [--config path]` | 启动服务 |
| `gopaw version` | 显示版本信息 |

---

## 📁 项目结构

```
gopaw/
├── cmd/gopaw/         # 程序入口
├── internal/          # 核心业务逻辑
│   ├── agent/         # ReAct Agent 引擎
│   ├── memory/        # 记忆系统（SQLite + FTS5）
│   ├── channel/       # 频道管理
│   ├── skill/         # Skill 加载器
│   ├── tool/          # Tool 注册表
│   ├── llm/           # LLM 客户端
│   ├── scheduler/     # Cron 调度器
│   ├── server/        # HTTP/WebSocket 服务
│   ├── config/        # 配置管理
│   ├── platform/      # 内置频道插件
│   └── tools/         # 内置工具
├── pkg/               # 公共接口（插件开发者使用）
│   ├── plugin/        # ChannelPlugin / Tool / Skill 接口
│   └── types/         # 统一消息类型
└── skills/            # 用户自定义 Skill 目录
```

---

## 🌐 REST API

| 路由 | 方法 | 说明 |
|------|------|------|
| `/api/agent/chat` | POST | 发送消息 |
| `/api/agent/chat/stream` | GET | SSE 流式响应 |
| `/api/agent/sessions` | GET | 列出所有会话 |
| `/api/skills` | GET/PUT | 技能管理 |
| `/api/channels/health` | GET | 频道健康状态 |
| `/api/cron` | GET/POST | 定时任务管理 |
| `/api/system/version` | GET | 版本信息 |
| `/health` | GET | 健康检查 |
| `/ws` | WS | WebSocket 双向通信 |

---

## 🧩 插件开发

### 开发频道插件

```go
package myplugin

import "github.com/gopaw/gopaw/internal/channel"

type MyPlugin struct{}

func (p *MyPlugin) Name() string { return "my_channel" }
// ... 实现所有接口方法

func init() {
    channel.Register(&MyPlugin{})
}
```

### 开发工具插件

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

在 `skills/` 目录下创建子目录：

```yaml
# skills/my_skill/manifest.yaml
name: my_skill
version: 1.0.0
display_name: 我的技能
level: 1  # 1=Prompt / 2=Config / 3=Code
```

```markdown
<!-- skills/my_skill/prompt.md -->
## 我的技能能力描述
当用户询问...时，你应该...
```

---

## 🛠️ 构建命令

```bash
make dev            # 开发模式：Vite HMR + Go 后端
make build          # 生产模式：前端压缩 + 嵌入 Go 二进制
make test           # 运行所有测试
make docker-build   # 构建 Docker 镜像
make lint           # 代码检查
```

---

## 📋 系统要求

| 环境 | 要求 |
|------|------|
| **生产部署** | Docker 或 Go 1.22+ 运行时 |
| **最低内存** | 50MB（不含 LLM API 响应） |
| **存储** | SQLite 数据库（默认 `~/.gopaw/gopaw.db`） |
| **网络** | 能访问 LLM API（OpenAI 或兼容接口） |

---

## 📄 License

AGPL-3.0

<p align="center">
  Designed with ❤️ by <a href="https://github.com/xiaodou997">xiaodou997</a>
</p>
