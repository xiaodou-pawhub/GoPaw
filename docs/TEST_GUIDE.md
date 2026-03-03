# GoPaw 基础功能测试指南

> **测试目标**: 验证项目骨架是否正常工作  
> **预计时间**: 15-30 分钟  
> **创建时间**: 2026-03-01

---

## 📋 测试清单

- [ ] 环境检查
- [ ] 编译测试
- [ ] CLI 命令测试
- [ ] 配置生成测试
- [ ] 服务启动测试（需要 API Key）
- [ ] API 接口测试

---

## 1️⃣ 环境检查

### 1.1 检查 Go 版本

```bash
go version
```

**预期输出**:
```
go version go1.22.x darwin/amd64  (或 arm64)
```

**要求**: Go 1.22 或更高版本

**如果 Go 版本过低或未安装**:
```bash
# macOS (Homebrew)
brew install go@1.22

# 或从官网下载
# https://go.dev/dl/
```

---

### 1.2 检查项目文件

```bash
cd /Users/luoxiaodou/workspace/projects/claw/GoPaw

# 查看项目结构
ls -la

# 应该看到以下关键文件/目录:
# go.mod, go.sum, Makefile, config.yaml.example
# cmd/, internal/, pkg/, plugins/, ai/
```

---

## 2️⃣ 编译测试

### 2.1 下载依赖

```bash
go mod download
```

**预期**: 无错误，下载所有依赖包

**可能的问题**:
```
# 如果网络慢，使用国内镜像
export GOPROXY=https://goproxy.cn,direct
go mod download
```

---

### 2.2 编译项目

```bash
# 方式 1: 直接编译
go build ./...

# 方式 2: 使用 Make
make build

# 方式 3: 编译主程序
go build -o gopaw ./cmd/gopaw
```

**✅ 成功标志**: 无错误输出，生成 `gopaw` 二进制文件

**❌ 如果编译失败**:
```bash
# 清理缓存后重试
go clean -modcache
go mod tidy
go build ./...

# 记录错误信息，反馈给我
```

---

### 2.3 验证编译产物

```bash
# 检查二进制文件
ls -lh gopaw

# 查看文件类型
file gopaw

# 预期输出类似:
# gopaw: Mach-O 64-bit executable arm64
```

---

## 3️⃣ CLI 命令测试

### 3.1 版本命令

```bash
./gopaw version
# 或
go run ./cmd/gopaw version
```

**预期输出**:
```
GoPaw version 0.1.0
Go go1.22.x darwin/amd64
```

---

### 3.2 帮助命令

```bash
./gopaw --help
```

**预期输出**:
```
GoPaw - Lightweight AI Assistant Workbench

Usage:
  gopaw <command> [flags]

Commands:
  init      Generate a default config.yaml in the current directory
  start     Start the GoPaw server
  version   Print version information
```

---

### 3.3 Init 命令（生成配置）

```bash
./gopaw init
```

**预期输出**:
```
Generated config file: config.yaml
```

**验证**:
```bash
# 查看生成的配置文件
cat config.yaml

# 检查文件大小
ls -lh config.yaml
```

**config.yaml 应该包含**:
```yaml
app:
  name: "GoPaw"
  language: zh
  timezone: Asia/Shanghai
  debug: false

server:
  host: 0.0.0.0
  port: 8088

storage:
  type: sqlite
  path: data/gopaw.db

llm:
  provider: openai_compatible
  base_url: https://api.openai.com/v1
  api_key: ${OPENAI_API_KEY}
  model: gpt-4o-mini
  ...
```

---

## 4️⃣ 配置测试

### 4.1 环境变量设置

**方式 1: 使用 .env 文件**

```bash
# 复制示例文件
cp .env.example .env

# 编辑 .env 文件
vim .env
# 或
nano .env
```

在 `.env` 中添加:
```bash
OPENAI_API_KEY=sk-your-api-key-here
```

**方式 2: 直接修改 config.yaml**

```bash
vim config.yaml
```

找到 `llm.api_key` 行，直接填入你的 API Key:
```yaml
llm:
  api_key: sk-your-api-key-here  # 直接填写，不用 ${}
```

> **注意**: 如果你没有 API Key，可以先跳过启动测试，只做前面的编译和 CLI 测试

---

### 4.2 配置验证

```bash
# 检查配置文件格式
cat config.yaml | head -30
```

---

## 5️⃣ 服务启动测试

### ⚠️ 前提条件

启动服务需要有效的 LLM API Key。如果你没有，可以跳过这一步。

### 5.1 启动服务

```bash
# 方式 1: 使用二进制文件
./gopaw start

# 方式 2: 使用 go run（开发模式，支持热重载）
go run ./cmd/gopaw start

# 方式 3: 指定配置文件
./gopaw start --config config.yaml
```

---

### 5.2 观察启动日志

**✅ 成功启动的日志**:
```
2026-03-01T12:00:00.000Z    INFO    GoPaw started
2026-03-01T12:00:00.000Z    INFO    server started    {"addr": "0.0.0.0:8088"}
```

**如果看到错误**:
```
# 记录完整的错误信息
# 特别是 "ERROR" 或 "fatal" 开头的行
# 反馈给我分析
```

**常见错误及解决**:

