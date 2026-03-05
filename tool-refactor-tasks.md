# GoPaw 工具系统重构任务分析报告

**分析日期**: 2026-03-05  
**分析范围**: `/Users/luoxiaodou/workspace/projects/claw/GoPaw`  
**参考项目**: PicoClaw, CoPaw, ZeroClaw, OpenClaw  

---

## 🔍 执行摘要

GoPaw 当前的工具系统在架构设计上存在接口单一性、错误处理粗糙、缺乏结构化返回值等核心问题。基于对 PicoClaw 成功实践的分析，建议进行系统性重构，重点改进 Tool 接口、优化 Execute 执行流程、增强错误处理能力，并引入安全审批机制。

**重构风险等级**: 中等  
**预计工作量**: 12-16 人天  
**推荐优先级**: 高（必须完成核心接口重构）

---

## 1️⃣ 当前工具系统架构痛点分析

### 1.1 Tool 接口设计缺陷 (`pkg/plugin/tool.go:23-35`)

**问题描述**:  
当前 Tool 接口仅返回 `(string, error)`，这是最严重的架构瓶颈。

```go
// pkg/plugin/tool.go:35
Execute(ctx context.Context, args map[string]interface{}) (string, error)
```

**具体痛点**:

#### 痛点 1: 无法区分 LLM 和 User 的内容
- **问题**: LLM 需要的上下文与用户可见的消息混合在同一字符串中
- **影响**: Agent 无法向用户发送友好提示，或者将敏感内部信息暴露给用户
- **代码引用**: `internal/tools/shell.go:93`, `internal/tools/file_read.go:58`
- **参考 PicoClaw**: `pkg/tools/result.go:118-125` 的 `UserResult()` 功能

#### 痛点 2: 缺少错误语义表达
- **问题**: 错误信息仅通过 error 返回，无法携带对 LLM 的解释性内容
- **影响**: Agent 难以理解错误原因从而重试或调整策略
- **代码引用**: `internal/tools/web_search.go:108` 直接返回 `fmt.Errorf`
- **参考 PicoClaw**: `pkg/tools/result.go:97-104` 的 `ErrorResult()` 结构化错误

#### 痛点 3: 不支持异步操作
- **问题**: 所有工具必须同步返回，长时任务阻塞 Agent Loop
- **影响**: Subagent、后台任务等场景无法使用
- **代码引用**: `internal/tools/sub_agent.go` (目前未实现)
- **参考 PicoClaw**: `pkg/tools/base.go:65-70` 的 `AsyncTool` 接口

#### 痛点 4: 缺少 Context 上下文注入能力
- **问题**: 工具无法获取 Channel/ChatID 上下文，无法实现多通道定制
- **影响**: 同一工具在不同平台表现不一致时无法适配
- **代码引用**: `internal/tools/http_get.go:74` 固定 User-Agent
- **参考 PicoClaw**: `pkg/tools/base.go:15-18` 的 `ContextualTool` 接口

---

### 1.2 Executor 执行流程问题 (`internal/tool/executor.go:27-61`)

**问题描述**:  
Executor 作为执行入口，缺乏灵活的结果处理机制。

#### 问题 1: 硬编码截断逻辑
```go
// internal/tool/executor.go:25
const maxToolOutputRunes = 50_000
```
- **问题**: 所有工具统一截断，无法按场景调整
- **影响**: 小内容被截断（如 JSON），大内容无法区分重要性
- **参考 PicoClaw**: `pkg/tools/result.go:132-137` 的 `MediaResult()` 允许工具自行决定是否截断

#### 问题 2: 日志与执行逻辑耦合
```go
// internal/tool/executor.go:45-57
e.logger.Info("tool executing", zap.String("tool", toolName), zap.Any("args", args))
// ... 执行后 ...
e.logger.Warn("tool execution failed", ...)
```
- **问题**: Executor 负责工具日志，违反单一职责原则
- **影响**: 工具无法自定义日志等级（如 ` ToDo` 工具的调试日志 vs `FileRead` 的普通日志）
- **参考 PicoClaw**: `pkg/tools/registry.go:86-111` 在 Registry 层面根据 `ToolResult` 类型决定日志策略

