# [feat] 会话 Token 统计展示报告

**报告日期**: 2026-03-02
**开发者**: 小G (AI Assistant)
**关联任务**: N-9
**涉及文件数**: 4 个

---

## 功能概述

在聊天界面实时展示当前会话的 Token 消耗分布，帮助用户监控 LLM 额度使用情况并了解上下文压缩状态。

---

## 实现说明

### 核心逻辑

1. **Mock 适配**: 后端提供 `GetSessionStats` 接口（当前为 Mock 实现），返回消息数、用户/助手/总 Token 量。
2. **前端仪表盘**: 在聊天窗口 Header 右侧集成 `n-tooltip` 带详情的统计徽章。
3. **数字格式化**: 封装 `formatTokens` 函数，将大数字转换为 `1.2k` 格式，优化视觉体验。
4. **触发时机**: 在切换会话、消息发送及流式响应结束 (done) 时自动刷新统计数据。

```typescript
// 统计刷新核心
async function loadStats(id: string) {
  sessionStats.value = await getSessionStats(id)
}
```

**审查要点**：
- ✅ 统计刷新逻辑与 SSE 状态机深度解耦。
- ✅ 增加了 `SessionStats` 统一类型定义。

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/server/handlers/agent.go` | 修改 | 新增 GetSessionStats 接口 |
| `web/src/types/index.ts` | 修改 | 添加 SessionStats 类型定义 |
| `web/src/api/agent.ts` | 修改 | 封装统计 API 调用 |
| `web/src/pages/Chat.vue` | 修改 | 实现统计展示 UI |

---

## 验收标准完成情况

- [x] 成功展示 Token 统计徽章。
- [x] 实现 1k+ 数字格式化显示。
- [x] 切换会话自动同步统计信息。

---

## 自检结果

```bash
pnpm run type-check ✅ 通过
```
