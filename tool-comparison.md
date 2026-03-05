# GoPaw 工具系统重构分析报告

## 一、GoPaw 当前工具系统

### 1.1 工具架构

**核心接口** (`pkg/plugin/tool.go`):
```go
type Tool interface {
    Name() string
    Description() string
    Parameters() ToolParameters
    Execute(ctx context.Context, args map[string]interface{}) (string, error)
}
```

**注册与执行**:
- `internal/tool/registry.go`: 工具注册表
- `internal/tool/executor.go`: 工具执行器
- GoPaw 使用 `init()` 自动注册机制

### 1.2 当前内置工具 (10 个)

| 工具名称 | 文件 | 行数 | 功能 | 状态 |
|---------|------|------|------|------|
| `file_read` | `tools/file_read.go` | 59 | 读取文件 | ✅ 内置 |
| `file_write` | `tools/file_write.go` | 64 | 写入文件 | ✅ 内置 |
| `file_search` | `tools/file_search.go` | 66 | 文件名搜索 | ✅ 内置 |
| `shell_execute` | `tools/shell.go` | 94 | 执行 Shell 命令 | ✅ 内置 |
| `web_search` | `tools/web_search.go` | 131 | Web 搜索 (Tavily) | ✅ 内置 |
| `http_get` | `tools/http_get.go` | 94 | HTTP GET 请求 | ✅ 内置 |
| `http_post` | `tools/http_post.go` | ~80 | HTTP POST 请求 | ✅ 内置 |
| `memory_recall` | `tools/memory_recall.go` | 82 | 记忆搜索 | ✅ 内置 |
| `memory_store` | `tools/memory_store.go` | - | 记忆存储 | ✅ 内置 |
| `memory_read` | `tools/memory_read.go` | - | 读取记忆文件 | ✅ 内置 |
| `memory_write` | `tools/memory_write.go` | - | 写入记忆文件 | ✅ 内置 |
| `memory_forget` | `tools/memory_forget.go` | - | 删除记忆 | ✅ 内置 |
| `memory_note` | `tools/memory_note.go` | - | 记录 Daily Note | ✅ 内置 |
| `sub_agent` | `tools/sub_agent.go` | - | 创建子 Agent | ✅ 内置 |

**总计**: ~12 个工具

---

## 二、参考项目工具系统对比

### 2.1 PicoClaw 工具系统 (2000+ 行)

| 类别 | 工具数 | 说明 |
|------|--------|------|
| **核心接口** | 3 个 | Tool / ContextualTool / AsyncTool |
| **文件系统** | 5 个 | read/write/list/edit/append |
| **Web 工具** | 2 个 | web_search/web_fetch |
| **消息工具** | 1 个 | message |
| **技能工具** | 2 个 | find_skills/install_skill |
| **子代理** | 2 个 | spawn/subagent |
| **MCP 工具** | 1 个 | MCP Server |
| **硬件工具** | 2 个 | i2c/spi |
| **定时任务** | 1 个 | cron |
| **Shell** | 1 个 | exec |
| **总计** | **20+** | |

**可复用工具** (直接适配 GoPaw):
- ✅ `file_read` / `file_write` / `file_search` (已有)
- ✅ `shell_execute` (已有)
- ✅ `web_search` (已有，Tavily)
- ✅ `http_get` / `http_post` (已有)
- ✅ `web_fetch` (可添加)
- ✅ `message` (可添加 - 消息发送)
- ✅ `spawn` (可添加 - 子代理)
- ✅ `cron` (可添加 - 定时任务管理)

### 2.2 CoPaw 工具系统 (~3,630 行)

| 类别 | 工具数 | 说明 |
|------|--------|------|
| **内置工具** | 11 个 | 文件/Shell/记忆/浏览器等 |
| **技能工具** |~4 个 | PDF/DOCX/XLSX/PPTX |
| **MCP 工具** | 按需 | HTTP/本地进程 |
| **总计** | **15+** | |

**可复用工具** (直接适配 GoPaw):
- ✅ `read_file` (已有)
- ✅ `write_file` (已有)
- ✅ `grep_search` / `glob_search` (已有 file_search)
- ✅ `execute_shell_command` (已有)
- ✅ `memory_search` (已有 memory_recall)
- ✅ `send_file_to_user` (可添加)
- ✅ `browser_use` (高优先级 - 浏览器自动化)
- ✅ `desktop_screenshot` (中优先级 - 截图)

