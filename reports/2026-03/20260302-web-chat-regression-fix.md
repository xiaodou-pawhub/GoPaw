# [fix] Chat 组件重构功能退化与协议漏洞紧急修复报告

**报告日期**: 2026-03-02
**开发者**: 小G (AI Assistant)
**涉及文件数**: 2 个

---

## 问题描述

在上一轮架构重构 (Score 56/100) 中，由于疏忽引发了严重的质量倒退：
1. **安全漏洞 (P0)**: 流式通信回退至 GET 模式，敏感内容通过 URL 参数传输，且受长度限制。
2. **功能丢失 (P0)**: 误删了附件上传入口和停止响应按钮。
3. **交互缺陷 (P1)**: 删除按钮未阻止事件冒泡，引发路由误跳转。

---

## 修复方案

### 1. 协议拨乱反正 (P0)
- **POST 流式实装**: 在 `useChatStream.ts` 中废弃 `EventSource`，改用自定义 `fetch` 实现。
- **Body 传输**: 确保 `content` 放在请求体中发送，遵循生产环境安全规范。
- **手动中断**: 集成 `AbortController`，使用户能通过 UI 主动停止助手响应。

### 2. 核心功能回归 (P0)
- **UI 恢复**: 重新在 `Chat.vue` 输入区域上方挂载附件上传（`n-upload`）和停止按钮。
- **逻辑重连**: 确保附件入口在非发送状态下可见，停止按钮在发送状态下可见。

### 3. 交互漏洞修复 (P1)
- **冒泡拦截**: 在删除按钮上实装 `@click.stop`，彻底隔绝列表点击事件的影响。

```typescript
// 核心：基于 fetch 的 POST 流式处理
const response = await fetch('/api/agent/chat/stream', {
  method: 'POST',
  body: JSON.stringify({ session_id, content }),
  signal: abortController.signal
})
```

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `web/src/pages/composables/useChatStream.ts` | 修改 | 重构为 POST 协议，增加中断支持 |
| `web/src/pages/Chat.vue` | 修改 | 恢复附件/停止 UI，修复冒泡，对接新流式接口 |

---

## 验收标准完成情况

- [x] 流式对话通过 POST 发送，Body 包含消息内容。
- [x] “停止响应”功能正常，能即时切断 SSE。
- [x] 附件上传 UI 入口已找回。
- [x] 点击删除按钮不再导致路由跳转。

---

## 自检结果摘要

```bash
# 构建验证
> npx vite build
✓ built in 3.9s
```
