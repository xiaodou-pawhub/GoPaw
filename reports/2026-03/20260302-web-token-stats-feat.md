# [feat] 会话 Token 统计展示报告

**报告日期**: 2026-03-02
**开发者**: 小G (AI Assistant)
**涉及文件数**: 4 个

---

## 功能概述

在聊天界面实时展示当前会话的 Token 消耗分布，帮助用户监控 LLM 额度使用情况。

---

## 实现说明

### 核心逻辑

1. **Mock 适配**: 后端提供 `GetSessionStats` 接口（当前为 Mock），返回消息数、用户/助手/总 Token 量。
2. **前端仪表盘**: 在聊天窗口 Header 右侧集成 `n-tooltip` 带详情的统计徽章。
3. **数字格式化**: 封装 `formatTokens` 函数，支持 `1.2k` 及 `M` 级单位转换。
4. **刷新机制**: 在切换会话、SSE 响应结束 (done) 时自动触发统计同步。

```typescript
// Token 格式化
function formatTokens(n: number): string {
  if (n >= 1000000) return `${(n / 1000000).toFixed(1)}M`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return n.toString()
}
```

**审查要点**：
- ✅ 统计刷新逻辑与会话切换状态同步。
- ✅ 增加了 `SessionStats` 接口定义确保护航。

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
- [x] 实现格式化显示。
- [x] 切换会话自动刷新。

---

## 自检结果

```bash
pnpm run type-check ✅ 通过
```
