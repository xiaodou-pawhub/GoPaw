# Code Review 报告（轻量审查）

**审核日期**: 2026-03-02
**审核者**: 小X
**被审核报告**: `reports/2026-03/20260302-web-chat-file-upload-closure.md`
**关联上次审查**: `reports/2026-03/20260302-web-chat-regression-fix-review.md`（77/100，有条件合并）
**变更类型**: fix · 范围：`web/src/pages/Chat.vue`

---

## 📊 评分汇总

| 维度 | 上次 | 本次 | 满分 |
|------|------|------|------|
| 一：代码质量与规范 | 19 | 20 | 25 |
| 二：业务逻辑与架构 | 19 | 17 | 25 |
| 三：安全性与数据隐私 | 13 | 13 | 15 |
| 四：可测试性 | 8 | 8 | 10 |
| 五：工程规范与兼容性 | 8 | 8 | 10 |
| 专项（fix）| 10 | 10 | 15 |
| **总分** | **77** | **76** | **100** |

**合并决策**：⚠️ 有条件合并（整改 1 项 P1 后可合并，无需再次全量 Review）

---

## ✅ 已修复项确认

### 上次 P1：上传链路结构性接通（✅ 已修复）

以下改动正确：

| 改动 | 确认 |
|------|------|
| `pendingFile` ref 状态管理 | ✅ 类型定义清晰，响应式正确 |
| `n-upload` 绑定 `@finish="handleUploadFinish"` | ✅ 回调已注册 |
| `n-tag` 待发文件预览，支持关闭 | ✅ 交互符合预期 |
| 发送按钮禁用逻辑更新：`!inputMessage && !pendingFile` | ✅ 纯附件发送正确放开 |
| `handleSend` 内拼接 `pendingFile` 并发送后清除 | ✅ 发送后即焚逻辑正确 |
| `resetCurrentSessionState` 清除 `pendingFile` | ✅ 删除会话后清理正确 |

---

## 🔴 新发现阻塞问题

### P1：前端读取字段与后端响应字段不匹配（必须修复）

**这是本次整改的核心语义缺陷**，上传链路的"最后一公里"未打通。

**后端实际响应**（`internal/server/handlers/upload.go`）：

```go
type UploadResponse struct {
    Filename string `json:"filename"`
    Type     string `json:"type"`    // "text" or "image"
    Content  string `json:"content"` // 文本内容 or base64 图片
}
// 无 url 字段
```

**前端实际读取**（`Chat.vue:203-212`）：

```typescript
const res = JSON.parse(response)
pendingFile.value = {
    name: file.name,
    url: res.url || ''   // ← res.url 始终为 undefined → url = ''
}
```

**发送给 Agent 的内容**（`Chat.vue:286`）：

```typescript
const fileDesc = `\n\n[附件: ${pendingFile.value.name}](${pendingFile.value.url})`
// 实际输出：[附件: report.txt]()   ← 空 URL，Agent 无法读取任何内容
```

**结果**：Agent 收到的是一个空链接，不是文件内容。用户以为发送了文件，Agent 实际什么都没收到。UI 看起来正常，行为完全错误。

**整改方案**（改动量极小）：

```typescript
// pendingFile 结构改为存 content，不存 url
const pendingFile = ref<{ name: string; content: string; type: string } | null>(null)

function handleUploadFinish({ file, event }: { file: any; event?: ProgressEvent }) {
  const response = (event?.target as any)?.response
  try {
    const res = JSON.parse(response)
    pendingFile.value = {
      name: res.filename || file.name,
      content: res.content,   // ← 读 content，不读 url
      type: res.type || 'text'
    }
    message.success(`文件已就绪：${pendingFile.value.name}`)
  } catch {
    message.error('文件上传失败，请重试')   // 解析失败应报错，不应伪装成功
    pendingFile.value = null
  }
}

// handleSend 中改为内联内容，让 Agent 真正读到文件
if (pendingFile.value) {
  if (pendingFile.value.type === 'image') {
    // 图片：直接嵌入 base64（部分 LLM 支持视觉输入时使用）
    content += `\n\n[图片附件: ${pendingFile.value.name}]`
  } else {
    // 文本文件：将内容直接附加，Agent 可读
    content += `\n\n[文件: ${pendingFile.value.name}]\n${pendingFile.value.content}`
  }
  pendingFile.value = null
}
```

---

## 🟡 P2 问题（可遗留）

### P2-1：`createNewSession()` 未清除 `pendingFile`

**定位**：`Chat.vue:265`

```typescript
function createNewSession() {
    stopChatStream()
    // pendingFile 未清除
    const newId = crypto.randomUUID()
    ...
}
```

用户有未发送附件时点击"新建对话"，`pendingFile` 仍保留，下一条消息会把上一个会话的附件带走。建议加一行 `pendingFile.value = null`。

---

### P2-2：catch 兜底给出误导性成功提示

**定位**：`Chat.vue:210-213`

```typescript
} catch (e) {
    pendingFile.value = { name: file.name, url: '#' }
    message.success(`文件已就绪: ${file.name}`)  // ← 失败了却提示"成功"
}
```

JSON 解析失败通常意味着上传请求本身出错（后端返回 HTML 错误页等）。这种情况不应给出"已就绪"的成功反馈。整改 P1 时顺手改为 `message.error(...)` + `pendingFile.value = null`。

---

## 🧭 整改指引

本次修改量极小，只涉及 `Chat.vue` 约 15 行：

1. `pendingFile` 的类型定义：`url` → `content` + `type`
2. `handleUploadFinish`：读 `res.content` 替代 `res.url`；catch 改为报错
3. `handleSend` 的拼接逻辑：由 Markdown 链接改为内联文件内容
4. `createNewSession`：加 `pendingFile.value = null`（P2，顺手做）

整改后**无需再次提交 Review**，由人工确认以下两点后直接合并：
- 上传 `.txt` 文件后发送，聊天气泡中可见文件内容
- 新建对话后，附件标签已消失

---

## 总结

本次整改完成了结构性接通（状态管理、回调注册、UI 联动），工程质量有明显提升。阻塞点是 `res.url` / `res.content` 的字段名错配——这是一个未核对后端接口文档直接写代码导致的低级失误，修复量极小但影响是核心功能完全失效。修复字段名后，整个上传链路即可端到端跑通。