### 2.3 ZeroClaw 工具系统 (60+ 个)

| 类别 | 工具数 | 说明 |
|------|--------|------|
| **文件系统** | 6 个 | read/write/edit/glob_search/git |
| **Web 工具** | 4 个 | web_search/web_fetch/http_request/url_validation |
| **记忆工具** | 4 个 | store/recall/observe/forget |
| **MCP 工具** | 3 个 | mcp_tool/mcp_client/mcp_transport |
| **系统工具** | 4 个 | shell/process/schedule/bg_run |
| **硬件工具** | 3 个 | board_info/memory_map/memory_read |
| **文档工具** | 4 个 | docx_read/pdf_read/xlsx_read/pptx_read |
| **子 Agent** | 5 个 | spawn/list/manage/delegate/ipc |
| **定时任务** | 6 个 | cron_add/list/remove/run/update |
| **WASM** | 2 个 | wasm_tool/wasm_module |
| **其他** | ~20 个 | browser/screenshot/feishu_doc等 |
| **总计** | **60+** | |

**可复用工具** (高优先级):
- ✅ `read_file` / `write_file` / `edit_file` (已有基础版)
- ✅ `web_search` (已有)
- ✅ `memory_search` / `memory_store` (已有)
- ✅ `shell` / `process` (已有)
- ✅ `subagent_spawn` (可添加)
- ✅ `cron_add` / `cron_list` / `cron_remove` (可添加)
- ⚠️ `pdf_read` / `docx_read` / `xlsx_read` (文档处理 - 中优先级)
- ⚠️ `browser` (浏览器控制 - 高优先级，复杂度高)

### 2.4 OpenClaw 工具系统 (~24,000 行, 92 个文件)

**工具分类**:
| 类别 | 文件数 | 行数 | 说明 |
|------|-------|------|------|
| **浏览器工具** | 6 | 20,715 | Chrome 控制 (最复杂) |
| **消息工具** | 14 | ~5,000 | 跨会话/平台消息 |
| **平台工具** | 12 | ~5,000 | Discord/Slack/Telegram/WhatsApp |
| **Web 工具** | 12 | ~2,500 | 搜索/抓取 |
| **媒体工具** | 6 | ~4,000 | 图像/PDF |
| **系统工具** | 8 | ~3,000 | Cron/nodes/gateway |
| **核心工具** | 14 | ~2,000 | Common/agent-step |
| **其他工具** | 20 | ~2,000 | 子代理/Canvas/TTS等 |
| **总计** | **92** | **~24,000** | |

**可复用工具** (高优先级):
- ✅ `web_search` (已有)
- ✅ `memory_search` (已有)
- ⚠️ `browser` (需要 Playwright)
- ⚠️ `message` / `sessions_send` (消息系统)
- ⚠️ `subagents` (子 Agent 编排)
- ⚠️ `cron` (已有基础版)

---

## 三、工具复用优先级矩阵

### 3.1 直接可用 (零工作量)

| 工具名称 | PicoClaw | CoPaw | ZeroClaw | OpenClaw | GoPaw | 状态 |
|---------|---------|-------|---------|---------|------|------|
| `file_read` | ✅ | ✅ | ✅ | - | ✅ | 已有 |
| `file_write` | ✅ | ✅ | ✅ | - | ✅ | 已有 |
| `file_search` | ✅ | ✅ | ✅ | - | ✅ | 已有 |
| `shell_execute` | ✅ | ✅ | ✅ | - | ✅ | 已有 |
| `web_search` | ✅ | ✅ | ✅ | ✅ | ✅ | 已有 (Tavily) |
| `http_get` | ✅ | ✅ | ✅ | - | ✅ | 已有 |
| `http_post` | ✅ | ✅ | ✅ | - | ✅ | 已有 |
| `memory_recall` | ✅ | ✅ | ✅ | ✅ | ✅ | 已有 |

**结论**: ✅ **8 个工具已覆盖**

### 3.2 低工作量可添加 (1-2 天)

