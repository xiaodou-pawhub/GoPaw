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

## 🐾 Lightweight AI Assistant Workbench

**GoPaw** is a lightweight, plugin-based personal AI assistant workbench implemented in Go. With ReAct reasoning loop, multi-channel integration, and a three-level skill system, you can quickly build your own专属 AI assistant.

### Core Advantages

| Feature | Description |
|---------|-------------|
| 🚀 **Ultra Lightweight** | Memory < 150MB, single binary, no browser dependency |
| 🔌 **True Plugin Architecture** | Channels, tools, and skills are all plugin-based, load on demand |
| 🖥️ **Server-Friendly** | One-click Docker deployment, no GUI required |
| 🎯 **Low Barrier** | No coding required for users, fully extensible for developers |

---

## ✨ Core Features

### 🧠 ReAct Agent

Based on ReAct (Reasoning + Acting) reasoning loop:
- **Thought-Action-Observation** cyclic reasoning
- **Multi-tool invocation** - File operations, Shell, Web search, HTTP requests
- **Context-aware** - Auto-loads conversation history and memory

### 📺 Multi-Channel Integration

| Channel | Description | Configuration |
|---------|-------------|---------------|
| **Feishu/Lark** | Enterprise IM, group/private chat | Web UI: AppID/Secret |
| **DingTalk** | Enterprise IM, group/private chat | Web UI: ClientID/Secret |
| **Web Console** | Built-in web console | Access http://localhost:8088 |
| **Webhook** | Standard HTTP interface | Custom callback URL supported |

### 🎨 Three-Level Skill System

```
Level 1: Prompt Skill (Zero Code)
  └─ manifest.yaml + prompt.md
  └─ Extend capabilities through prompt injection

Level 2: Config Skill (Low Code)
  └─ workflow.yaml orchestrates multi-step tasks
  └─ Combine existing Tools for complex workflows

Level 3: Code Skill (Full Code)
  └─ skill.go implements custom Tools
  └─ Full control over execution logic
```

### 🛠️ Built-in Tool Set

| Tool | Function | Example |
|------|----------|---------|
| `file_read` / `file_write` | File operations | Read config, write logs |
| `shell_execute` | Shell command execution | Run scripts, system admin |
| `web_search` | Web search (Tavily) | Query real-time information |
| `http_get` / `http_post` | HTTP requests | Call external APIs |

### ⏰ Scheduled Tasks

- **Cron expression scheduling** - Second-level precision
- **Active time window** - Avoid disturbance periods
- **Isolated sessions** - Don't pollute main conversation history

### 💾 Persistent Memory

- **SQLite + FTS5** - Full-text search support
- **Context compression** - Auto-summarize historical conversations
- **Long-term memory archiving** - Periodic compression storage

### 🔧 Hot Configuration Reload

- **config.yaml** - Auto-reload on modification
- **AGENT.md** - System prompt takes effect immediately
- **Skill management** - Web UI dynamic enable/disable

---

## 🚀 Quick Start

### Docker Deployment (Recommended)

```bash
# 1. Prepare configuration
cp config.yaml.example config.yaml

# 2. Start service
docker compose up -d

# 3. Access Web UI
open http://localhost:8088
```

> 💡 **Tip**: After first startup, configure API Key via Web UI → Settings → LLM Provider, no need to modify config files.

### Local Development

```bash
# Prerequisites: Go 1.22+, Node.js 18+, pnpm
git clone https://github.com/gopaw/gopaw.git && cd gopaw

# Install dependencies
go mod download && make web-install

# Initialize configuration
go run ./cmd/gopaw init

# Start development server
make dev
```

Access:
- **Frontend (HMR hot reload)**: http://localhost:5173
- **Backend API**: http://localhost:8088

### Production Mode (Single Binary)

```bash
# Build
make build

# Initialize and start
./gopaw init
./gopaw start
```

---

## 📚 Documentation

| Document | Description |
|----------|-------------|
| [Deployment Guide](docker/DEPLOY.md) | Docker deployment, server configuration,运维 commands |
| [Skill Development](skills/SKILLS.md) | Create custom skills, Prompt writing guide |
| [Plugin Specification](GoPaw_Design.md#10-plugin-specification) | Develop channel plugins, tool plugins |
| [API Reference](#rest-api) | REST API, WebSocket interface |

---

## 🔧 CLI Commands

| Command | Description |
|---------|-------------|
| `gopaw init` | Generate default config.yaml |
| `gopaw start [--config path]` | Start service |
| `gopaw version` | Show version information |

---

## 📁 Project Structure

```
gopaw/
├── cmd/gopaw/         # Program entry
├── internal/          # Core business logic
│   ├── agent/         # ReAct Agent engine
│   ├── memory/        # Memory system (SQLite + FTS5)
│   ├── channel/       # Channel management
│   ├── skill/         # Skill loader
│   ├── tool/          # Tool registry
│   ├── llm/           # LLM client
│   ├── scheduler/     # Cron scheduler
│   ├── server/        # HTTP/WebSocket service
│   ├── config/        # Configuration management
│   ├── platform/      # Built-in channel plugins
│   └── tools/         # Built-in tools
├── pkg/               # Public interfaces (for plugin developers)
│   ├── plugin/        # ChannelPlugin / Tool / Skill interfaces
│   └── types/         # Unified message types
└── skills/            # User-defined Skill directory
```

---

## 🌐 REST API

| Route | Method | Description |
|-------|--------|-------------|
| `/api/agent/chat` | POST | Send message |
| `/api/agent/chat/stream` | GET | SSE streaming response |
| `/api/agent/sessions` | GET | List all sessions |
| `/api/skills` | GET/PUT | Skill management |
| `/api/channels/health` | GET | Channel health status |
| `/api/cron` | GET/POST | Scheduled task management |
| `/api/system/version` | GET | Version information |
| `/health` | GET | Health check |
| `/ws` | WS | WebSocket bidirectional communication |

---

## 🧩 Plugin Development

### Develop Channel Plugin

```go
package myplugin

import "github.com/gopaw/gopaw/internal/channel"

type MyPlugin struct{}

func (p *MyPlugin) Name() string { return "my_channel" }
// ... implement all interface methods

func init() {
    channel.Register(&MyPlugin{})
}
```

### Develop Tool Plugin

```go
package mytools

import "github.com/gopaw/gopaw/internal/tool"

type MyTool struct{}

func (t *MyTool) Name() string { return "my_tool" }
// ... implement all interface methods

func init() {
    tool.Register(&MyTool{})
}
```

### Create Skill

Create subdirectory in `skills/`:

```yaml
# skills/my_skill/manifest.yaml
name: my_skill
version: 1.0.0
display_name: My Skill
level: 1  # 1=Prompt / 2=Config / 3=Code
```

```markdown
<!-- skills/my_skill/prompt.md -->
## My Skill Capability Description
When user asks..., you should...
```

---

## 🛠️ Build Commands

```bash
make dev            # Development mode: Vite HMR + Go backend
make build          # Production mode: Frontend minified + embedded Go binary
make test           # Run all tests
make docker-build   # Build Docker image
make lint           # Code linting
```

---

## 📋 System Requirements

| Environment | Requirements |
|-------------|--------------|
| **Production** | Docker or Go 1.22+ runtime |
| **Minimum Memory** | 50MB (excluding LLM API response) |
| **Storage** | SQLite database (default `~/.gopaw/gopaw.db`) |
| **Network** | Access to LLM API (OpenAI or compatible) |

---

## 📄 License

AGPL-3.0

<p align="center">
  Designed with ❤️ by <a href="https://github.com/xiaodou997">xiaodou997</a>
</p>
