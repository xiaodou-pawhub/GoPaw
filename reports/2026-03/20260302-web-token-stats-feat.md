# [feat] 会话 Token 统计展示报告

**报告日期**: 2026-03-02
**开发者**: 小G (AI Assistant)
**关联任务**: N-9
**涉及文件数**: 4 个

---

## 功能概述

在聊天界面实时展示当前会话的 Token 消耗情况，帮助用户监控 LLM 使用量并理解上下文压缩状态。

---

## 实现说明

### 核心逻辑

1. **Mock 接口**: 由于小M尚未完成 M-3，当前在后端提供了一个固定的 stats 响应。
2. **前端仪表盘**: 在聊天窗口 Header 右侧增加统计徽章。
3. **数字格式化**: 实现 `formatTokens` 工具函数，将 `1500` 转换为 `1.5k`，提升可读性。
4. **实时刷新**: 切换会话、发送消息、SSE 响应结束时均会自动触发统计更新。

```typescript
// Token 格式化逻辑
function formatTokens(n: number): string {
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return n.toString()
}
```

**审查要点**：
- ✅ 统计刷新时机覆盖了所有关键对话节点。
- ✅ 增加了 Tooltip 详情展示，不仅看总额，还能看分布。

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/server/handlers/agent.go` | 修改 | 新增 GetSessionStats Mock 接口 |
| `web/src/types/index.ts` | 修改 | 添加 SessionStats 接口定义 |
| `web/src/api/agent.ts` | 修改 | 封装统计 API 调用 |
| `web/src/pages/Chat.vue` | 修改 | 实现统计展示 UI |

---

## 验收标准完成情况

- [x] 成功展示 Token 统计徽章。
- [x] 实现 1k+ 格式化逻辑。
- [x] 实现会话切换时的自动更新。

---

## 自检结果

```bash
pnpm run type-check ✅ 通过
```