| 工具名称 | 功能 | 优先级 | 工作量 | 说明 |
|---------|------|--------|-------|------|
| `http_file_download` | 下载文件 | Medium | 低 | 类似 http_get |
| `memory_store` | 存储记忆 | Medium | 低 | 可复用 memory_recall 逻辑 |
| `memory_read` | 读取记忆 | Low | 低 | 简单文件读取 |
| `memory_write` | 写入记忆 | Low | 低 | 类似 file_write |
| `memory_forget` | 删除记忆 | Low | 低 | 删除记忆记录 |
| `memory_note` | Daily Note | Low | 低 | 创建每日笔记 |
| `sub_agent` | 子 Agent | High | 中 | 复用 spawning 逻辑 |
| `cron_list` | 列出定时任务 | Low | 低 | 列出 cron_jobs 表 |
| `cron_remove` | 删除定时任务 | Low | 低 | 删除 cron 记录 |

**预估**: 5-7 个工具，工作量 3-5 天

### 3.3 中工作量可添加 (3-7 天)

| 工具名称 | 功能 | 优先级 | 工作量 | 说明 |
|---------|------|--------|-------|------|
| `web_fetch` | 抓取网页 | High | 中 | 类似 web_search |
| `grep_search` | 内容搜索 | Medium | 低 | 使用 grep 命令 |
| `send_file_to_user` | 发送文件 | Medium | 中 | Channel 插件交互 |
| `web_search_multi` | 多搜索 | High | 中 | Brave/Tavily/Serper |
| `shell_timeout` | Shell 超时 | High | 低 | 增强 shell_execute |
| `http_post_with_files` | POST 文件 | Medium | 中 | 支持 multipart |
| `process_execute` | 进程管理 | Medium | 中 | exec.Command + 状态 |
| `cron_add` | 添加定时任务 | Medium | 中 | 写入 cron_jobs 表 |
| `cron_update` | 更新定时任务 | Medium | 中 | 更新 cron_jobs 表 |

**预估**: 9 个工具，工作量 7-14 天

### 3.4 高工作量可添加 (2-4 周)

| 工具名称 | 功能 | 优先级 | 工作量 | 说明 |
|---------|------|--------|-------|------|
| `browser_use` | 浏览器控制 | High | High | Playwright 集成 |
| `desktop_screenshot` | 桌面截图 | Medium | High | Playwright/屏幕捕获 |
| `pdf_read` | PDF 读取 | Medium | High | PDFLib/Poppler |
| `docx_read` | Word 读取 | Medium | High | docx 库 |
| `xlsx_read` | Excel 读取 | Medium | High | Excelize |
| `subagent_list` | 列出子 Agent | Medium | High | 管理会话树 |
| `subagent_kill` | 终止子 Agent | Medium | High | 会话树遍历 |
| `message_tool` | 消息工具 | Medium | High | 跨会话消息 |

**预估**: 8 个工具，工作量 2-3 个月

### 3.5 低优先级/可选 (v2.0+)

| 工具名称 | 功能 | 优先级 | 说明 |
|---------|------|--------|------|
| `mcp_tool` | MCP 工具 | Low | MCP 协议支持 |
| `i2c/spi` | 硬件工具 | Very Low | 仅树莓派等 |
| `git_operations` | Git 操作 | Low | 多项目支持 |
| `screenshot` | 截图 | Very Low | Playwright |
| `canvas` | Canvas | Very Low | 可视化画布 |
| ` tts` | TTS | Very Low | 语音合成 |
| `wasm_tool` | WASM | Very Low | 外部插件 |

---

## 四、推荐实现路线图

### Phase 1: 核心工具完善 (1-2 周) ⭐⭐⭐⭐⭐
```
目标: 完善基础工具集，覆盖 80% 使用场景

✅ 已完成:
- file_read / file_write / file_search
- shell_execute / http_get / http_post
- web_search (Tavily)
- memory recall/store/read/write/forget/note
- sub_agent

高优先级添加:
1. web_fetch (抓取网页) - 2 天
2. grep_search (内容搜索) - 1 天
3. send_file_to_user (发送文件) - 2 天
4. cron_add / cron_list / cron_remove - 3 天
5. process_execute (进程管理) - 3 天
```

### Phase 2: 浏览器自动化 (2-3 周) ⭐⭐⭐⭐
```
目标: 支持网页交互和自动化

1. browser_use (Chrome 控制) - 2 周
   - 基于 Playwright
   - 页面导航/点击/输入/截图
   - 多页面支持
2. desktop_screenshot (截图) - 2 天
3. browser_navigate (导航) - 1 天
```

