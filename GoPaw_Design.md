# GoPaw 设计文档

> 版本：v0.1.0
> 状态：设计阶段
> 最后更新：2026-03-01

---

## 目录

1. [项目概述](#1-项目概述)
2. [设计原则](#2-设计原则)
3. [整体架构](#3-整体架构)
4. [Agent 引擎](#4-agent-引擎)
5. [频道系统](#5-频道系统)
6. [Skills 系统](#6-skills-系统)
7. [Tools 系统](#7-tools-系统)
8. [记忆系统](#8-记忆系统)
9. [调度系统](#9-调度系统)
10. [插件规范](#10-插件规范)
11. [数据层设计](#11-数据层设计)
12. [API 设计](#12-api-设计)
13. [配置管理](#13-配置管理)
14. [Web Console](#14-web-console)
15. [部署方案](#15-部署方案)
16. [开发路线图](#16-开发路线图)
17. [商业模式](#17-商业模式)

---

## 1. 项目概述

### 1.1 什么是 GoPaw

GoPaw 是一个**轻量级、插件化的个人 AI 助理工作台**，使用 Go 语言实现。它允许用户通过飞书、钉钉、Web 界面或 Webhook 等渠道与 AI 交互，并通过可扩展的 Skills 和 Tools 系统完成各类自动化任务。

GoPaw 从 CoPaw（Python 版）吸取经验，重新设计，核心目标是：

- **极轻量**：无浏览器、无桌面环境，核心内存占用 < 100MB
- **真插件化**：频道、技能、工具均可插件化，按需加载
- **低门槛**：普通用户无需懂代码，开发者可自由扩展
- **服务友好**：面向服务器部署，Docker 一键启动

### 1.2 与 CoPaw 的关系

| 维度 | CoPaw（Python） | GoPaw（Go） |
|------|----------------|------------|
| 内存占用 | ~1.3GB（含桌面） | < 150MB |
| 启动时间 | 5-10s | < 1s |
| 部署方式 | pip + Python 环境 | 单二进制 / Docker |
| 插件加载 | 运行时 import | 接口注册 + 配置驱动 |
| 并发模型 | GIL 限制 | Goroutine 原生并发 |
| Skills | 代码 + Markdown | 三层体系（Prompt/Config/Code） |
| 桌面依赖 | xfce4 + xvfb | 无 |

### 1.3 目标用户

- **个人用户**：部署在自己服务器上的个人 AI 助理
- **小型团队/工作室**：为团队提供统一的 AI 助理服务（2-20人规模）
- **开发者**：基于 GoPaw 二次开发或贡献插件

---

## 2. 设计原则

```
P1  极简核心        核心引擎保持最小，功能通过插件扩展
P1  插件优先        频道、工具、技能全部插件化，接口标准化
P1  配置驱动        未启用的插件零资源占用，无代码侵入
P2  安全默认        最小权限，危险操作显式授权（沙箱延后实现）
P2  可观测          结构化日志，健康检查，错误可追踪
P3  开发者友好      清晰的插件接口规范，完善的文档
```

---

## 3. 整体架构

### 3.1 分层架构

```
┌─────────────────────────────────────────────────────────────┐
│                      接入层（Access）                        │
│                                                             │
│   ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│   │  Web Console │  │   REST API   │  │    WebSocket     │  │
│   │  (React SPA) │  │   (Gin)      │  │    (实时消息)    │  │
│   └──────────────┘  └──────────────┘  └──────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      应用层（App）                           │
│                                                             │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────────┐ │
│  │  Channel   │  │  Plugin    │  │   Scheduler            │ │
│  │  Manager   │  │  Manager   │  │   (Cron)               │ │
│  └────────────┘  └────────────┘  └────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      领域层（Domain）                        │
│                                                             │
│  ┌────────────┐  ┌────────────┐  ┌────────────────────────┐ │
│  │   Agent    │  │   Memory   │  │   Skills               │ │
│  │  (ReAct)   │  │  Manager   │  │   Registry             │ │
│  └────────────┘  └────────────┘  └────────────────────────┘ │
│                                                             │
│  ┌────────────┐  ┌────────────┐                            │ │
│  │   Tools    │  │    LLM     │                            │ │
│  │  Registry  │  │   Client   │                            │ │
│  └────────────┘  └────────────┘                            │ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     基础设施层（Infra）                      │
│                                                             │
│        SQLite（持久化）      Config（配置管理）              │
│        Logger（结构化日志）  FileSystem（文件访问）          │
└─────────────────────────────────────────────────────────────┘
```

### 3.2 消息流转

```
用户（飞书/钉钉/Web/Webhook）
        │
        ▼
  Channel Plugin
  （接收、格式标准化）
        │
        ▼
  Message Queue（带缓冲的 channel，1000）
        │
        ▼
  Agent Engine
  ├── 读取 Memory（历史上下文）
  ├── 构建 System Prompt（核心 + 激活的 Skills Prompt）
  ├── 调用 LLM
  └── ReAct 循环
        ├── 解析 Thought / Action / ActionInput
        ├── 执行 Tool（调用 Tools Registry）
        └── Observation → 继续循环 / 输出 Final Answer
        │
        ▼
  写入 Memory
        │
        ▼
  Channel Plugin（发送响应）
        │
        ▼
用户收到回复
```

### 3.3 目录结构

```
gopaw/
├── cmd/
│   └── gopaw/
│       └── main.go                 # 程序入口
├── internal/
│   ├── agent/                      # Agent 引擎
│   │   ├── agent.go                # ReActAgent 实现
│   │   ├── prompt.go               # System Prompt 构建
│   │   └── session.go              # 会话管理
│   ├── memory/                     # 记忆系统
│   │   ├── manager.go
│   │   ├── store.go                # SQLite 存储
│   │   └── compress.go             # 上下文压缩
│   ├── channel/                    # 频道管理
│   │   ├── manager.go
│   │   ├── registry.go
│   │   └── message.go              # 统一消息模型
│   ├── skill/                      # Skills 系统
│   │   ├── manager.go
│   │   ├── registry.go
│   │   ├── loader.go               # 三层 Skill 加载
│   │   └── executor.go
│   ├── tool/                       # Tools 系统
│   │   ├── registry.go
│   │   └── executor.go
│   ├── llm/                        # LLM 客户端
│   │   ├── client.go               # 接口定义
│   │   ├── openai.go               # OpenAI 兼容实现
│   │   └── custom.go               # 自定义接口实现
│   ├── scheduler/                  # 定时任务
│   │   ├── manager.go
│   │   └── job.go
│   ├── server/                     # HTTP 服务
│   │   ├── server.go
│   │   ├── ws.go                   # WebSocket
│   │   └── middleware.go
│   └── config/                     # 配置管理
│       ├── config.go
│       └── watcher.go              # 热加载
├── plugins/                        # 内置插件实现
│   ├── channels/
│   │   ├── feishu/
│   │   ├── dingtalk/
│   │   ├── console/                # Web Console 频道
│   │   └── webhook/
│   ├── tools/
│   │   ├── file_read.go
│   │   ├── file_write.go
│   │   ├── file_search.go
│   │   ├── shell.go
│   │   ├── web_search.go
│   │   └── http_get.go
│   └── skills/
│       ├── pdf/
│       ├── docx/
│       ├── xlsx/
│       ├── news/
│       └── cron_skill/
├── pkg/                            # 对外暴露的公共包（插件开发者使用）
│   ├── plugin/                     # 插件接口定义
│   │   ├── channel.go
│   │   ├── tool.go
│   │   └── skill.go
│   └── types/                      # 公共数据类型
│       ├── message.go
│       └── response.go
├── web/                            # 前端 Console
│   └── dist/                       # 构建产物（嵌入二进制）
├── skills/                         # 用户自定义 Skill 目录（运行时）
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
└── config.yaml.example
```

---

## 4. Agent 引擎

### 4.1 核心接口

```go
// pkg/plugin/agent.go

type Agent interface {
    // 处理一条用户消息，返回响应
    Process(ctx context.Context, req *Request) (*Response, error)
}

type Request struct {
    SessionID string            // 会话 ID（区分不同用户/频道）
    UserID    string            // 用户 ID
    Channel   string            // 来源频道
    Content   string            // 消息内容
    MsgType   MessageType       // text / image / file
    Files     []FileAttachment  // 附件（如有）
    Metadata  map[string]string
}

type Response struct {
    Content   string
    MsgType   MessageType
    Files     []FileAttachment
    Error     error
}
```

### 4.2 ReAct 实现

```go
// internal/agent/agent.go

type ReActAgent struct {
    llm       llm.Client
    tools     *tool.Registry
    skills    *skill.Registry
    memory    *memory.Manager
    maxSteps  int             // 默认 20，防无限循环
    sysPrompt string          // 基础系统提示
}

func (a *ReActAgent) Process(ctx context.Context, req *Request) (*Response, error) {
    // 1. 从 Memory 获取历史上下文
    history, _ := a.memory.GetContext(req.SessionID, 20)

    // 2. 构建完整 System Prompt
    //    = 基础 Prompt + 激活的 Skills Prompt 片段 + Tools 描述
    systemPrompt := a.buildSystemPrompt(req)

    // 3. 构建消息列表
    messages := buildMessages(systemPrompt, history, req.Content)

    // 4. ReAct 循环
    for step := 0; step < a.maxSteps; step++ {
        reply, err := a.llm.Chat(ctx, messages)
        if err != nil {
            return nil, err
        }

        parsed := parseReActOutput(reply)

        // Final Answer：直接返回
        if parsed.IsFinal {
            a.memory.Add(req.SessionID, req.Content, parsed.Answer)
            return &Response{Content: parsed.Answer}, nil
        }

        // Action：执行工具
        observation, err := a.tools.Execute(ctx, parsed.Action, parsed.ActionInput)
        if err != nil {
            observation = fmt.Sprintf("Error: %v", err)
        }

        // 将 Thought/Action/Observation 追加到消息
        messages = appendReActStep(messages, reply, observation)
    }

    return nil, errors.New("max steps reached")
}
```

### 4.3 System Prompt 构建

System Prompt 由三部分动态拼接：

```
┌─────────────────────────────────┐
│  基础 Prompt（来自 config.yaml） │
│  "你是一个个人助理，名字是..."   │
├─────────────────────────────────┤
│  Skills Prompt 片段（动态注入）  │
│  每个激活的 Skill 贡献一段描述  │
│  "你具备 PDF 处理能力，当用户..." │
├─────────────────────────────────┤
│  Tools 描述（自动生成）          │
│  所有可用工具的名称和参数说明    │
└─────────────────────────────────┘
```

---

## 5. 频道系统

### 5.1 频道插件接口

所有频道插件必须实现此接口：

```go
// pkg/plugin/channel.go

type ChannelPlugin interface {
    // 元信息
    Name() string        // 唯一标识，如 "feishu"
    DisplayName() string // 显示名称，如 "飞书"

    // 生命周期
    Init(cfg json.RawMessage) error
    Start(ctx context.Context) error
    Stop() error

    // 消息收发
    Receive() <-chan *Message   // 接收来自该频道的消息
    Send(msg *Message) error    // 向该频道发送消息

    // 健康检查
    Health() HealthStatus
}

// 统一消息模型
type Message struct {
    ID        string
    SessionID string      // 频道内的会话标识（如群ID+用户ID）
    UserID    string
    Channel   string
    Content   string
    MsgType   MessageType // text / image / file / markdown
    Files     []FileAttachment
    ReplyTo   string      // 引用消息 ID（如有）
    Timestamp int64
    Raw       interface{} // 原始平台消息，供插件内部使用
}
```

### 5.2 内置频道

| 频道 | 标识 | 说明 |
|------|------|------|
| Web Console | `console` | 内置，随核心启动，提供网页聊天界面 |
| 飞书 | `feishu` | 插件，需配置 App ID / Secret |
| 钉钉 | `dingtalk` | 插件，需配置 Client ID / Secret |
| Webhook | `webhook` | 插件，提供标准 HTTP 接收/发送接口 |

### 5.3 Webhook 频道规范

Webhook 是对外开放的通用接入方式，允许任意第三方系统接入：

**接收消息（外部 → GoPaw）：**
```
POST /webhook/{token}
Content-Type: application/json

{
  "user_id": "user123",
  "session_id": "session456",   // 可选，不填则按 user_id 创建
  "content": "帮我查一下天气",
  "msg_type": "text"            // text / image / file
}
```

**回调推送（GoPaw → 外部）：**
```yaml
# config.yaml 中配置回调地址
plugin:
  webhook:
    token: your-secret-token
    callback_url: https://your-server.com/ai-reply  # GoPaw 主动推送响应
    # 或者外部系统轮询：GET /webhook/{token}/poll
```

### 5.4 自定义频道扩展规范

第三方开发者实现自定义频道的步骤：

```
1. 实现 ChannelPlugin 接口
2. 编译为 Go plugin（.so）或独立进程（gRPC 接入）
3. 放置到 plugins/channels/ 目录
4. 在 config.yaml 中启用并配置
5. 重启 GoPaw 生效
```

---

## 6. Skills 系统

Skills 是扩展 Agent 能力的高层模块，采用**三层架构**，从零代码到完整代码，覆盖不同使用场景。

### 6.1 三层 Skill 体系

```
Level 1：Prompt Skill（零代码）
─────────────────────────────────────────────────────
  组成：manifest.yaml + prompt.md
  原理：把 prompt.md 内容注入系统提示，借助已有 Tools 完成任务
  适用：80% 的常见需求
  优点：你可以随时在 Skills Registry 更新，用户无感知升级

Level 2：Config Skill（低代码）
─────────────────────────────────────────────────────
  组成：manifest.yaml + prompt.md + workflow.yaml
  原理：用 YAML 编排多步骤工作流，组合已有 Tools
  适用：需要固定流程的复杂任务
  优点：不需要编程，非技术人员可维护

Level 3：Code Skill（完整代码）
─────────────────────────────────────────────────────
  组成：manifest.yaml + prompt.md + skill.go
  原理：注册专属 Tool，完整控制执行逻辑
  适用：需要特殊 API 调用、复杂状态管理的场景
  优点：完全灵活，无限制
```

### 6.2 Skill 目录结构

```
skills/
└── news/                           # Skill 名称
    ├── manifest.yaml               # 元信息（必须）
    ├── prompt.md                   # 系统提示注入（必须）
    ├── workflow.yaml               # 工作流定义（Level 2 可选）
    └── skill.go                    # 代码实现（Level 3 可选）
```

### 6.3 manifest.yaml 规范

```yaml
# skills/news/manifest.yaml

name: news                          # 唯一标识
version: 1.2.0
display_name: 资讯摘要
description: 聚合并摘要各类资讯，支持定时推送
author: GoPaw Team
level: 1                            # 1=Prompt / 2=Config / 3=Code

# 激活方式
activation:
  always: false                     # true=始终激活，false=按需激活
  keywords:                         # 触发关键词（用于提示用户）
    - 新闻
    - 资讯
    - 热点

# 依赖的 Tools（需要这些 Tool 存在才能工作）
requires_tools:
  - web_search
  - http_get

# 用户可配置项（在 Web Console 中展示配置表单）
config_schema:
  sources:
    type: array
    items: string
    default: ["hacker_news", "v2ex", "weibo_hot"]
    description: 资讯来源列表
    required: false
  language:
    type: string
    enum: ["zh", "en"]
    default: zh
    description: 输出语言
  daily_limit:
    type: integer
    default: 10
    min: 1
    max: 50
    description: 每次获取条数
```

### 6.4 prompt.md 规范

```markdown
<!-- skills/news/prompt.md -->
<!-- 此文件内容会被注入到系统提示中 -->

## 资讯摘要能力

你具备资讯聚合与摘要能力。当用户询问新闻、资讯、热点话题时：

1. 使用 `web_search` 搜索相关内容
2. 使用 `http_get` 获取页面正文（如有必要）
3. 提炼要点，按重要程度排序
4. 输出格式：标题 + 一句话摘要 + 来源链接

注意：
- 优先引用权威来源
- 标注信息时效性（如"截至今日"）
- 不捏造未经证实的内容
```

### 6.5 workflow.yaml 规范（Level 2）

```yaml
# skills/daily_report/workflow.yaml

steps:
  - id: fetch_news
    tool: web_search
    input:
      query: "今日 AI 行业动态"
      limit: "${config.daily_limit}"

  - id: fetch_weather
    tool: web_search
    input:
      query: "今日天气 ${user.city}"

  - id: summarize
    llm: true
    prompt: |
      基于以下信息生成今日日报：
      新闻：${fetch_news.result}
      天气：${fetch_weather.result}
      格式：先天气，后资讯，语气轻松

  - id: send
    tool: channel_send
    input:
      content: "${summarize.result}"
      channel: "${trigger.channel}"
```

### 6.6 内置 Skills 列表

| Skill | Level | 说明 |
|-------|-------|------|
| `pdf` | 3 | PDF 文件读取与摘要 |
| `docx` | 3 | Word 文档处理 |
| `xlsx` | 3 | Excel 数据读取与分析 |
| `news` | 1 | 资讯聚合摘要 |
| `web_reader` | 1 | 网页内容提取与摘要 |
| `daily_report` | 2 | 定时日报（天气+资讯） |
| `reminder` | 2 | 定时提醒 |

---

## 7. Tools 系统

Tools 是 Agent 可调用的原子操作，纯代码实现，LLM 按 Schema 调用。

### 7.1 Tool 接口

```go
// pkg/plugin/tool.go

type Tool interface {
    // 工具名称（唯一，snake_case）
    Name() string

    // 描述（LLM 读取，决定何时调用）
    Description() string

    // 参数 Schema（OpenAI function calling 格式）
    Parameters() ToolParameters

    // 执行
    Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

type ToolParameters struct {
    Type       string                     `json:"type"`       // "object"
    Properties map[string]ToolProperty    `json:"properties"`
    Required   []string                   `json:"required"`
}
```

### 7.2 内置 Tools

| Tool | 说明 | 参数 |
|------|------|------|
| `file_read` | 读取文件内容 | `path: string` |
| `file_write` | 写入文件内容 | `path: string, content: string` |
| `file_search` | 搜索文件 | `dir: string, pattern: string` |
| `shell_execute` | 执行 Shell 命令 | `command: string, timeout: int` |
| `web_search` | 网络搜索（Tavily/Serper） | `query: string, limit: int` |
| `http_get` | HTTP GET 请求 | `url: string, headers: object` |
| `http_post` | HTTP POST 请求 | `url: string, body: object` |

> **注：** `shell_execute` 在 v0.1 中不做沙箱，v0.3 引入白名单机制。

### 7.3 注册方式

所有内置 Tool 在程序启动时通过 `init()` 自动注册，无需配置：

```go
// plugins/tools/web_search.go
func init() {
    tool.Register(&WebSearchTool{})
}
```

Skills 专属 Tool（Level 3）在 Skill 初始化时注册，Skill 卸载时自动注销。

---

## 8. 记忆系统

### 8.1 设计思路

```
短期记忆（对话上下文）
  → 内存存储，滑动窗口
  → 超过 token 限制时触发压缩
  → 每个 SessionID 独立

长期记忆（持久化存储）
  → SQLite FTS5 全文搜索
  → 存储关键对话摘要和用户偏好
  → 定期压缩归档

不做向量搜索（v1.0）
  → SQLite FTS5 足以覆盖大多数检索需求
  → 向量搜索作为可选插件后续加入
```

### 8.2 接口定义

```go
// internal/memory/manager.go

type Manager interface {
    // 添加一轮对话
    Add(sessionID, userMsg, assistantMsg string) error

    // 获取最近 N 条对话（用于构建上下文）
    GetContext(sessionID string, limit int) ([]Message, error)

    // 全文搜索历史（用于长期记忆检索）
    Search(sessionID, query string, limit int) ([]MemorySnippet, error)

    // 压缩上下文（当 token 超限时调用）
    Compress(ctx context.Context, sessionID string) error

    // 清空会话记忆
    Clear(sessionID string) error
}
```

### 8.3 压缩策略

```
触发条件：当前 Session 的消息 token 总数 > 阈值（默认 4000 token）

压缩流程：
  1. 取最旧的 50% 消息
  2. 调用 LLM 生成摘要（100 字以内）
  3. 摘要替换原始消息，写入 SQLite
  4. 在上下文头部插入：[以下是历史摘要：xxx]
  5. 保留最近 50% 消息原文
```

---

## 9. 调度系统

### 9.1 功能概述

定时任务（Heartbeat）是 GoPaw 的核心特性之一，允许在指定时间自动触发 Agent 执行预设任务。

### 9.2 数据模型

```go
type CronJob struct {
    ID          string    // UUID
    Name        string    // 任务名称
    Description string    // 人类可读描述
    CronExpr    string    // Cron 表达式，如 "0 8 * * *"
    Channel     string    // 推送到哪个频道
    SessionID   string    // 使用哪个会话上下文
    Prompt      string    // 触发时发给 Agent 的指令
    Enabled     bool
    ActiveFrom  string    // 活跃时间窗口开始，如 "08:00"
    ActiveUntil string    // 活跃时间窗口结束，如 "22:00"
    LastRun     time.Time
    NextRun     time.Time
    CreatedAt   time.Time
}
```

### 9.3 示例

```yaml
# 每日早报任务
- name: morning_briefing
  cron: "0 8 * * 1-5"       # 周一到周五早上 8 点
  channel: feishu
  prompt: "生成今日早报，包括：今日天气、AI 圈热点 3 条、今日待办事项"
  active_from: "07:00"
  active_until: "09:00"
```

---

## 10. 插件规范

### 10.1 插件类型

| 类型 | 接口 | 说明 |
|------|------|------|
| Channel Plugin | `ChannelPlugin` | 消息频道接入 |
| Tool Plugin | `Tool` | Agent 可调用工具 |
| Skill Plugin | Skill 目录规范 | Agent 能力扩展 |

### 10.2 插件加载机制

GoPaw 采用**接口注册 + 配置驱动**的方式加载插件：

```
启动流程：
  1. 程序启动，内置插件通过 init() 自动注册到 Registry
  2. 读取 config.yaml 的 plugins.enabled 列表
  3. 只有在列表中的插件才会执行 Init() 和 Start()
  4. 未启用的插件注册了也不初始化，零资源占用
```

这个设计的好处：**无需动态加载 .so 文件，避免 Go plugin 的平台兼容问题，同时保持按需启用的效果。**

### 10.3 插件开发规范

第三方开发者开发频道插件的完整规范：

**Step 1：实现接口**
```go
package myplugin

import (
    "github.com/gopaw/gopaw/pkg/plugin"
)

type MyChannelPlugin struct {
    config *Config
    // ...
}

func (p *MyChannelPlugin) Name() string { return "my_channel" }
// ... 实现所有接口方法
```

**Step 2：注册插件**
```go
func init() {
    channel.Register(&MyChannelPlugin{})
}
```

**Step 3：配置启用**
```yaml
# config.yaml
plugins:
  enabled:
    - my_channel
  my_channel:
    api_key: xxx
    webhook_url: xxx
```

### 10.4 Webhook 插件作为通用扩展桥梁

对于不想用 Go 开发的场景，Webhook 频道可以作为外部系统接入的标准桥梁：

```
外部系统（Python/Node.js/任意语言）
    │
    │  POST /webhook/{token}
    ▼
  GoPaw Webhook Channel
    │
    ▼
  Agent 处理
    │
    │  POST callback_url
    ▼
外部系统收到响应
```

---

## 11. 数据层设计

### 11.1 存储方案

| 数据类型 | 存储方式 | 说明 |
|---------|---------|------|
| 对话历史 | SQLite | 主要持久化存储 |
| 定时任务 | SQLite | 任务定义和执行记录 |
| 配置数据 | config.yaml | 文件驱动，热加载 |
| 短期上下文 | 内存 | 重启后从 SQLite 恢复 |
| 会话状态 | 内存 + SQLite | 内存加速，SQLite 持久化 |

### 11.2 SQLite Schema

```sql
-- 会话表
CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    channel     TEXT NOT NULL,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
CREATE INDEX idx_sessions_user ON sessions(user_id);

-- 消息表
CREATE TABLE messages (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES sessions(id),
    role        TEXT NOT NULL CHECK(role IN ('user','assistant','system')),
    content     TEXT NOT NULL,
    token_count INTEGER DEFAULT 0,
    created_at  INTEGER NOT NULL
);
CREATE INDEX idx_messages_session ON messages(session_id);

-- FTS5 全文搜索（长期记忆）
CREATE VIRTUAL TABLE messages_fts USING fts5(
    content,
    session_id UNINDEXED,
    content='messages',
    content_rowid='rowid'
);

-- 定时任务表
CREATE TABLE cron_jobs (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    description  TEXT,
    cron_expr    TEXT NOT NULL,
    channel      TEXT NOT NULL,
    session_id   TEXT,
    prompt       TEXT NOT NULL,
    enabled      INTEGER DEFAULT 1,
    active_from  TEXT,
    active_until TEXT,
    last_run     INTEGER,
    next_run     INTEGER,
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL
);

-- 记忆摘要表
CREATE TABLE memory_summaries (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL,
    summary     TEXT NOT NULL,
    msg_from    INTEGER,    -- 摘要覆盖的消息起始 rowid
    msg_to      INTEGER,    -- 摘要覆盖的消息结束 rowid
    created_at  INTEGER NOT NULL
);
```

---

## 12. API 设计

### 12.1 REST API 概览

| 分组 | 路径前缀 | 说明 |
|------|---------|------|
| Agent | `/api/agent` | 消息发送、会话管理 |
| Config | `/api/config` | 配置读写 |
| Skills | `/api/skills` | 技能管理 |
| Channels | `/api/channels` | 频道管理 |
| Cron | `/api/cron` | 定时任务 |
| System | `/api/system` | 健康检查、版本信息 |

### 12.2 核心接口

**发送消息：**
```
POST /api/agent/chat
{
  "session_id": "session_001",   // 不传则自动创建
  "content": "帮我搜索今天的 AI 新闻",
  "msg_type": "text"
}

Response:
{
  "session_id": "session_001",
  "content": "好的，以下是今天的 AI 新闻...",
  "msg_type": "text",
  "elapsed_ms": 2341
}
```

**流式输出（SSE）：**
```
GET /api/agent/chat/stream?session_id=xxx&content=xxx

data: {"delta": "好的，"}
data: {"delta": "以下是"}
data: {"delta": "今天的 AI 新闻..."}
data: {"done": true}
```

**WebSocket 实时双向通信：**
```
WS /ws?session_id=xxx

// 客户端发送
{"type": "message", "content": "你好"}

// 服务端推送
{"type": "delta", "content": "你好！"}
{"type": "done"}
```

### 12.3 Webhook 接口

```
# 接收外部消息
POST /webhook/{token}
{
  "user_id": "user123",
  "content": "查天气",
  "msg_type": "text"
}

# 查询响应（轮询模式）
GET /webhook/{token}/messages?since=1234567890

# 健康检查
GET /webhook/{token}/health
```

---

## 13. 配置管理

### 13.1 完整配置示例

```yaml
# config.yaml

# 基础设置
app:
  name: "我的AI助手"          # Agent 显示名称
  language: zh
  timezone: Asia/Shanghai
  debug: false

# 服务器
server:
  host: 0.0.0.0
  port: 8088

# 数据存储
storage:
  type: sqlite
  path: data/gopaw.db

# LLM 配置
llm:
  # 使用 OpenAI 兼容接口
  provider: openai_compatible
  base_url: https://api.openai.com/v1
  api_key: ${OPENAI_API_KEY}
  model: gpt-4o-mini
  timeout: 60
  max_tokens: 4096

  # 或自定义接口（实现适配器）
  # provider: custom
  # endpoint: https://your-llm-api.com/chat
  # api_key: ${CUSTOM_API_KEY}
  # request_template: ...
  # response_path: "data.choices[0].message.content"

# Agent 配置
agent:
  system_prompt: |
    你是一个智能助理，名字叫 ${app.name}。
    你会帮助用户完成各种任务，回答问题，处理文件。
    请用中文回复，语气友好自然。
  max_steps: 20
  memory:
    context_limit: 4000      # token 数超过此值触发压缩
    history_limit: 50        # 最多保留最近 50 条消息

# 插件启用列表（只有列出的插件才会启动）
plugins:
  enabled:
    - console      # Web Console 频道（通常始终启用）
    - feishu       # 飞书
    # - dingtalk   # 钉钉（需要时取消注释）
    # - webhook    # Webhook 接入

# 各插件独立配置
plugin:
  feishu:
    app_id: ${FEISHU_APP_ID}
    app_secret: ${FEISHU_APP_SECRET}
    verification_token: ${FEISHU_VERIFICATION_TOKEN}
    encrypt_key: ${FEISHU_ENCRYPT_KEY}

  dingtalk:
    client_id: ${DINGTALK_CLIENT_ID}
    client_secret: ${DINGTALK_CLIENT_SECRET}

  webhook:
    token: ${WEBHOOK_TOKEN}
    callback_url: ""         # 留空则不主动推送，外部轮询

# Skills 配置
skills:
  dir: skills/               # 自定义 Skill 目录
  enabled:                   # 启用的 Skill 列表（不填则全部启用）
    - news
    - pdf
    - docx

  # 各 Skill 的用户配置
  config:
    news:
      sources: [hacker_news, v2ex]
      language: zh

# 日志
log:
  level: info                # debug / info / warn / error
  format: json
  output: stdout             # stdout / file
  file: logs/gopaw.log       # 当 output=file 时生效
```

### 13.2 环境变量

所有敏感信息通过环境变量注入，配置文件中使用 `${ENV_VAR}` 引用：

```bash
# .env 文件（不提交到 git）
OPENAI_API_KEY=sk-xxx
FEISHU_APP_ID=cli_xxx
FEISHU_APP_SECRET=xxx
FEISHU_VERIFICATION_TOKEN=xxx
DINGTALK_CLIENT_ID=xxx
DINGTALK_CLIENT_SECRET=xxx
WEBHOOK_TOKEN=your-random-token
```

### 13.3 配置热加载

修改 `config.yaml` 后无需重启，GoPaw 自动检测变更并重载非关键配置：

| 配置项 | 是否热加载 |
|-------|----------|
| `agent.system_prompt` | ✅ |
| `llm.model` / `llm.api_key` | ✅ |
| `skills.config.*` | ✅ |
| `plugins.enabled` | ❌ 需重启 |
| `server.port` | ❌ 需重启 |

---

## 14. Web Console

### 14.1 页面结构

```
/                → 重定向到 /chat
/chat            → 主聊天界面
/chat/:sessionId → 指定会话
/settings        → 设置（模型、基础配置）
/channels        → 频道管理
/skills          → 技能管理
/cron            → 定时任务管理
/logs            → 运行日志（简单展示）
```

### 14.2 各页面功能

**聊天界面（/chat）**
- 左侧：会话列表 + 新建会话
- 右侧：聊天窗口（Markdown 渲染、文件上传、流式输出）
- 支持清空会话、查看 Token 使用情况

**设置页（/settings）**
- LLM 提供商选择和 API Key 配置
- Agent 名称和基础 Prompt 编辑
- 记忆系统参数调整

**频道管理（/channels）**
- 各频道的启用/禁用开关
- 频道配置表单（根据插件 manifest 动态生成）
- 频道连接状态展示

**技能管理（/skills）**
- 已安装技能列表（启用/禁用/配置）
- 本地技能目录内容展示
- 预留：连接 Skills Registry 的入口

**定时任务（/cron）**
- 任务列表（名称、Cron 表达式、下次执行时间、状态）
- 新增/编辑/删除任务
- 手动触发执行
- 执行历史记录

### 14.3 前端技术栈

| 组件 | 选型 |
|------|------|
| 框架 | React 18 + TypeScript |
| 构建 | Vite |
| UI 库 | Ant Design v5 |
| 状态管理 | Zustand（轻量） |
| 网络请求 | Axios |
| WebSocket | 原生 API |
| Markdown | react-markdown + highlight.js |
| 国际化 | i18next（初期只做中文） |

前端构建产物通过 Go embed 嵌入二进制，无需额外的静态文件服务。

---

## 15. 部署方案

### 15.1 Docker 部署（推荐）

```dockerfile
# Dockerfile（多阶段构建）

# Stage 1：构建前端
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# Stage 2：构建 Go 后端
FROM golang:1.22-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN go build -o gopaw ./cmd/gopaw

# Stage 3：最终镜像
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=backend /app/gopaw .

VOLUME ["/app/data", "/app/skills", "/app/logs"]
EXPOSE 8088
CMD ["./gopaw", "--config", "/app/config.yaml"]
```

```yaml
# docker-compose.yml
services:
  gopaw:
    image: gopaw:latest
    ports:
      - "8088:8088"
    volumes:
      - ./config.yaml:/app/config.yaml
      - ./data:/app/data           # SQLite 数据库
      - ./skills:/app/skills       # 自定义 Skills
      - ./logs:/app/logs
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - FEISHU_APP_ID=${FEISHU_APP_ID}
      - FEISHU_APP_SECRET=${FEISHU_APP_SECRET}
    restart: unless-stopped
```

**快速启动：**
```bash
# 1. 复制配置模板
cp config.yaml.example config.yaml

# 2. 编辑配置，填入 API Key
vim config.yaml

# 3. 启动
docker compose up -d

# 4. 访问
open http://localhost:8088
```

### 15.2 预期资源占用

| 场景 | 内存 | 镜像大小 |
|------|------|---------|
| 核心 + Web Console | ~50MB | - |
| + 飞书插件 | ~70MB | - |
| + 钉钉插件 | ~90MB | - |
| + 常用 Skills | ~100MB | - |
| Docker 镜像 | - | ~50MB |

对比 CoPaw：内存减少约 **93%**，镜像减少约 **97%**。

---

## 16. 开发路线图

### v0.1 MVP（目标：基础能用，开发者可跑通）

```
核心引擎
  ✅ Go 项目骨架 + CLI（init / start / stop）
  ✅ 配置加载（YAML + 环境变量 + 热加载）
  ✅ SQLite 存储初始化
  ✅ 结构化日志（Zap）

Agent
  ✅ ReAct 引擎实现
  ✅ LLM Client（OpenAI 兼容 + 自定义接口）
  ✅ 基础记忆系统（滑动窗口 + SQLite 持久化）
  ✅ Tools Registry

频道
  ✅ Web Console 频道（内置）
  ✅ 飞书频道插件
  ✅ Webhook 频道插件

Tools（内置）
  ✅ file_read / file_write / file_search
  ✅ shell_execute
  ✅ web_search（对接 Tavily API）
  ✅ http_get / http_post

Web Console
  ✅ 聊天界面（流式输出）
  ✅ 基础设置页（模型配置）

部署
  ✅ Dockerfile + docker-compose.yml
  ✅ 单二进制构建

里程碑：用飞书 / Web Console 和 AI 完成一次有工具调用的对话
```

### v0.2（目标：普通用户可上手）

```
  - 钉钉频道插件
  - Skills 三层体系实现（Level 1/2/3 加载器）
  - 内置 Skills：pdf / docx / xlsx / news
  - Web Console：频道管理页 + Skills 管理页
  - 定时任务系统（Cron）+ Web Console 任务管理页
  - 上下文压缩优化
  - 记忆全文搜索（FTS5）

里程碑：非技术用户能独立配置并日常使用
```

### v0.3（目标：开源生态建设）

```
  - 插件开发文档和 SDK 完善
  - 自定义频道插件规范发布
  - Skills 目录标准化（方便社区贡献）
  - GitHub Actions CI/CD
  - 完整测试覆盖（>70%）
  - 性能基准测试

里程碑：有外部开发者能独立开发插件
```

### v0.4（目标：付费服务基础）

```
  - Skills Registry 服务上线
  - Web Console 内置技能商店入口
  - 一键安装 + 版本管理
  - 静默热更新机制
  - Persona 预设系统

里程碑：付费模式冷启动
```

### v0.5+（目标：商业化完善）

```
  - 重型 Skill 远程执行服务（浏览器/OCR）
  - 使用统计与分析面板
  - MCP 客户端支持
  - 向量记忆插件（可选）
  - Telegram 频道插件

里程碑：对外正式推广
```

---

## 17. 商业模式

### 17.1 Open Core 模型

```
开源（Apache 2.0）                付费（GoPaw Cloud）
────────────────────              ────────────────────
GoPaw Core 引擎                   官方 Skills Registry
频道插件系统（接口规范）            官方精选 Skills（质量保证）
基础工具集                        Skills 静默热更新
Web Console                       Persona 预设包
本地 Skills 加载                   远程执行服务（浏览器/OCR）
插件开发 SDK                       使用统计面板
社区 Skills（GitHub）              优先技术支持
```

### 17.2 付费版核心价值

| 痛点 | 开源版 | 付费版 |
|------|-------|-------|
| 想用好 Skills 但不会写 | 自己找、自己写 | 官方商店一键安装 |
| Skills 有 bug 要手动更新 | 手动操作 | 静默自动更新 |
| 想快速给团队配好 | 一个个配置 | Persona 一键开始 |
| 需要浏览器功能但服务器太重 | 本地装 Chromium | 云端执行，本地轻量 |

### 17.3 License 选择

采用 **Apache 2.0**：
- 开发者社区接受度高
- 允许企业集成使用
- 核心竞争力在 Cloud 层，开源越传播越好
- 无需担心竞品：没有 Skills Registry 和持续维护就无法提供同等体验

---

## 附录 A：技术选型汇总

| 组件 | 选型 | 理由 |
|------|------|------|
| 语言 | Go 1.22+ | 性能、并发、单二进制部署 |
| Web 框架 | Gin | 生态成熟、性能好、中间件丰富 |
| 数据库 | SQLite (modernc) | 零依赖、单文件、FTS5 支持 |
| 配置 | Viper | 多格式、热加载、环境变量覆盖 |
| 日志 | Zap | 高性能结构化日志 |
| 定时任务 | robfig/cron | 轻量级 Cron 表达式解析 |
| 嵌入前端 | Go embed | 单二进制内置静态文件 |
| 前端框架 | React 18 + TypeScript | 成熟稳定 |
| 前端 UI | Ant Design v5 | 组件完整，admin 类界面首选 |
| 前端构建 | Vite | 快速 |

## 附录 B：不做的事（边界）

```
v1.0 明确不做：
  ✗ 多租户 / 复杂权限系统
  ✗ 本地模型（llama.cpp / MLX）
  ✗ 桌面客户端
  ✗ MCP 服务端（仅 MCP 客户端延后做）
  ✗ 向量数据库
  ✗ 浏览器自动化（作为可选插件延后）
  ✗ iMessage 频道
  ✗ Tool 沙箱机制（延后做）
  ✗ Skills 代码签名验证（延后做）
```

---

*文档由 GoPaw 团队维护，随项目迭代持续更新。*
