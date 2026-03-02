# [refactor] 聊天会话路由持久化重构报告

**报告日期**: 2026-03-02
**开发者**: 小G (AI Assistant)
**关联任务**: G-1
**涉及文件数**: 2 个

---

## 问题描述

重构前，聊天页面 `/chat` 是静态路径，存在以下体验缺陷：
1. **状态易失**: 用户刷新浏览器后，当前选中的会话 ID 丢失，页面回退到初始状态。
2. **无法分享**: 无法通过特定的 URL 直接定位到某个历史会话。
3. **交互副作用**: 删除、切换等逻辑直接操作本地状态变量，缺乏统一的状态驱动源。

---

## 修复方案

### 1. 路由配置升级
在 `router/index.ts` 中将聊天路径重构为可选参数模式：
`path: 'chat/:id?'`

### 2. 路由驱动的架构重构
重写 `Chat.vue` 的核心逻辑，使其遵循“URL 即状态”的原则：
- **初始化**: `onMounted` 优先读取 `route.params.id`。
- **切换**: 点击列表项不再直接修改变量，而是触发 `router.push`。
- **降级恢复**: 在 `handleSessionSwitch` 中增加了非法 ID 检测逻辑。若访问不存在的会话 ID，系统会自动重定向（`router.replace`）到有效的首个会话或触发新建会话。
- **竞争修复**: 引入 `isCreatingNew` 状态标记，解决了新建会话时前端生成的欢迎语被 `watch` 触发的空历史加载请求意外覆盖的问题。
- **注释规范**: 彻底清理了残留的英文标签，全站遵循纯中文注释。

```typescript
// 降级恢复核心
try {
  const history = await getSessionMessages(id)
  messages.value = history
} catch (error) {
  if (!sessions.value.some(s => s.id === id)) {
    fallbackToValidSession() // 自动重定向到有效状态
  }
}
```

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `web/src/router/index.ts` | 修改 | 支持可选 ID 参数的聊天路由 |
| `web/src/pages/Chat.vue` | 修改 | 重构整个生命周期与数据加载逻辑 |

---

## 验收标准完成情况

- [x] 刷新页面后能保持当前会话上下文。
- [x] 手动输入 `/chat/invalid-id` 能正确降级处理。
- [x] 删除当前会话后路由正确跳转。
- [x] 全站注释符合 v0.2.0 纯中文规范。

---

## 自检结果摘要

```bash
# 前端构建验证
> pnpm run build
✓ built in 3.8s
```
