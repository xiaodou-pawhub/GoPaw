# GoPaw P1 阶段任务规划

> **阶段目标**: 完善功能和稳定性，达到普通用户可上手使用  
> **预计周期**: 2026-03-02 ~ 2026-03-15 (2 周)  
> **文档版本**: v0.1.0

---

## 📋 任务总览

| 任务 | 优先级 | 预计耗时 | 依赖 | 状态 |
|------|--------|---------|------|------|
| P1-1: 记忆压缩功能 | P1 | 4h | P0 完成 | ⏳ |
| P1-2: 定时任务功能 | P1 | 6h | P0 完成 | ⏳ |
| P1-3: 单元测试补全 | P1 | 8h | P0 完成 | ⏳ |
| P1-4: 飞书频道接入 | P1 | 6h | P0 完成 | ⏳ |
| P1-5: 钉钉频道接入 | P1 | 4h | P1-4 | ⏳ |
| P1-6: Web Console 前端 | P1 | 16h | P0 完成 | ⏳ |

---

## ✅ P1-1: 记忆压缩功能

**位置**: `internal/memory/compress.go`, `internal/memory/manager.go`

**目标**: 当对话历史超过 token 限制时自动压缩，不丢失信息

**实现内容**:
- [ ] 精确 token 计数（使用 tiktoken-go）
- [ ] 自动触发压缩（超过 context_limit 的 80%）
- [ ] 用 LLM 生成摘要（100 字以内）
- [ ] 压缩后保留摘要 + 最近 N 条原始消息
- [ ] 压缩记录写入 SQLite memory_summaries 表
- [ ] 上下文构建时在头部插入摘要

**验收标准**:
```bash
# 1. 长对话后自动触发压缩
# 2. 查看数据库有压缩记录
sqlite3 data/gopaw.db "SELECT * FROM memory_summaries;"

# 3. 重启后对话历史包含摘要
curl -X POST http://localhost:8088/api/agent/chat \
  -d '{"session_id":"test","content":"我们之前聊了什么？"}'
```

**预计耗时**: 4 小时

---

## ✅ P1-2: 定时任务功能

**位置**: `internal/scheduler/manager.go`, `internal/scheduler/job.go`

**目标**: 能通过 API 创建定时任务并执行

**实现内容**:
- [ ] Cron 表达式解析（robfig/cron v3）
- [ ] 定时任务 CRUD API
- [ ] 任务到点触发，向指定频道发送消息
- [ ] 活跃时间窗口生效（active_from/active_until）
- [ ] 任务执行记录写入 SQLite
- [ ] 重启后任务从 SQLite 恢复

**API 设计**:
```
POST   /api/cron          # 创建任务
GET    /api/cron          # 列出任务
PUT    /api/cron/:id      # 更新任务
DELETE /api/cron/:id      # 删除任务
POST   /api/cron/:id/run  # 手动触发
```

**验收标准**:
```bash
# 1. 创建定时任务
curl -X POST http://localhost:8088/api/cron \
  -H "Content-Type: application/json" \
  -d '{
    "name": "每日早报",
    "cron_expr": "0 8 * * *",
    "channel": "console",
    "prompt": "发送今日天气和新闻"
  }'

# 2. 列出任务
curl http://localhost:8088/api/cron

# 3. 手动触发
curl -X POST http://localhost:8088/api/cron/xxx/run
```

**预计耗时**: 6 小时

---

## ✅ P1-3: 单元测试补全

**目标**: 核心模块测试覆盖率 > 60%

**优先覆盖的模块**:

### internal/agent/agent_test.go
```go
// Mock LLM Client
type mockLLM struct {
    responses []string
}

// 测试用例
func TestReActAgent_Process(t *testing.T) {
    // 1. 正常对话流程
    // 2. 工具调用流程
    // 3. 超过 maxSteps 处理
    // 4. LLM 错误处理
}
```

### internal/memory/store_test.go
```go
// 测试用例
func TestStore_CRUD(t *testing.T) {
    // 1. 会话创建
    // 2. 消息添加
    // 3. 消息查询
    // 4. FTS5 搜索
}

func TestStore_FTS5Triggers(t *testing.T) {
    // 1. INSERT 触发器
    // 2. DELETE 触发器
    // 3. UPDATE 触发器
}
```

### internal/tool/executor_test.go
```go
// 测试用例
func TestExecutor_Execute(t *testing.T) {
    // 1. file_read 执行
    // 2. file_write 执行
    // 3. shell_execute 执行
    // 4. 错误处理
}
```

### plugins/tools/*_test.go
```go
// 每个工具的独立测试
func TestFileRead_Execute(t *testing.T) { ... }
func TestShell_Execute(t *testing.T) { ... }
```

**验收标准**:
```bash
# 运行测试
go test ./... -v

# 查看覆盖率
go test ./... -cover

# 目标：核心模块 > 60%
```

**预计耗时**: 8 小时

---

## ✅ P1-4: 飞书频道接入

**位置**: `plugins/channels/feishu/plugin.go`

**目标**: 飞书机器人能收发消息