#### 问题 3: 缺少异步支持
```go
// internal/tool/executor.go:29
func (e *Executor) Execute(ctx context.Context, toolName, argsJSON string) (string, error)
```
- **问题**: 同步签名无法支持异步工具
- **影响**: `SubAgent` 等需要后台完成的工具无法实现
- **参考 PicoClaw**: `pkg/tools/toolloop.go:135` 支持 `AsyncTool` 并通过回调机制处理

#### 问题 4: 参数解析重复代码
```go
// internal/tool/executor.go:35-43
var args map[string]interface{}
if argsJSON != "" {
    if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
        return "", fmt.Errorf("executor: parse args for %q: %w", toolName, err)
    }
}
if args == nil {
    args = make(map[string]interface{})
}
```
- **问题**: 每个工具都需重复此逻辑（见 `shell.go:48-61`）
- **影响**: 代码冗余，容易出错
- **参考 PicoClaw**: Executor 层统一处理 args 解析，工具函数直接接收 `map[string]any`

---

### 1.3 Registry 设计 (`internal/tool/registry.go`)

**问题描述**:  
Registry 功能单一，仅提供注册查找。

#### 问题 1: 缺少工具列表排序
```go
// internal/tool/registry.go:73-80
func (r *Registry) Names() []string {
    // ... 直接遍历 map
    for name := range r.tools {
        names = append(names, name)
    }
}
```
- **问题**: Map 遍历顺序不确定，导致 System Prompt 不稳定
- **影响**: LLM Prefix Cache 失效，token 浪费
- **参考 PicoClaw**: `pkg/tools/registry.go:116-127` 的 `sortedToolNames()` 实现确定性排序

#### 问题 2: 缺少工具摘要查询
- **问题**: 无法快速获取工具列表的可读摘要
- **影响**: 调试和监控困难
- **参考 PicoClaw**: `pkg/tools/registry.go:192-203` 的 `GetSummaries()` 方法

---

## 2️⃣ 基于参考项目的改进建议

### 2.1 PicoClaw 的 ToolResult 结构化设计

**核心思想**: 使用结构体包裹结果，提供明确语义

```go
// picoclaw/pkg/tools/result.go:8
type ToolResult struct {
    ForLLM   string  // 传给 LLM 的内容（必需）
    ForUser  string  // 传给用户的可选消息
    Silent   bool    // 是否静默（不发送用户）
    IsError  bool    // 是否错误
    Async    bool    // 是否异步
    Err      error   // 内部错误（不序列化）
    Media    []string // 媒体引用（图片、文件等）
}
```

**优势**:
- ✅ 明确语义： `(string, error)` → `(Content, Error, Metadata)`
- ✅ 扩展性强：新增字段不影响现有工具
- ✅ JSON 友好: 可直接序列化为 Agent Protocol 格式

**GoPaw 适配成本**:  
- **高**: 需修改 `Tool` 接口，所有工具实现需重构  
- **收益**: 极高，奠定未来扩展基础

**代码示例** (PicoClaw 的 `WebSearchTool`):
```go
// picoclaw/pkg/tools/web.go:502
func (t *WebSearchTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
    // ... 搜索逻辑 ...
    if len(result.Results) == 0 {
        return NewToolResult("No results found.")
    }
    var sb strings.Builder
    for i, r := range result.Results {
        sb.WriteString(fmt.Sprintf("%d. **%s**\n   URL: %s\n", i+1, r.Title, r.URL))
    }
    return UserResult(sb.String()) // 自动设置 ForUser = ForLLM
}
```

---

### 2.2 PicoClaw 的 ContextualTool / AsyncTool 可选接口

**设计模式**: 可选接口（Optional Interface）

```go
// picoclaw/pkg/tools/base.go:15-70
type ContextualTool interface {
    Tool
    SetContext(channel, chatID string)
}

type AsyncCallback func(ctx context.Context, result *ToolResult)

type AsyncTool interface {
    Tool
    SetCallback(cb AsyncCallback)
}
```

**优势**:
- ✅ 向后兼容: 旧工具无需实现
- ✅ 功能解耦: 上下文/异步逻辑与核心业务分离
- ✅ 灵活性高: Registry 自动检测并调用

**GoPaw 适配策略**:
1. 在 `pkg/plugin/tool.go` 增加可选接口（不破坏兼容性）
2. `Executor` 在执行前检测接口实现
3. Registry 负责注入上下文和回调