| 错误信息 | 原因 | 解决方法 |
|---------|------|---------|
| `failed to load config` | 配置文件格式错误 | 检查 YAML 缩进 |
| `OPENAI_API_KEY is empty` | 未配置 API Key | 设置环境变量或修改 config.yaml |
| `port already in use` | 8088 端口被占用 | 修改 config.yaml 中的 port |
| `failed to open database` | data 目录权限问题 | `mkdir -p data && chmod 755 data` |

---

### 5.3 保持服务运行

启动后，服务会持续运行。**不要关闭终端**，打开**新的终端窗口**继续下一步测试。

---

## 6️⃣ API 接口测试

在**新终端**中执行以下命令（服务运行时）:

### 6.1 健康检查

```bash
curl -v http://localhost:8088/health
```

**预期响应** (HTTP 200):
```json
{"status": "ok"}
```

---

### 6.2 版本信息

```bash
curl -s http://localhost:8088/api/system/version | jq .
```

**预期响应**:
```json
{
  "version": "0.1.0",
  "build_time": "...",
  "go_version": "..."
}
```

---

### 6.3 已加载的 Skills

```bash
curl -s http://localhost:8088/api/skills | jq .
```

**预期响应**:
```json
{
  "skills": [
    {
      "name": "...",
      "version": "...",
      "level": 1
    }
  ]
}
```

---

### 6.4 对话测试（需要 API Key）

```bash
# 普通对话
curl -X POST http://localhost:8088/api/agent/chat \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "test-001",
    "content": "你好，你是谁？"
  }' | jq .

# 流式输出（SSE）
curl -N "http://localhost:8088/api/agent/chat/stream?session_id=test-001&content=你好"
```

**预期响应** (普通对话):
```json
{
  "code": 0,
  "data": {
    "session_id": "test-001",
    "content": "你好，我是 GoPaw...",
    "elapsed_ms": 1234
  }
}
```

---

### 6.5 工具调用测试（需要 API Key + TAVILY_API_KEY）

```bash
# 测试 web_search 工具
curl -X POST http://localhost:8088/api/agent/chat \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": "test-002",
    "content": "搜索一下最新的 AI 新闻"
  }' | jq .
```

---

## 7️⃣ 数据库验证

### 7.1 检查 SQLite 数据库

```bash
# 检查数据库文件是否存在
ls -lh data/gopaw.db

# 查看数据库表结构
sqlite3 data/gopaw.db ".tables"

# 预期输出:
# cron_jobs          messages           sessions
# memory_summaries   messages_fts
```

### 7.2 查看对话历史

```bash
# 查看会话
sqlite3 data/gopaw.db "SELECT id, user_id, channel FROM sessions;"

# 查看消息
sqlite3 data/gopaw.db "SELECT id, role, content FROM messages LIMIT 5;"
```

---

## 8️⃣ 日志检查

### 8.1 查看日志文件

```bash
# 如果配置了文件日志输出
ls -lh logs/

# 查看最新日志
tail -f logs/gopaw.log
```

### 8.2 控制台日志

观察服务启动后的控制台输出，应该看到:
```
INFO    GoPaw started
INFO    channel started    {"channel": "console"}
INFO    server started     {"addr": "0.0.0.0:8088"}
```

---

## 📝 测试记录模板

请记录以下信息:

```markdown
## 测试环境
- Go 版本：go1.___
- 操作系统：macOS ___ (Intel/Apple Silicon)
- 测试时间：2026-03-01 __:__

## 测试结果

### 编译测试
- [ ] go build ./...  ✅ / ❌
- [ ] ./gopaw version  ✅ / ❌

### CLI 测试
- [ ] ./gopaw init  ✅ / ❌
- [ ] config.yaml 生成  ✅ / ❌

### 启动测试
- [ ] ./gopaw start  ✅ / ❌ / ⏭️ (无 API Key)
- [ ] 服务正常启动  ✅ / ❌ / ⏭️

### API 测试
- [ ] /health  ✅ / ❌ / ⏭️
- [ ] /api/system/version  ✅ / ❌ / ⏭️
- [ ] /api/skills  ✅ / ❌ / ⏭️
- [ ] /api/agent/chat  ✅ / ❌ / ⏭️

## 遇到的问题

1. [问题描述]
   - 错误信息：___
   - 复现步骤：___

2. ...
```

---

## 🆘 遇到问题怎么办？

### 问题分类

| 类型 | 描述 | 处理 |
|------|------|------|
| 🔴 阻塞性 | 编译失败、启动失败 | 立即反馈，优先解决 |
| 🟡 功能性 | API 返回错误、功能异常 | 记录详细信息，稍后解决 |
| 🟢 优化性 | 性能慢、日志不清晰 | 记录，后续优化 |

### 反馈格式

遇到问题时，请提供:

```
【问题类型】编译失败/启动失败/API 错误/其他

【错误信息】
(复制完整的错误日志)

【复现步骤】
1. ___
2. ___
3. ___

【环境信息】
- Go 版本：___
- 操作系统：___
- 配置文件：(脱敏后粘贴 config.yaml)
```

---

## ✅ 测试完成标准

全部测试通过后，你应该能够:

- [ ] 成功编译项目
- [ ] 运行所有 CLI 命令
- [ ] 启动服务（有 API Key 的情况下）
- [ ] 调用健康检查 API
- [ ] 进行简单对话（有 API Key 的情况下）
- [ ] 查看数据库中的对话记录

---

*文档结束*
