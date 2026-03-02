# GoPaw Web 前端运行指南

> **文档版本**: v1.0  
> **创建日期**: 2026-03-02  
> **适用对象**: 开发者和测试人员

---

## 📋 目录

1. [快速开始](#1-快速开始)
2. [运行方式](#2-运行方式)
3. [页面预览](#3-页面预览)
4. [功能测试](#4-功能测试)
5. [常见问题](#5-常见问题)
6. [开发调试](#6-开发调试)

---

## 1. 快速开始

### 最简单的方式

```bash
# 1. 进入 web 目录
cd /Users/luoxiaodou/workspace/projects/claw/GoPaw/web

# 2. 启动开发服务器
pnpm run dev

# 3. 打开浏览器访问
# http://localhost:5173/
```

**预期输出**:
```
VITE v5.4.21  ready in 500 ms

➜  Local:   http://localhost:5173/
➜  Network: use --host to expose
➜  press h + enter to show help
```

---

## 2. 运行方式

### 方式一：开发模式（推荐）

**适用场景**: 前端开发、功能测试、UI 调试

**优点**:
- ✅ 支持热重载（修改代码自动刷新）
- ✅ 错误提示友好
- ✅ 启动速度快

**步骤**:
```bash
cd /Users/luoxiaodou/workspace/projects/claw/GoPaw/web
pnpm run dev
```

**访问**: `http://localhost:5173/`

---

### 方式二：后端 + 前端同时运行

**适用场景**: 完整功能测试、API 集成测试

**步骤**:

```bash
# 终端 1: 启动后端
cd /Users/luoxiaodou/workspace/projects/claw/GoPaw
./gopaw start

# 终端 2: 启动前端
cd /Users/luoxiaodou/workspace/projects/claw/GoPaw/web
pnpm run dev
```

**访问**:
- 前端：`http://localhost:5173/`
- 后端 API：`http://localhost:8088/`

> ✅ Vite 开发服务器会自动将 `/api/*` 请求代理到后端 `http://localhost:8088/api/*`

---

### 方式三：生产构建预览

**适用场景**: 验证生产环境构建、性能测试

**步骤**:
```bash
# 1. 构建生产版本
cd /Users/luoxiaodou/workspace/projects/claw/GoPaw/web
pnpm run build

# 2. 预览构建产物
pnpm run preview
```

**访问**: `http://localhost:4173/`

---

## 3. 页面预览

### 3.1 引导页 (/setup)

**访问**: `http://localhost:5173/setup`

**界面预览**:
```
┌─────────────────────────────────────────┐
│                                         │
│           🐾 欢迎使用 GoPaw             │
│                                         │
│   请先配置 LLM 提供商以开始使用          │
│                                         │
│   ┌─────────────────────────────────┐  │
│   │      [ 开始配置 ]               │  │
│   └─────────────────────────────────┘  │
│                                         │
└─────────────────────────────────────────┘
```

**功能**:
- ✅ 检查 LLM 配置状态
- ✅ 未配置时引导到设置页
- ✅ 已配置时跳转到聊天页

---

### 3.2 LLM 配置页 (/settings/providers)

**访问**: `http://localhost:5173/settings/providers`

**界面预览**:
```
┌─────────────────────────────────────────┐
│  LLM 提供商                              │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │ + 添加提供商                       │ │
│  └───────────────────────────────────┘ │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ [OpenAI] gpt-4o-mini      [编辑]│   │
│  │                          [删除] │   │
│  └─────────────────────────────────┘   │
│                                         │
└─────────────────────────────────────────┘
```

**功能**:
- ✅ 添加 LLM 提供商（OpenAI、Azure 等）
- ✅ 编辑现有提供商
- ✅ 删除提供商
- ✅ 切换活跃提供商

---

### 3.3 Agent 设定页 (/settings/agent)

**访问**: `http://localhost:5173/settings/agent`

**界面预览**:
```
┌─────────────────────────────────────────┐
│  Agent 设定                              │
│                                         │
│  编辑 Agent 的角色设定和系统提示          │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 你是一个智能助理，名字叫 GoPaw。 │   │
│  │ 你会帮助用户完成各种任务...     │   │
│  │                                 │   │
│  │                                 │   │
│  └─────────────────────────────────┘   │
│                                         │
│  [ 保存 ]                               │
└─────────────────────────────────────────┘
```

**功能**:
- ✅ 编辑 AGENT.md 内容
- ✅ 保存 Agent 系统提示
- ✅ 即时生效（无需重启）

---

### 3.4 频道配置页 (/settings/channels)

**访问**: `http://localhost:5173/settings/channels`

**界面预览**:
```
┌─────────────────────────────────────────┐
│  频道配置                                │
│                                         │
│  ⚠️ 频道配置功能开发中...               │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 🏢 飞书                          │   │
│  │ 飞书机器人配置                   │   │
│  │                           [编辑]│   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 🚀 钉钉                          │   │
│  │ 钉钉机器人配置                   │   │
│  │                           [编辑]│   │
│  └─────────────────────────────────┘   │
│                                         │
└─────────────────────────────────────────┘
```

**功能**:
- ⚠️ 占位页面（功能开发中）

---

### 3.5 聊天页 (/chat)

**访问**: `http://localhost:5173/chat`

**界面预览**:
```
┌─────────────────────────────────────────┐
│  聊天                                    │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 🤖                              │   │
│  │ 你好！我是 GoPaw，你的 AI 助理。  │   │
│  │ 有什么可以帮你的吗？             │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 🧑                              │   │
│  │ 你好，你能帮我做什么？           │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 🤖 我可以帮助你...              │   │
│  └─────────────────────────────────┘   │
│                                         │
│  ┌─────────────────────────────────┐   │
│  │ 输入消息...                      │   │
│  │                                 │   │
│  └─────────────────────────────────┘   │
│  [ 发送 ]                               │
└─────────────────────────────────────────┘
```

**功能**:
- ✅ Markdown 消息渲染
- ✅ 流式输出（待后端集成）
- ✅ 历史会话显示
- ✅ Mock 回复（当前为测试数据）

---

## 4. 功能测试

### 测试 1: 引导页流程

**步骤**:
```
1. 访问 http://localhost:5173/setup
2. 查看是否显示 "请先配置 LLM 提供商"
3. 点击 "开始配置" 按钮
4. 确认跳转到 /settings/providers
```

**预期结果**: ✅ 跳转成功

---

### 测试 2: 添加 LLM 提供商

**步骤**:
```
1. 访问 http://localhost:5173/settings/providers
2. 点击 "添加提供商" 按钮
3. 填写表单:
   - 名称：OpenAI
   - API 地址：https://api.openai.com/v1
   - API Key: sk-xxx
   - 模型：gpt-4o-mini
4. 点击 "保存"
5. 确认列表刷新并显示新添加的提供商
```

**预期结果**: ✅ 添加成功

---

### 测试 3: 编辑 LLM 提供商

**步骤**:
```
1. 在提供商列表中点击 "编辑"
2. 修改模型名称（如改为 gpt-4o）
3. 点击 "保存"
4. 确认修改生效
5. 确认 API Key 未被覆盖（仍为脱敏显示）
```

**预期结果**: ✅ 修改成功，API Key 保留

---

### 测试 4: 删除 LLM 提供商

**步骤**:
```
1. 在提供商列表中点击 "删除"
2. 确认删除操作
3. 确认列表刷新并移除该提供商
```

**预期结果**: ✅ 删除成功

---

### 测试 5: 聊天功能

**步骤**:
```
1. 访问 http://localhost:5173/chat
2. 查看是否显示欢迎消息
3. 在输入框输入消息
4. 点击 "发送" 按钮
5. 查看是否显示 Mock 回复
```

**预期结果**: ✅ 消息发送和显示正常

---

## 5. 常见问题

### 问题 1: `pnpm: command not found`

**原因**: 未安装 pnpm 包管理器

**解决方案**:
```bash
# 方式 1: 使用 npm 安装
npm install -g pnpm

# 方式 2: 使用 corepack（Node.js 16+）
corepack enable
corepack prepare pnpm@latest --activate

# 方式 3: 使用 Homebrew（macOS）
brew install pnpm
```

**验证安装**:
```bash
pnpm --version
```

---

### 问题 2: 端口 5173 被占用

**错误信息**:
```
Port 5173 is already in use
```

**解决方案**:

**方式 1**: 使用其他端口
```bash
pnpm run dev --port 3000
```

**方式 2**: 关闭占用端口的进程
```bash
# macOS / Linux
lsof -ti:5173 | xargs kill -9

# Windows
netstat -ano | findstr :5173
taskkill /PID <PID> /F
```

---

### 问题 3: node_modules 缺失

**错误信息**:
```
Could not resolve entry module
```

**解决方案**:
```bash
# 删除 node_modules 和 lock 文件
rm -rf node_modules pnpm-lock.yaml

# 重新安装依赖
pnpm install

# 重新启动
pnpm run dev
```

---

### 问题 4: 构建失败

**错误信息**:
```
Build failed with errors
```

**解决方案**:
```bash
# 1. 清理缓存
rm -rf node_modules dist

# 2. 重新安装
pnpm install

# 3. 重新构建
pnpm run build
```

**常见构建错误**:

| 错误 | 原因 | 解决 |
|------|------|------|
| `Logo is not exported` | 图标不存在 | 使用 `PawOutline` 替代 |
| `t('save') is not defined` | i18n key 错误 | 使用 `t('common.save')` |
| `Cannot find module` | 依赖缺失 | `pnpm install` |

---

### 问题 5: 后端 API 无法访问

**检查步骤**:
```bash
# 1. 确认后端已启动
curl http://localhost:8088/health

# 预期输出：{"status":"ok"}

# 2. 如果后端未启动
cd /Users/luoxiaodou/workspace/projects/claw/GoPaw
./gopaw start
```

**Vite 代理配置**:

确认 `vite.config.ts` 中的代理配置：
```typescript
server: {
  proxy: {
    '/api': {
      target: 'http://localhost:8088',
      changeOrigin: true
    },
    '/ws': {
      target: 'ws://localhost:8088',
      ws: true
    }
  }
}
```

---

### 问题 6: TypeScript 类型错误

**错误信息**:
```
Type 'xxx' is not assignable to type 'yyy'
```

**解决方案**:
```bash
# 运行类型检查
pnpm run type-check

# 如果有类型错误，根据提示修复
# 或者临时跳过类型检查直接构建
npx vite build
```

> ⚠️ 注意：当前 vue-tsc 与 Node.js v24 存在兼容性问题，建议直接运行 `npx vite build` 跳过类型检查。

---

## 6. 开发调试

### 调试工具

**浏览器 DevTools**:
- F12 打开开发者工具
- Console 查看日志
- Network 查看 API 请求
- Application 查看本地存储

**Vue DevTools** (推荐安装):
```
Chrome 扩展：Vue.js devtools
Firefox 扩展：Vue.js devtools
```

---

### 常用调试命令

```bash
# 查看当前运行的进程
ps aux | grep -E "vite|gopaw"

# 停止前端开发服务器
# 在运行 pnpm run dev 的终端按 Ctrl+C

# 查看前端构建产物
cd web && pnpm run build
ls dist/

# 查看依赖包
pnpm list --depth=0

# 更新依赖
pnpm update
```

---

### 日志查看

**前端日志**:
```bash
# 浏览器 Console 中查看
# 或使用命令行查看构建日志
pnpm run dev 2>&1 | tee dev.log
```

**后端日志**:
```bash
# 启动时查看
./gopaw start 2>&1 | tee server.log
```

---

### 性能优化

**开发模式优化**:
```bash
# 使用 --force 强制刷新缓存
pnpm run dev --force

# 使用 --open 自动打开浏览器
pnpm run dev --open
```

**生产构建优化**:
```bash
# 分析构建产物大小
pnpm run build -- --json > stats.json
npx vite-bundle-visualizer stats.json

# 使用 Brotli 压缩
pnpm install -D vite-plugin-compression
```

---

## 📱 快速参考卡片

### 启动命令

```bash
# 仅前端
cd web && pnpm run dev

# 前端 + 后端
# 终端 1: ./gopaw start
# 终端 2: cd web && pnpm run dev
```

### 访问地址

| 服务 | 地址 | 说明 |
|------|------|------|
| 前端开发 | http://localhost:5173 | Vite 开发服务器 |
| 前端生产 | http://localhost:4173 | 生产构建预览 |
| 后端 API | http://localhost:8088 | Go 后端服务 |
| 后端健康 | http://localhost:8088/health | 健康检查 |

### 页面路由

| 页面 | 路由 | 说明 |
|------|------|------|
| 引导页 | `/setup` | LLM 配置检查 |
| 聊天页 | `/chat` | AI 对话界面 |
| LLM 配置 | `/settings/providers` | 提供商管理 |
| Agent 设定 | `/settings/agent` | 系统提示编辑 |
| 频道配置 | `/settings/channels` | 频道管理（开发中） |

---

## 🎯 下一步

1. ✅ 启动前端开发服务器
2. ✅ 访问各个页面测试功能
3. ✅ 测试 LLM 配置流程
4. ✅ 测试聊天功能
5. ⏳ 集成真实后端 API（待后端支持）

---

**文档结束**

*如有问题，请查看 CODE_FIX_REPORT.md 了解最新修复内容*