**前提条件**:
1. 在飞书开放平台创建应用
2. 获取 App ID / App Secret
3. 配置 Webhook 事件订阅地址

**实现内容**:
- [ ] 接收飞书消息事件
- [ ] Challenge 验证（首次配置）
- [ ] 加解密（如配置了 encrypt_key）
- [ ] 消息格式转换为 types.Message
- [ ] Agent 回复发送到飞书
- [ ] Token 过期自动刷新
- [ ] 支持 Markdown 格式回复

**验收标准**:
```bash
# 1. 飞书机器人发消息，GoPaw 能收到
# 2. GoPaw 回复能正确发送到飞书
# 3. 支持 @机器人 和私聊两种模式
# 4. 支持文本、图片、富文本消息
```

**配置示例**:
```yaml
plugins:
  enabled:
    - feishu
  plugin:
    feishu:
      app_id: cli_xxx
      app_secret: xxx
      verification_token: xxx
      encrypt_key: xxx  # 可选
```

**预计耗时**: 6 小时

---

## ✅ P1-5: 钉钉频道接入

**位置**: `plugins/channels/dingtalk/plugin.go`

**目标**: 钉钉机器人能收发消息

**实现内容**:
- [ ] 接收钉钉消息事件
- [ ] HMAC-SHA256 签名验证
- [ ] 消息格式转换
- [ ] Agent 回复发送
- [ ] 支持 Markdown 格式

**验收标准**:
```bash
# 1. 钉钉机器人发消息，GoPaw 能收到
# 2. GoPaw 回复能正确发送到钉钉
# 3. 支持 @机器人 和私聊
```

**预计耗时**: 4 小时（可参考飞书实现）

---

## ✅ P1-6: Web Console 前端

**位置**: `web/` 目录

**目标**: 开发 React 前端，嵌入二进制

**技术栈**:
- React 18 + TypeScript
- Ant Design v5
- Vite (构建工具)
- go:embed (嵌入方式)

**页面列表**:

### /chat — 聊天界面
- [ ] Markdown 渲染
- [ ] 流式输出（SSE）
- [ ] 文件上传
- [ ] 历史会话列表
- [ ] 新建/删除会话

### /settings — 设置页
- [ ] LLM 配置（API Key、模型选择）
- [ ] Agent 系统提示编辑
- [ ] 记忆系统配置
- [ ] 调试模式开关

### /channels — 频道管理
- [ ] 启用/禁用频道
- [ ] 状态展示
- [ ] 配置表单（飞书/钉钉等）

### /skills — 技能管理
- [ ] 技能列表
- [ ] 启用/禁用
- [ ] 技能配置

### /cron — 定时任务
- [ ] 任务列表
- [ ] 新增任务
- [ ] 手动触发
- [ ] 执行历史

**嵌入方式**:
```go
// cmd/gopaw/main.go
import _ "embed"

//go:embed web/dist/*
var webFS embed.FS

// Gin 静态服务
engine.StaticFS("/web", http.FS(webFS))
```

**验收标准**:
```bash
# 1. 构建前端
cd web
npm run build

# 2. 编译 GoPaw（嵌入前端）
go build -o gopaw ./cmd/gopaw

# 3. 启动访问
./gopaw start
# 访问 http://localhost:8088/web
```

**预计耗时**: 16 小时

---

## 📅 迭代计划

### Iteration 1 (2026-03-02 ~ 2026-03-05)
- [x] P0 阶段完成
- [ ] P1-1: 记忆压缩功能
- [ ] P1-2: 定时任务功能

### Iteration 2 (2026-03-06 ~ 2026-03-09)
- [ ] P1-3: 单元测试补全
- [ ] P1-4: 飞书频道接入

### Iteration 3 (2026-03-10 ~ 2026-03-12)
- [ ] P1-5: 钉钉频道接入
- [ ] P1-6: Web Console 前端（50%）

### Iteration 4 (2026-03-13 ~ 2026-03-15)
- [ ] P1-6: Web Console 前端（剩余）
- [ ] 集成测试
- [ ] 文档完善

---

## 🎯 P1 阶段完成标准

全部 P1 任务完成后，应该达到：

- ✅ 记忆系统自动压缩，长时间对话不丢失
- ✅ 定时任务可按 Cron 表达式执行
- ✅ 核心模块测试覆盖率 > 60%
- ✅ 飞书、钉钉频道可真实接入
- ✅ Web Console 前端可用（聊天、配置、管理）
- ✅ 普通用户可上手使用（有文档）

**此时项目达到 v0.2 发布标准。**

---

## 📞 通知节点

每完成一个 P1 任务，我会总结并通知你进行测试：

1. **P1-1 完成后** → 测试记忆压缩
2. **P1-2 完成后** → 测试定时任务
3. **P1-3 完成后** → 运行测试查看覆盖率
4. **P1-4 完成后** → 测试飞书接入（需要 App ID）
5. **P1-5 完成后** → 测试钉钉接入（需要 App ID）
6. **P1-6 完成后** → 测试 Web Console 前端

---

*文档结束*