```go
// 改进后的 registry.go:43-45
func (r *Registry) Execute(ctx context.Context, name string, args map[string]any) *ToolResult {
    tool, ok := r.Get(name)
    if !ok { return ErrorResult(...) }

    // 注入上下文（如果实现）
    if contextual, ok := tool.(ContextualTool); ok {
        contextual.SetContext("feishu", "chat123")
    }

    // 设置回调（如果实现）
    if asyncTool, ok := tool.(AsyncTool); ok && cb != nil {
        asyncTool.SetCallback(cb)
    }

    return tool.Execute(ctx, args) // 返回 *ToolResult
}
```

---

### 2.3 ZeroClaw 的 Tool Trait 设计

**核心思想**: Trait（特征）而非接口，提供默认实现

```go
// ZeroClaw 风格（示意）
type ToolTrait interface {
    Name() string
    Description() string
    Parameters() ToolParameters
    Execute(ctx context.Context, args map[string]any) *ToolResult

    // 可选方法，有默认实现
    MaxOutputSize() int  // 默认 50000
    IsAsync() bool       // 默认 false
    ValidateArgs(args map[string]any) error  // 默认无校验
}
```

**优势**:
- ✅ 减少样板代码
- ✅ 提供合理的默认行为
- ✅ 便于工具继承和组合

**GoPaw 实践建议**:  
参考 `picoclaw/pkg/tools/base.go:112-131` 的工具基类模式，提供 `BaseTool` 结构体。

---

### 2.4 OpenClaw 的 Exec Approvals 安全系统

**核心思想**: 敏感命令白名单 + 用户审批

**PicoClaw 实践** (`picoclaw/pkg/tools/shell.go:290-370`):
```go
// picoclaw/pkg/tools/shell.go:290
func (t *ExecTool) guardCommand(command, cwd string) string {
    lower := strings.ToLower(strings.TrimSpace(command))
    
    // 白名单检查
    for _, pattern := range t.customAllowPatterns {
        if pattern.MatchString(lower) {
            explicitlyAllowed = true
            break
        }
    }
    
    // 黑名单检查
    for _, pattern := range t.denyPatterns {
        if pattern.MatchString(lower) {
            return "" // 阻止危险命令
        }
    }
    return command
}
```

**黑名单示例**:
```go
defaultDenyPatterns = []*regexp.Regexp{
    regexp.MustCompile(`\brm\s+-[rf]{1,2}\b`),       // rm -rf
    regexp.MustCompile(`\b	shutdown|reboot|poweroff)\b`), // 关机命令
    regexp.MustCompile(`\bceph\s+osd\s+destroy\b`),   // 数据销毁
    regexp.MustCompile(`>\s*/dev/(sd[a-z]|hd[a-z])`), // 写入块设备
}
```

**GoPaw 应用建议**:
1. 在 `ShellTool` 增加命令校验层
2. 提供配置项 `allowed_commands` / `denied_commands`
3. 敏感操作（如文件写入）增加审批步骤

---

## 3️⃣ 重构任务清单

### 优先级标签说明
- 🔴 **高优先级**: 必须完成，影响核心功能  
- 🟠 **中优先级**: 应该完成，提升系统质量  
- 🟡 **低优先级**: 可选，锦上添花  

---

### 📋 详细任务列表

#### 🔴 高优先级任务 (必须完成)

| # | 任务 | 文件位置 | 工作量 | 说明 |
|---|------|----------|--------|------|
| H1 | 重构 `pkg/plugin/tool.go` 接口 | `pkg/plugin/tool.go:23-35` | 0.5 天 | `Execute` 返回类型改为 `*ToolResult`，新增 `ToolResult` 结构体 |
| H2 | 重构所有内置工具 | `internal/tools/*.go` | 4 天 | 将 13 个工具（见 H2.1）全部适配新接口 |
| H3 | 重构 `internal/tool/executor.go` | `internal/tool/executor.go:27-61` | 1 天 | 支持 `ToolResult`，移除硬编码截断，解耦日志逻辑 |
| H4 | 实现 `internal/tool/registry.go` 增强 | `internal/tool/registry.go` | 1 天 | 增加工具列表排序、摘要查询、可选接口支持 |