### Phase 3: 文档处理 (2-3 周) ⭐⭐⭐
```
目标: 支持 Office/PDF 文档

1. pdf_read (PDF 读取) - 1 周
   - PDFText/Poppler 集成
   - 文本提取/页码提取
2. docx_read (Word 读取) - 5 天
   - docx 库
   - 文本/表格提取
3. xlsx_read (Excel 读取) - 5 天
   - Excelize
   - 表格数据提取
4. subagent_tools (子 Agent 管理) - 1 周
   - list/kill/steer
```

### Phase 4: MCP/其他 (v2.0+) ⭐⭐
```
1. MCP 工具集成
2. WASM 工具支持
3. 硬件工具 (i2c/spi)
4. 文档工具 (PPTX)
```

---

## 五、工具接口优化建议

### 5.1 借鉴 PicoClaw 的 ToolResult 设计

**当前问题**: GoPaw 工具返回 `string`, 错误通过 `error`

**建议**: 引入结构化结果类型

```go
// pkg/plugin/tool_result.go
type ToolResult struct {
    ForLLM  string `json:"for_llm"`   // 给 LLM 的内容
    ForUser string `json:"for_user"`  // 给用户的内容
    Silent  bool   `json:"silent"`    // 静默模式
    IsError bool   `json:"is_error"`  // 错误标记
    Async   bool   `json:"async"`     // 异步标记
    Media   []string `json:"media"`   // 媒体引用
    Err     error  `json:"-"`         // 错误对象
}

// 工厂函数
func NewResult(content string) *ToolResult
func SilentResult(content string) *ToolResult
func ErrorResult(message string) *ToolResult
func UserResult(content string) *ToolResult
```

**优势**:
- ✅ 语义明确分离 LLM/用户内容
- ✅ 支持异步工具
- ✅ 支持媒体文件
- ✅ 错误处理更清晰

### 5.2 借鉴 PicoClaw 的 ContextualTool

**当前**: 工具无法获取 channel/chatID 上下文

**建议**: 添加可选接口

```go
type ContextualTool interface {
    Tool
    SetContext(channel, chatID string)  // 设置上下文
}

// 使用
func (t *MyTool) SetContext(channel, chatID string) {
    t.channel = channel
    t.chatID = chatID
}
```

**应用场景**:
- 消息工具发送到正确渠道
- 记忆记录关联会话
- 日志记录上下文信息

### 5.3 借鉴 PicoClaw 的 AsyncTool

**当前**: 所有工具同步执行

**建议**: 添加异步支持

```go
type AsyncTool interface {
    Tool
    SetCallback(cb AsyncCallback)
}

type AsyncCallback func(ctx context.Context, result *ToolResult)

// 使用
func (t *SpawnTool) Execute(ctx context.Context, args map[string]interface{}) *ToolResult {
    go t.runSubagent(ctx, args)
    return AsyncResult("Subagent spawned, will report back")
}
```

**应用场景**:
- 子 Agent spawn (不阻塞主循环)
- 长时间运行的 Web 请求
- 后台任务报告

### 5.4 工具执行器优化

**当前**: `internal/tool/executor.go`

**建议**: 增强执行器

```go
type Executor struct {
    registry *Registry
    logger   *zap.Logger
    // 新增字段
    maxOutputRunes int
    sandbox        bool  // 沙箱模式
}

func (e *Executor) ExecuteWithContext(ctx context.Context, name, argsJSON, channel, chatID string) (*ToolResult, error) {
    // 实现 ContextualTool 和 AsyncTool 支持
}

func (e *Executor) ExecuteAsync(ctx context.Context, name, argsJSON string, callback AsyncCallback) (*ToolResult, error) {
    // 异步执行工具
}
```

---

## 六、工具安全建议

### 6.1 借鉴 OpenClaw 的 Exec Approvals

**问题**: shell_execute 可能执行危险命令

**建议**: 添加审批系统

```go
// internal/tool/approvals.go
type ApprovalConfig struct {
    Security string   // "deny" | "allowlist" | "full"
    Allowlist []string // Glob patterns
}

func (e *Executor) Execute(ctx context.Context, name, argsJSON string) (string, error) {
    // 1. 检查工具是否需要审批
    if requiresApproval(name) {
        if !approved(argsJSON) {
            return "", fmt.Errorf("tool %s requires approval", name)
        }
    }
    // 2. 执行工具 (沙箱/直接)
}
```

