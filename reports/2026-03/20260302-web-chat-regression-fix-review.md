# Code Review 报告（二次审查）

**审核日期**: 2026-03-02
**审核者**: 小X
**被审核报告**: `reports/2026-03/20260302-web-chat-regression-fix.md`
**关联首次审查**: `reports/2026-03/20260302-web-chat-component-refactor-review.md`（56/100，不得合并）
**变更类型**: fix（针对首次审查整改）

---

## 📊 评分汇总

| 维度 | 首次 | 本次 | 满分 |
|------|------|------|------|
| 一：代码质量与规范 | 18 | 19 | 25 |
| 二：业务逻辑与架构 | 11 | 19 | 25 |
| 三：安全性与数据隐私 | 8 | 13 | 15 |
| 四：可测试性 | 7 | 8 | 10 |
| 五：工程规范与兼容性 | 7 | 8 | 10 |
| 专项（fix）| 5 | 10 | 15 |
| **总分** | **56** | **77** | **100** |

**合并决策**：⚠️ 有条件合并（整改 1 项 P1 后可合并）

---

## ✅ 已修复问题逐项确认

### P0-1：流式协议回退（已修复 ✅）

**定位**：`web/src/pages/composables/useChatStream.ts:33`

```typescript
// 修复后：直接复用 sendChatStream（POST + body + AbortController）
await sendChatStream(sessionId, content, { onDelta, onDone, onError }, { signal: abortController.signal })
```

确认：
- `useChatStream.ts` 不再引用 `EventSource` 或 `getChatStreamUrl`，全链路走 `sendChatStream`
- `sendChatStream`（`agent.ts:63`）使用 `fetch POST`，`content` 放在 body，无 URL 长度限制
- `AbortController` 正确传入 `signal`，中断调用路径完整

**安全维度**：`content` 不再出现在 URL / 代理日志，安全边界恢复。

---

### P0-2：文件上传入口（部分修复 ⚠️）

**定位**：`web/src/pages/Chat.vue:107-114`

```html
<n-upload v-if="!isStreaming" action="/api/agent/upload" :show-file-list="false">
  <n-button quaternary circle size="small">
    <template #icon><n-icon :component="AttachOutline" /></template>
  </n-button>
</n-upload>
<n-button v-else ... @click="stopChatStream">停止</n-button>
```

确认：
- 上传按钮 UI 已恢复，停止响应按钮也已找回 ✅
- `v-if="!isStreaming"` 切换逻辑正确 ✅

**但存在遗留问题（见下方 P1-1）**：`n-upload` 无 `@finish` / `@success` 回调，文件上传后内容未进入发送链路。

---

### P1：删除按钮冒泡（已修复 ✅）

**定位**：`web/src/pages/Chat.vue:29`

```html
<n-button @click.stop="() => handleDeleteSession(session.id, resetCurrentSessionState)">
```

确认：`.stop` 修饰符正确添加，删除操作不再触发 `selectSession`。

---

## 🔍 本次新发现问题

### P1-1：文件上传内容未接入发送链路（必须整改）

**定位**：`web/src/pages/Chat.vue:107`

`<n-upload>` 仅配置了 `action="/api/agent/upload"` 和 `:show-file-list="false"`，没有 `@finish` 或 `@success` 事件处理器。上传成功后：
1. 后端返回的文件内容（`{ filename, type, content }`）被丢弃
2. 用户发送消息时，文件内容不会附带进去
3. 用户没有任何视觉反馈（文件是否上传成功）

首次审查 P0-2 的核心要求是"**附件随消息发送能力**"恢复，当前只恢复了 UI，行为仍缺失。

**整改方案**（最小改动）：

```typescript
// 在 <script setup> 里增加：
const pendingFile = ref<{ name: string; content: string } | null>(null)

function handleUploadFinish({ file, event }: { file: any; event: ProgressEvent }) {
  try {
    const res = JSON.parse((event.target as XMLHttpRequest).responseText)
    pendingFile.value = { name: res.filename, content: res.content }
    message.success(`已附加文件：${res.filename}`)
  } catch {
    message.error('文件解析失败')
  }
}
```

```html
<!-- n-upload 加上回调 -->
<n-upload
  v-if="!isStreaming"
  action="/api/agent/upload"
  :show-file-list="false"
  @finish="handleUploadFinish"
>
```

```typescript
// handleSend 里拼接文件内容
async function handleSend() {
  let content = inputMessage.value
  if (pendingFile.value) {
    content = `[附件: ${pendingFile.value.name}]\n${pendingFile.value.content}\n\n${content}`
    pendingFile.value = null
  }
  // ...后续发送逻辑不变
}
```

---

### P2-1：`console.error` 替换（可遗留）

**定位**：
- `useChatStream.ts:56`：`console.error('Chat stream failed:', err)`
- `useSessions.ts`：`console.error('Failed to load sessions:', error)`

项目规范要求使用 `useMessage()` 给用户反馈，或直接 `throw`，不使用 `console.error`。

---

### P2-2：删除会话时 UI 重置早于 API 调用（可遗留）

**定位**：`useSessions.ts` `handleDeleteSession`

```typescript
if (isCurrent) {
  onCurrentDeleted()          // UI 先清空
}
await apiDeleteSession(id)    // API 后调用——若失败，UI 已不可恢复
```

若 `apiDeleteSession` 网络失败，UI 状态已被清空，但会话实际未删除。后续 `loadSessions()` 会把会话重新加载回列表，但消息区仍是空白。建议将 `onCurrentDeleted()` 移到 `await apiDeleteSession(id)` 之后。

---

## 🧭 合并建议

| 级别 | 问题 | 处理方式 |
|------|------|---------|
| **P1（整改后合并）** | 文件上传内容未接入发送链路 | 加 `@finish` 回调 + `handleSend` 拼接内容 |
| P2（可遗留） | `console.error` 未替换 | 记录技术债，下次重构一并处理 |
| P2（可遗留） | 删除会话 UI 重置时序问题 | 记录技术债 |

**流程**：整改 P1-1 后，本 fix 可合并，无需再次全量 Review，由人工确认上传链路可用即可。

---

## 总结

本次整改完成度良好：核心安全问题（协议回退）和交互缺陷（事件冒泡）均已彻底修复，得分从 56 提升至 77。唯一阻塞点是文件上传的后半段逻辑（内容接入发送），修改量小，补上即可合并。