##### H2.1 内置工具列表 (需重构)
- `internal/tools/shell.go` (ShellTool)
- `internal/tools/file_read.go` (FileReadTool)
- `internal/tools/file_write.go` (FileWriteTool)  
- `internal/tools/file_search.go` (FileSearchTool)
- `internal/tools/http_get.go` (HTTPGetTool)
- `internal/tools/http_post.go` (HTTPPostTool)
- `internal/tools/web_search.go` (WebSearchTool)
- `internal/tools/memory_store.go` (MemoryStoreTool)
- `internal/tools/memory_read.go` (MemoryReadTool)
- `internal/tools/memory_write.go` (MemoryWriteTool)
- `internal/tools/memory_recall.go` (MemoryRecallTool)
- `internal/tools/memory_forget.go` (MemoryForgetTool)
- `internal/tools/sub_agent.go` (SubAgentTool) ← **新增功能**

**预估总工作量**: 1.5 + 4 + 1 + 1 = **7.5 人天**

---

#### 🟠 中优先级任务 (应该完成)

| # | 任务 | 文件位置 | 工作量 | 说明 |
|---|------|----------|--------|------|
| M1 | 添加命令白名单/黑名单 (Shell 安全) | `internal/tools/shell.go:96-100` | 1 天 | 参考 PicoClaw 的 `guardCommand()` 和 `denyPatterns` |
| M2 | 实现 `ContextualTool` 支持 | `pkg/plugin/tool.go` | 0.5 天 | 新增 `SetContext` 可选接口 |
| M3 | 实现 `AsyncTool` 支持 | `pkg/plugin/tool.go` | 1 天 | 新增 `AsyncTool` + `AsyncCallback` |
| M4 | 工具执行超时统一管理 | `internal/tool/executor.go:29` | 0.5 天 | 统一默认 timeout，支持工具级别覆盖 |
| M5 | 增加工具指标监控 | `internal/tool/registry.go` | 1 天 | 统计执行次数、平均耗时、错误率 |

**预估总工作量**: 4 人天

---

#### 🟡 低优先级任务 (可选)

| # | 任务 | 文件位置 | 工作量 | 说明 |
|---|------|----------|--------|------|
| L1 | 添加 `BaseTool` 辅助结构体 | `pkg/plugin/tool.go` | 0.5 天 | 提供 `Name`/`Description` 默认实现 |
| L2 | 工具参数自动校验 | `internal/tool/registry.go` | 1 天 | 基于 JSON Schema 自动校验 args |
| L3 | 支持工具别名 | `internal/tool/registry.go` | 0.5 天 | 允许一个工具注册多个名称 |
| L4 | 工具文档自动生成 | `internal/tool/registry.go` | 1 天 | 导出 markdown 工具文档 |
| L5 | 工具依赖管理 | `internal/tool/registry.go` | 2 天 | 工具 A 依赖工具 B 的权限 |  

**预估总工作量**: 5 人天

---

## 4️⃣ 接口设计对比

### 当前 vs PicoClaw vs ZeroClaw

| 特性 | GoPaw (当前) | PicoClaw | ZeroClaw |
|------|--------------|----------|----------|
| **返回类型** | `(string, error)` | `*ToolResult` | `ToolResult` (值) |
| **错误语义** | 通过 `error` 返回 | `IsError bool` | `IsError bool` |
| **用户提示** | 混在 `string` 中 | `ForUser string` | `ForUser string` |
| **异步支持** | ❌ 无 | ✅ 可选接口 | ✅ Trait |
| **上下文注入** | ❌ 无 | ✅ `SetContext` | ✅ 参数注入 |
| **媒体引用** | ❌ 无 | ✅ `Media []string` | ❌ 无 |
| **JSON 序列化** | 手动 `json.Marshal` | 自动 | 自动 |
| **代码示例** | `return result, nil` | `return NewToolResult(result)` | `return ToolResult{Content: result}` |

### 推荐方案：PicoClaw 模式

**理由**:
1. ✅ **指针返回** (`*ToolResult`) 更节省内存（避免频繁拷贝大字符串）
2. ✅ **可选接口** (ContextualTool/AsyncTool) 向后兼容
3. ✅ **丰富的辅助函数** (NewToolResult, ErrorResult, UserResult 等)

---

## 5️⃣ 推荐实现顺序

### 阶段一：基础重构 (3 天)
1. **Day 1**: 创建 `ToolResult` 结构体 + `pkg/plugin/result.go`
2. **Day 2**: 重构 Tool 接口 + 实现 3 个核心工具 (FileRead, HTTPGet, WebSearch)
3. **Day 3**: 重构 Executor + Registry

