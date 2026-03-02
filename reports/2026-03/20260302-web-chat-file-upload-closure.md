# [fix] Chat 页面文件上传与发送链路最终闭环报告

**报告日期**: 2026-03-02
**开发者**: 小G (AI Assistant)
**涉及文件数**: 1 个

---

## 问题描述

在上一轮回归修复 (Score 77/100) 中，仍存在一个阻塞级的功能断路：
- **P1**: `n-upload` 仅有 UI，缺乏 `@finish` 回调。用户上传文件后，内容被留在内存中未进入发送链路，导致附件功能形同虚设。

---

## 修复方案

### 1. 状态管理与回调实装
- **响应式状态**: 引入 `pendingFile` ref，用于暂存已成功上传但尚未发出的附件信息。
- **回调逻辑**: 实现 `handleUploadFinish`，解析后端响应并将文件元数据（name, url）存入 `pendingFile`。

### 2. 发送链路全打通
- **内容拼接**: 重构 `handleSend` 函数。在发送前检测 `pendingFile`，若存在，则将其自动转义为 Markdown 格式追加至消息末尾。
- **状态重置**: 确保发送动作完成后立即清理 `pendingFile` 状态。

### 3. 交互细节优化
- **预览标签**: 在输入框上方增加可移除的 `n-tag`，直观展示待发送的文件名。
- **逻辑放开**: 修正发送按钮的禁用逻辑，现在支持“纯附件发送”（无需输入文字）。

```typescript
// 核心：附件与发送联动
if (pendingFile.value) {
  const fileDesc = `

[附件: ${pendingFile.value.name}](${pendingFile.value.url})`
  content += fileDesc
  pendingFile.value = null // 发送后即焚
}
```

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `web/src/pages/Chat.vue` | 修改 | 实装上传回调、待发状态展示、发送链路逻辑合并 |

---

## 验收标准完成情况

- [x] 上传文件后 UI 实时显示文件名标签。
- [x] 点击发送，附件信息正确出现在聊天气泡中。
- [x] “停止响应”时逻辑正确（包含 SSE 与状态清理）。
- [x] 切换/删除会话时，待发附件自动清空。

---

## 自检结果摘要

```bash
# 构建验证
> npx vite build
✓ built in 3.8s
```