### 6.2 工具访问控制

**建议**: 添加权限控制

```go
type Permissions struct {
    FileReadPaths   []string // 允许读取的路径
    FileWritePaths  []string // 允许写入的路径
    ShellCommands   []string // 允许的 Shell 命令
    NetworkHosts    []string // 允许访问的网络
}

func (e *Executor) Execute(ctx context.Context, name, argsJSON string) (string, error) {
    // 检查权限
    if !e.canAccess(tool, args) {
        return "", fmt.Errorf("permission denied")
    }
}
```

---

## 七、结论与优先级总结

### 7.1 当前优势

✅ **核心工具齐全**: 文件操作、Shell、Web、记忆系统
✅ **架构清晰**: Registry + Executor 分离
✅ **扩展性好**: 插件化设计
✅ **安全意识**: 文件读取限制、超时控制

### 7.2 短期优化 (1-2 周)

**P0 - 必须完成**:
1. web_fetch (网页抓取) - ⭐⭐⭐⭐⭐
2. grep_search (内容搜索) - ⭐⭐⭐⭐
3. cron_add/list/remove (定时任务管理) - ⭐⭐⭐⭐
4. send_file_to_user (发送文件) - ⭐⭐⭐⭐

**P1 - 重要**:
5. process_execute (进程管理) - ⭐⭐⭐⭐
6. http_file_download (文件下载) - ⭐⭐⭐
7. subagent_list/kill (子 Agent 管理) - ⭐⭐⭐

### 7.3 中期优化 (1-2 月)

**P2 - 增强功能**:
1. browser_use (浏览器自动化) - ⭐⭐⭐⭐⭐ (复杂度高)
2. pdf/docx/xlsx read (文档处理) - ⭐⭐⭐⭐

**P3 - 扩展接口**:
3. ToolResult 结构化 - ⭐⭐⭐⭐
4. ContextualTool 支持 - ⭐⭐⭐
5. AsyncTool 支持 - ⭐⭐⭐
6. Exec Approvals 安全系统 - ⭐⭐⭐⭐⭐

### 7.4 长期优化 (v2.0+)

**P4 - 高级特性**:
1. MCP 工具集成
2. WASM 工具支持
3. 硬件工具 (i2c/spi)
4. 文档工具 (PPTX)

---

## 八、行动建议

### Stage 1: 立即执行 (本周)
```
[ ] 1. 添加 web_fetch 工具
[ ] 2. 添加 grep_search 工具
[ ] 3. 完善 cron 工具 (add/list/remove)
[ ] 4. 添加 send_file_to_user 工具
```

### Stage 2: 两周内完成
```
[ ] 5. 添加 process_execute 工具
[ ] 6. 重构工具接口 (ToolResult + ContextualTool)
[ ] 7. 增强 Executor (异步支持)
[ ] 8. 基础安全系统 (执行审批)
```

### Stage 3: 一个月内完成
```
[ ] 9. 浏览器工具 (browser_use)
[ ] 10. 文档工具 (pdf/docx/xlsx read)
[ ] 11. 子 Agent 管理工具
[ ] 12. 完整测试覆盖率 (>70%)
```

---

## 九、工具统计汇总表

| 项目 | 工具数 | 代码行数 | 复用率 |
|------|--------|---------|-------|
| **GoPaw (当前)** | 12 | ~600 | - |
| **PicoClaw** | 20+ | ~2,000 | 40% |
| **CoPaw** | 15+ | ~3,630 | 53% |
| **ZeroClaw** | 60+ | ~15,000 | 20% |
| **OpenClaw** | 92 | ~24,000 | 15% |

**复用率分析**:
- GoPaw 基础工具较完善，可直接复用 CoPaw 的部分实现
- 高级工具 (浏览器/文档) 需要较多工作量
- 接口设计可借鉴 PicoClaw

---

**报告生成时间**: 2026-03-05  
**分析工具**: GoPaw / PicoClaw / CoPaw / ZeroClaw / OpenClaw  
**工具接口**: Go / TypeScript / Rust / Python  
**代码行数**: ~10,000 行