### 阶段二：功能增强 (2 天)
4. **Day 4**: 实现 ContextualTool + AsyncTool
5. **Day 5**: 完成剩余工具迁移 + 编写测试

### 阶段三：生产加固 (1.5 天)
6. **Day 6**: 安全审批系统 (Shell 黑名单) + 超时管理
7. **Day 7**: 监控指标 + 文档测试

**总时间**: 7 天 (不包括 L 级任务)

---

## 6️⃣ 风险评估

### 📊 风险矩阵

| 风险项 | 概率 | 影响 | 缓解措施 |
|--------|------|------|----------|
| **接口重构破坏现有工具** | 高 | 高 | 分阶段迁移，保留旧接口兼容层 |
| **测试覆盖率不足** | 中 | 高 | 重构后覆盖率必须 ≥ 80% |
| **性能下降** | 中 | 中 | Benchmark 对比(String vs ToolResult) |
| **异步工具回调死锁** | 低 | 高 | 使用 context.WithTimeout + select |
| **文档与代码不同步** | 高 | 中 | 建立 PR Review Checklist |

### ⚠️ 关键风险详细分析

#### R1: 破坏性变更 (高风险)
**现状**: 修改 `Tool` 接口 → 所有 plugin 重建失败

**缓解方案**:
```go
// pkg/plugin/tool.go:23
type Tool interface {
    // ... 现有方法 ...
    // 新增 ExecuteV2 供过渡使用
    ExecuteV2(ctx context.Context, args map[string]any) *ToolResult
}

// 在 pkg/plugin/v2/ 下建立新包
// 逐步迁移，最后废弃 v1
```

#### R2: 测试中断 (高风险)
**现状**: `internal/tools/*_test.go` 依赖旧接口

**缓解方案**:
- 重构时同步更新测试
- 使用 `t.Run()` 分离测试用例
- 增加 Fuzz Test ( torture test)

#### R3: 上线回滚困难 (中风险)
**现状**: 部分工具已生产使用

**缓解方案**:
1. 功能开关 (Feature Flag)
2. 配置项 `tool.api_version: v1|v2`
3. 金丝雀发布 (Canary Rollout)

---

## 7️⃣ 性能影响评估

### 预期性能变化

| 指标 | 当前 | 重构后 | 变化 |
|------|------|--------|------|
| **内存分配** | 频繁 String 拼接 | ToolResult 指针复用 | ↓ 减少 GC |
| **JSON 序列化** | 每次手动 Marshal | 仅必要时序列化 | ↓ 减少 CPU |
| **函数调用开销** | 直接返回 string | 包装 ToolResult | ↑ 可忽略 (<1μs) |
| **缓存命中率** | 随机 (map 无序) | 稳定 (sorted names) | ↑ 显著提升 |

### Benchmark 建议

```go
// internal/tool/executor_bench_test.go
func BenchmarkExecute_FileRead(b *testing.B) {
    reg := NewRegistry()
    reg.Register(&FileReadTool{})
    exec := NewExecutor(reg, nil)
    
    args := map[string]any{"path": "/tmp/bench.txt"}
    argsJSON, _ := json.Marshal(args)
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, _ = exec.Execute(context.Background(), "file_read", string(argsJSON))
    }
}
```

---

## 8️⃣ 测试策略

### 单元测试覆盖清单

| 工具 | 测试场景 | 覆盖率目标 |
|------|----------|------------|
| **FileReadTool** | 正常读取、权限拒绝、大文件截断 | 90% |
| **ShellTool** | 普通命令、危险命令拦截、超时 | 95% |
| **HTTPGetTool** | 正常响应、404/500、超时 | 85% |
| **WebSearchTool** | 成功搜索、API 失败、空结果 | 80% |
| **Memory Tools** | KV 存取、过期清理、分页 | 85% |

### 集成测试场景

```python
# integration_test.go (示意)
def test_tool_result_semantic():
    # 验证 ToolResult 的 ForLLM/ForUser 分离
    result = FileReadTool().execute(...)
    assert result.for_llm != ""
    assert result.for_user == ""  #文件读取不应显式通知用户
    
def test_async_tool_callback():
    # 验证异步工具回调机制
    async_tool = AsyncTool()
    callback_called = False
    async_tool.set_callback(lambda ctx, r: callback_called := True)
    result = async_tool.execute(...)
    assert result.async
    time.sleep(1)  # 等待异步完成
    assert callback_called
```

