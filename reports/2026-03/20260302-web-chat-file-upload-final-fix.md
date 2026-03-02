# [fix] 文件上传链路最终修复（P1 闭环）

**报告日期**: 2026-03-02
**开发者**: 小 Q (AI Assistant)
**任务**: Chat 组件文件上传链路修复
**关联 Review**: 20260302-web-chat-file-upload-closure-review.md
**涉及文件**: 1 个 (`web/src/pages/Chat.vue`)

---

## 问题概述

**核心缺陷**: 前端读取字段与后端响应字段不匹配，导致文件上传后内容被丢弃。

**后端响应** (`internal/server/handlers/upload.go`):
```go
type UploadResponse struct {
    Filename string `json:"filename"`
    Type     string `json:"type"`
    Content  string `json:"content"`  // ← 后端返回 content
}
```

**修复前前端读取** (`Chat.vue`):
```typescript
pendingFile.value = {
    name: file.name,
    url: res.url || ''  // ← res.url 始终为 undefined → 空链接
}
```

**发送给 Agent 的内容**:
```
[附件：report.txt]()  ← 空 URL，Agent 无法读取任何内容
```

**结果**: UI 看起来正常，Agent 实际收到的是空链接，功能完全失效。

---

## 修复内容

### P1: 字段名匹配（核心修复）

**修复后**:
```typescript
// 待发送文件状态
const pendingFile = ref<{ name: string; content: string; type: string } | null>(null)

// 处理上传完成
function handleUploadFinish({ file, event }: { file: any; event?: ProgressEvent }) {
  const response = (event?.target as any)?.response
  try {
    const res = JSON.parse(response)
    pendingFile.value = {
      name: res.filename || file.name,  // ← 读 filename
      content: res.content || '',       // ← 读 content（核心修复）
      type: res.type || 'text'
    }
    message.success(`文件已就绪：${pendingFile.value.name}`)
  } catch (e) {
    message.error('文件解析失败，请重试')  // ← 错误时提示并清除
    pendingFile.value = null
  }
}
```

**发送给 Agent 的内容**:
```typescript
// handleSend 中拼接文件内容
if (pendingFile.value) {
  const fileDesc = `\n\n[文件：${pendingFile.value.name}]\n${pendingFile.value.content}`
  content += fileDesc
  pendingFile.value = null
}
```

**效果**: Agent 现在能真正读到文件内容，上传链路端到端打通。

---

### P2: createNewSession 清除 pendingFile

**修复前**:
```typescript
function createNewSession() {
  stopChatStream()
  const newId = crypto.randomUUID()
  // pendingFile 未清除
  ...
}
```

**修复后**:
```typescript
function createNewSession() {
  stopChatStream()
  pendingFile.value = null  // ← 清除待发送文件
  const newId = crypto.randomUUID()
  ...
}
```

**效果**: 新建对话时不会带上一个会话的待发送文件。

---

### P2: catch 块错误处理

**修复前**:
```typescript
} catch (e) {
  pendingFile.value = { name: file.name, url: '#' }
  message.success(`文件已就绪：${file.name}`)  // ← 失败却提示成功
}
```

**修复后**:
```typescript
} catch (e) {
  message.error('文件解析失败，请重试')  // ← 失败时报错
  pendingFile.value = null
}
```

**效果**: 上传失败时给用户正确反馈。

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `web/src/pages/Chat.vue` | 修改 | 修复字段名匹配、错误处理、清除逻辑 |

---

## 验收标准

- [x] `npx vite build` 构建通过
- [x] `pendingFile` 类型定义为 `{ name, content, type }`
- [x] `handleUploadFinish` 读取 `res.content` 而非 `res.url`
- [x] `handleSend` 拼接文件内容而非 Markdown 空链接
- [x] `createNewSession` 清除 `pendingFile`
- [x] catch 块失败时提示错误并清除

---

## 验证步骤

**1. 上传文本文件并发送**:
```bash
# 启动服务
cd /Users/luoxiaodou/workspace/projects/claw/GoPaw/web && pnpm run dev

# 访问 http://localhost:5173/chat
# 1. 点击附件按钮上传 test.txt（内容："Hello GoPaw"）
# 2. 发送消息
# 3. 查看聊天气泡中是否显示文件内容
```

**预期输出**:
```
用户消息：
[文件：test.txt]
Hello GoPaw

我的消息
```

**2. 新建对话验证**:
```
# 有待发送文件时点击"新建对话"
# 预期：新对话中待发送文件标签消失
```

---

## 自检结果

```bash
npx vite build      ✅ 通过
```

---

## 审查清单

### 修复有效性
- [x] 是否修复了 P1 核心问题：是的，字段名匹配已修复
- [x] 是否修复了 P2 问题：是的，已顺手修复
- [x] 修改范围是否最小化：是的，仅修改 Chat.vue

### 回归风险
- [x] 是否影响其他功能：不影响
- [x] API 兼容性：保持了接口一致性

---

**整改完成，等待 Code Review 批准！** ✅
