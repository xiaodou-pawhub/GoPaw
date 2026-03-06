# GoPaw Skills 开发指南

GoPaw Skills（技能）是一种轻量的能力扩展机制：通过向 Agent 的系统提示注入特定指令，让 Agent 在处理特定类型请求时表现出专业化行为。

---

## 目录

1. [核心概念](#核心概念)
2. [技能目录](#技能目录)
3. [三个实现层级](#三个实现层级)
4. [manifest.yaml 完整参考](#manifestyaml-完整参考)
5. [prompt.md 编写指南](#promptmd-编写指南)
6. [激活机制详解](#激活机制详解)
7. [用户控制方式](#用户控制方式)
8. [config.yaml 配置](#configyaml-配置)
9. [最佳实践](#最佳实践)
10. [完整示例：translator](#完整示例translator)

---

## 核心概念

**技能 = manifest.yaml + prompt.md（+ 可选代码）**

每次用户发送消息时，系统根据激活规则判断哪些已启用的技能需要参与本次请求，将符合条件的技能的 `prompt.md` 内容拼接追加到系统提示末尾，然后再调用 LLM。Agent 看到这段额外指令后，就会按照技能定义的方式处理请求。

**关键特性：**
- 无入侵：不修改 Agent 核心逻辑，仅通过提示注入影响行为
- 按需激活：通过关键词匹配控制注入时机，避免无关提示干扰
- 运行时控制：用户可随时通过 Web UI 启用或禁用技能，无需重启

---

## 技能目录

所有技能存放在用户家目录的 `~/.gopaw/skills/` 下，每个技能占一个独立子目录：

```
~/.gopaw/skills/
├── translator/          # 翻译技能（本仓库的参考示例）
│   ├── manifest.yaml    # 必需：技能元数据与激活配置
│   └── prompt.md        # 必需（Level 1/2）：注入到系统提示的内容
├── my-skill/
│   ├── manifest.yaml
│   └── prompt.md
└── ...
```

**规则：**
- 目录名没有特殊要求，但建议与 `manifest.yaml` 中的 `name` 字段保持一致，便于维护
- 每个技能目录下至少包含 `manifest.yaml`
- Level-1 技能还需要 `prompt.md`
- 系统启动时自动扫描此目录，加载所有有效技能

> 本仓库 `skills/` 目录是参考示例，展示规范的技能结构。实际使用时请将技能放到 `~/.gopaw/skills/`。

---

## 三个实现层级

### Level 1 — Prompt Skill（推荐）

**文件**：`manifest.yaml` + `prompt.md`

纯提示注入，无需任何代码。适合绝大多数场景：角色扮演、格式规范、专业领域知识、行为约束等。

```
my-skill/
├── manifest.yaml   # level: 1
└── prompt.md       # 注入的提示内容
```

### Level 2 — Config Skill（规划中，v0.2 支持）

**文件**：`manifest.yaml` + `prompt.md` + `workflow.yaml`

通过 YAML 描述工具调用流程，无需编写 Go 代码即可实现带工具调用的技能。目前 `workflow.yaml` 会被识别但不执行，v0.2 版本正式支持。

### Level 3 — Code Skill

**文件**：`manifest.yaml` + Go 源码（编译进二进制）

适合需要注册自定义 Tool、与外部系统集成、或有复杂运行时逻辑的场景。
Level-3 技能的 Go 代码放在 `internal/skill/` 目录下，通过 `plugin.Skill` 接口实现，并在 `init()` 中注册。`prompt.md` 可选。

**大多数场景使用 Level-1 即可，Level-3 留给需要自定义工具的高级场景。**

---

## manifest.yaml 完整参考

```yaml
# ── 基本信息 ──────────────────────────────────────────────────────────────────

# [必填] 技能唯一标识符
#   - 英文小写，允许连字符（如 my-skill），不含空格
#   - 作为系统主键：Web UI、API、config.yaml enabled 列表均使用此值
#   - 发布后不应更改，否则已有的用户启用配置会失效
name: my-skill

# [建议填写] 语义化版本号，仅用于展示和日志
version: 1.0.0

# [建议填写] Web UI 中展示的友好名称，支持中文
display_name: 我的技能

# [建议填写] 一句话描述，显示在 Web UI 技能列表卡片上
description: 简要描述这个技能做什么。

# [可选] 作者信息，仅用于展示
author: Your Name

# [必填] 技能实现层级：1、2 或 3
#   1 = Prompt Skill  — manifest.yaml + prompt.md
#   2 = Config Skill  — 增加 workflow.yaml（v0.2 支持）
#   3 = Code Skill    — Go 代码实现，编译进二进制
level: 1

# ── 激活控制 ──────────────────────────────────────────────────────────────────

activation:
  # [必填] 是否始终注入
  #   true  → 每次请求都注入 prompt.md，不受关键词限制
  #   false → 仅在用户输入匹配下方关键词时才注入
  always: false

  # [当 always: false 时建议填写] 触发关键词列表
  #   - 大小写不敏感：系统将用户输入和关键词都转为小写后比较
  #   - 匹配方式：strings.Contains，即关键词是用户输入的子串即命中
  #   - 任意一个命中即触发，无需全部匹配
  #   - always: true 时此字段被忽略（但仍可作为文档说明保留）
  keywords:
    - 关键词一
    - keyword-two

# ── 工具依赖 ──────────────────────────────────────────────────────────────────

# [可选] 声明此技能依赖的内置工具名称
#   - 当前版本仅作文档说明，不影响运行时行为
#   - Level-1/2 技能无法直接调用工具（工具由 Agent 自主决定是否使用）
#   - Level-3 代码技能通过 Tools() 方法注册自定义工具，此字段填对应工具名
#   - 内置工具名称参见 internal/tool/builtin/ 目录
requires_tools: []
# 示例：
# requires_tools:
#   - web_search
#   - send_email

# ── 配置项定义 ────────────────────────────────────────────────────────────────

# [可选] 技能的用户可配置项（JSON Schema 子集）
#   - 当前版本此字段作为文档说明，未来 Web UI 将根据此生成配置表单
#   - Level-1 技能通常不需要配置，保持空 {} 即可
#   - Level-3 技能可在 Init(cfg map[string]interface{}) 中读取配置值
config_schema: {}
# 带配置的示例：
# config_schema:
#   target_language:
#     type: string
#     default: "zh"
#     description: "默认目标语言（zh/en）"
#   formality:
#     type: string
#     default: "auto"
#     description: "翻译语体：formal / casual / auto"
```

### manifest.yaml 字段速查

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | 是 | 唯一标识符，英文小写 |
| `version` | string | 否 | 语义化版本号 |
| `display_name` | string | 否 | Web UI 展示名称 |
| `description` | string | 否 | 一句话描述 |
| `author` | string | 否 | 作者信息 |
| `level` | int | 是 | 1 / 2 / 3 |
| `activation.always` | bool | 是 | 是否始终注入 |
| `activation.keywords` | []string | 否 | 触发关键词列表 |
| `requires_tools` | []string | 否 | 依赖工具声明 |
| `config_schema` | map | 否 | 用户配置项定义 |

---

## prompt.md 编写指南

`prompt.md` 的内容会被直接追加到 Agent 的系统提示末尾。Agent 会将其视为额外的行为指令。

### 结构建议

```markdown
# 技能标题（可选，帮助 LLM 理解上下文）

简要说明这段提示的用途（1-2 句话）。

## 核心行为规则

用清晰的列表或分点描述 Agent 应当怎么做。
避免模糊表述（"尽量"、"可能"），尽量使用明确指令。

## 输出格式（如需规定）

明确说明输出的结构、格式、长度要求。

## 边界条件（如需处理）

说明特殊情况下的处理方式。
```

### 编写原则

**1. 具体胜于模糊**

```markdown
# 差：
尽量给出准确的翻译。

# 好：
翻译时严格遵循：忠实原文含义 > 目标语言地道表达 > 保留原文文化特色。
不随意增减内容，不解释原文意思（除非用户明确要求）。
```

**2. 格式指令要有示例**

如果要求特定输出格式，直接给出模板比文字描述更有效：

```markdown
输出格式：

**原文**
{原文}

**译文**
{译文}
```

**3. 避免冲突与过度限制**

- 不要在技能提示中覆盖 Agent 的全局行为（如"永远不要使用工具"）
- 技能提示应聚焦在技能领域，让 Agent 在其他方面保持默认行为
- 多个技能同时激活时，提示会拼接在一起，避免相互矛盾的指令

**4. 长度控制**

- 单个技能的 `prompt.md` 建议控制在 500 字以内
- 过长的提示会占用宝贵的上下文窗口，且 LLM 对结尾内容的遵循度更低
- 优先级排序：重要规则放在文件开头

**5. Markdown 格式**

- 使用 `##` 标题组织结构，便于阅读和 LLM 识别段落意图
- 列表（`-` 或数字）比长段落更容易被 LLM 遵循
- 表格适合展示对照关系（如文本类型 vs 处理策略）

---

## 激活机制详解

每次用户消息到达时，系统按以下逻辑决定哪些技能参与本次请求：

```
对于每个已注册的技能：
  1. 如果 Enabled == false → 跳过（用户已禁用）
  2. 如果 Enabled == true：
     a. 如果 activation.always == true → 注入 prompt.md
     b. 如果 activation.always == false：
        对每个 keyword in activation.keywords：
          如果 strings.Contains(toLower(用户输入), toLower(keyword)) → 注入并 break

将所有匹配技能的 prompt.md 拼接（\n\n 分隔），追加到系统提示末尾
```

### always: true 适用场景

适合对 Agent 行为有全局要求的技能：
- 全局语言风格规范（如"始终使用简体中文回复"）
- 安全与合规约束
- 企业品牌/语气规范
- 始终需要的上下文增强（如"你是 XX 公司的客服助手"）

**注意**：`always: true` 的技能会消耗每次请求的 Token，请确保内容精简必要。

### always: false 适用场景

适合特定领域的专业增强：
- 翻译、代码审查、数据分析等专项能力
- 特定格式的输出规范
- 只在特定对话场景才有用的行为

### 关键词设计技巧

关键词采用 `strings.Contains` 匹配，因此：
- 短词比长词覆盖更广（"翻译"可匹配"翻译一下"、"请翻译"等）
- 避免过于宽泛的词（如"帮我"）导致误触发
- 覆盖常见的近义表达和英文写法
- 中文关键词无需分词，直接写词语即可

---

## 用户控制方式

用户有两种方式控制技能的启用/禁用：

### 方式一：Web UI（推荐）

访问 Web 管理界面 → `设置` → `技能`，可以看到所有已加载的技能列表，每个技能有独立的启用/禁用开关，实时生效，无需重启。

### 方式二：API

```bash
# 启用技能
curl -X PUT http://localhost:8088/api/skills/translator/enabled \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}'

# 禁用技能
curl -X PUT http://localhost:8088/api/skills/translator/enabled \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'

# 查看所有技能状态
curl http://localhost:8088/api/skills \
  -H "Authorization: Bearer <admin-token>"
```

**启用状态的优先级**：

```
用户 Enabled 开关（运行时）
    ↓
activation.always / keywords（决定本次请求是否注入）
```

只有 `Enabled == true` 且激活条件满足，技能才会参与当次请求。

---

## config.yaml 配置

在 GoPaw 的 `config.yaml` 中可以预配置技能相关选项：

```yaml
skills:
  # 技能目录路径（留空则默认使用 ~/.gopaw/skills/）
  dir: ""

  # 预设启用的技能列表（按 name 字段）
  # - 留空（不填）：启动时加载目录中的所有技能，全部默认启用
  # - 填写列表：只有列表中的技能会被启用，其余加载但禁用
  enabled:
    - translator
    # - my-other-skill
```

**说明**：
- `skills.dir` 为空时默认使用 `~/.gopaw/skills/`
- `skills.enabled` 只是初始状态，用户通过 Web UI 更改的启用状态会存入数据库并覆盖此配置
- 建议生产环境在 `config.yaml` 中声明期望的初始状态，日常调整通过 Web UI 操作

---

## 最佳实践

### 一个技能只做一件事

每个技能的 `prompt.md` 应聚焦单一职责。如果你的"翻译技能"同时想处理"代码审查"，请拆分为两个独立技能。多个技能可以同时激活，系统会自动组合。

### 关键词要有区分度

避免使用过于通用的关键词导致误触发。测试方法：用平时的日常聊天消息检验，看技能是否会被意外激活。

### always: false 是默认选择

绝大多数专业技能应使用 `always: false` + 关键词触发。`always: true` 只用于真正需要全局生效的规范，过多 `always: true` 技能会显著增加每次请求的 Token 消耗。

### prompt.md 要经过测试

编写完成后，实际向 GoPaw 发送几条典型请求，观察 Agent 的响应是否符合预期。如果 Agent 没有按技能指令行事，可能是：
- 提示太长或太模糊
- 关键词没有命中，技能未激活（查看日志确认）
- 与其他技能的提示产生冲突

### 版本管理

技能目录是普通文件，可以用 Git 管理：

```bash
cd ~/.gopaw/skills
git init
git add .
git commit -m "add translator skill"
```

---

## 完整示例：translator

本仓库 `skills/translator/` 是一个规范的 Level-1 技能示例，包含：

- **manifest.yaml**：完整注释的配置文件，展示所有字段的用法
- **prompt.md**：结构清晰、规则具体的提示，包含工作流程、特殊处理规则和自检清单

参考此示例创建新技能：

```bash
# 1. 在用户技能目录创建新技能
mkdir -p ~/.gopaw/skills/my-skill

# 2. 复制示例文件作为起点
cp skills/translator/manifest.yaml ~/.gopaw/skills/my-skill/
cp skills/translator/prompt.md ~/.gopaw/skills/my-skill/

# 3. 修改 manifest.yaml 中的 name、display_name、description、keywords
# 4. 重写 prompt.md 为新技能的专属指令

# 5. 重启 GoPaw 或在 Web UI 中刷新技能列表（技能在启动时加载）
```

> 当前版本技能在启动时加载。修改技能文件后需要重启 GoPaw 才能生效。