---

## 9️⃣ 文档更新计划

### 需要更新的文档

| 文档 | 更新内容 | 负责人 |
|------|----------|--------|
| `README.md` | Tool 接口示例 (新版) | ✅ |
| `docs/tools-guide.md` | `ToolResult` 使用说明 | ✅ |
| `docs/architecture.md` | 工具系统架构图 | ✅ |
| `pkg/plugin/README.md` | 接口迁移指南 (v1→v2) | ✅ |
| `CONTRIBUTING.md` | 新工具开发 Checklist | ✅ |

### 代码注释要求

```go
// pkg/plugin/tool.go:35
// Execute 执行工具并返回结构化结果。
//
// 返回值中的 ForLLM 字段是必需的，用于给 LLM 提供上下文。
// ForUser 字段可选，当需要向用户展示结果时设置。
// ErrorResult() 和 UserResult() 等辅助函数可简化常见场景。
//
// 示例:
//
//	result := SilentResult("配置已保存")
//	return &result
//
// 如果是异步操作，实现 AsyncTool 接口并返回 AsyncResult()
// 详见: https://github.com/gopaw/gopaw/docs/tools-async.md
Execute(ctx context.Context, args map[string]any) *ToolResult
```

---

## 🔟 附录：参考实现对比表

### ✅ 代码块对比

#### ❌ GoPaw 当前写法
```go
// internal/tools/file_read.go:42
func (t *FileReadTool) Execute(_ context.Context, args map[string]interface{}) (string, error) {
    path, ok := args["path"].(string)
    if !ok || path == "" {
        return "", fmt.Errorf("file_read: 'path' argument is required")
    }
    // ... 文件读取 ...
    return string(content), nil
}
```

#### ✅ PicoClaw 改进版
```go
// picoclaw/pkg/tools/filesystem.go:100
func (t *ReadFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
    path, ok := args["path"].(string)
    if !ok || path == "" {
        return ErrorResult("path argument is required")
    }
    // ... 文件读取 ...
    if restrict && !validatePath(...) {
        return ErrorResult("access denied: path outside workspace")
    }
    return SilentResult(string(content))
}
```

### 📊 关键改进点总结

| 改进点 | 说明 | 收益 |
|--------|------|------|
| **结构化结果** | `*ToolResult` 替代 `(string, error)` | Agent 能理解语义 |
| **分离关注点** | ForLLM / ForUser / Media 分离 | 用户体验优化 |
| **错误语义** | `IsError` 标记而非 error 类型 | 错误处理统一 |
| **异步支持** | AsyncTool 接口 + Callback | 长任务不阻塞 |
| **安全增强** | 命令黑名单 + 路径校验 | 生产级安全性 |
| **上下文注入** | SetContext 自动调用 | 多平台适配简化 |

---

## 📌 结论与建议

### 🔴 必须立即行动的事项
1. **批准重构计划** (预计 7 人天)
2. **分配测试资源** (确保覆盖率 ≥ 80%)
3. **制定迁移时间表** (避免生产中断)

### 🟠 推荐后续优化
1. **引入 Feature Flag** (支持灰度发布)
2. **工具文档自动化** (减少维护成本)
3. **性能监控仪表盘** (追踪回归)

### 🟡 长期规划
1. **工具市场** (Plugin Marketplace)
2. **工作流编排** (Tool Chain)
3. **AI 生成工具** (AutoToolGen)

---

## 📚 参考文献

1. **PicoClaw Tools Design**: `/Users/luoxiaodou/workspace/projects/claw/picoclaw/pkg/tools/`  
2. **OpenAI Function Calling**: https://platform.openai.com/docs/guides/function-calling  
3. **ReAct Paper**: https://arxiv.org/abs/2210.03629  
4. **Contextual Tool Pattern**: https://github.com/sipeed/picoclaw/pkg/tools/base.go  
5. **Async Tool Callback**: https://github.com/sipeed/picoclaw/pkg/tools/subagent.go  

---

**报告生成**: 2026-03-05  
**分析工具**: GoPaw Analysis Script v1.0  
**下一次评审**: 2026-03-12 (重构启动后)
