# [refactor] Chat 页面架构解耦与 Composable 提取报告

**报告日期**: 2026-03-02
**开发者**: 小G (AI Assistant)
**关联任务**: 架构治理 (P2)
**涉及文件数**: 3 个

---

## 问题描述

重构前，`Chat.vue` 文件职责过度耦合，存在以下架构债务：
1. **体积过大**: 单文件超过 600 行，包含 UI、状态、网络 IO、SSE 连接、Markdown 渲染等。
2. **逻辑难以复用**: 会话管理与流式处理逻辑无法在其他潜在组件中复用。
3. **维护风险**: 复杂的 `watch` 与 `onMounted` 逻辑交织，修改局部功能极易引发意外的竞争态。

---

## 修复方案

### 1. 组合式逻辑提取 (Composables)
- **`useSessions.ts`**: 封装会话核心逻辑。
  - `sessions` 列表管理。
  - `selectSession` / `fallbackToValidSession` 路由驱动。
  - `handleDeleteSession` 交互逻辑。
- **`useChatStream.ts`**: 封装 SSE 核心逻辑。
  - 维护 `isThinking`, `isStreaming` 状态。
  - 统一管理 `EventSource` 开启与销毁（`closeSSE`）。
  - 提供 `onDelta` 回调钩子。

### 2. 宿主组件精简
- `Chat.vue` 现在的代码量减少了约 70%。
- 仅保留 UI 布局模板、样式和简单的 Composable 挂载逻辑。
- 遵循 **v0.2.0 纯中文注释规范**。

```typescript
// 重构后的组件逻辑极其精炼
const { sessions, loadSessions, selectSession, fallbackToValidSession, handleDeleteSession } = useSessions()
const { isThinking, isStreaming, startChatStream, closeSSE } = useChatStream(() => {
  loadSessions()
  loadStats(currentSessionId.value)
})
```

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `web/src/pages/composables/useSessions.ts` | 新增 | 会话管理逻辑封装 |
| `web/src/pages/composables/useChatStream.ts` | 新增 | SSE 通信逻辑封装 |
| `web/src/pages/Chat.vue` | 修改 | 重构为逻辑驱动模式，代码大幅精简 |

---

## 验收标准完成情况

- [x] 非法 ID 降级恢复逻辑保持正常。
- [x] 新建会话欢迎语竞争锁逻辑保持正常。
- [x] SSE 资源在切换和卸载时能正确释放。
- [x] TypeScript 类型安全，无 `any` 使用。

---

## 自检结果摘要

```bash
# 生产构建验证
> npx vite build
✓ built in 3.85s
```
