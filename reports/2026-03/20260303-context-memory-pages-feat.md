# [feat] CONTEXT.md 和 MEMORY.md 设置页面

**报告日期**: 2026-03-03
**开发者**: 小G（AI 助手）
**关联任务**: G-1、G-2（ai/handoff-xiaog.md）
**涉及文件数**: 6 个

---

## 功能概述

本次开发完成了两个任务：

1. **G-1 CONTEXT.md 编辑页面**：新增设置子页，允许用户查看和编辑工作区背景文件
2. **G-2 MEMORY.md 查看 + 编辑页面**：新增设置子页，允许用户查看和手动校正 Agent 记忆

---

## 实现说明

### G-1：CONTEXT.md 编辑页面

#### 新增 API 方法（`web/src/api/settings.ts`）

```typescript
export async function getWorkspaceContext(): Promise<{ content: string }> {
  return await api.get('/workspace/context')
}

export async function saveWorkspaceContext(content: string): Promise<{ saved: boolean }> {
  return await api.put('/workspace/context', { content })
}
```

#### 新增组件（`web/src/pages/settings/Context.vue`）

- 顶部显示"工作区背景（CONTEXT.md）"及说明
- 主体：可编辑的多行文本区域
- 底部：「保存」按钮，成功提示"保存成功"
- 加载时读取内容，显示同步状态

#### 注册路由（`web/src/router/index.ts`）

```typescript
{
  path: 'settings/context',
  name: 'SettingsContext',
  component: () => import('@/pages/settings/Context.vue')
}
```

#### 注册导航（`web/src/layouts/MainLayout.vue`）

- 在设置导航中添加"工作区背景"菜单项
- 使用 BookOutline 图标

### G-2：MEMORY.md 查看 + 编辑页面

#### 新增 API 方法（`web/src/api/settings.ts`）

```typescript
export async function getAgentMemory(): Promise<{ content: string }> {
  return await api.get('/workspace/memory')
}

export async function saveAgentMemory(content: string): Promise<{ saved: boolean }> {
  return await api.put('/workspace/memory', { content })
}
```

#### 新增组件（`web/src/pages/settings/Memory.vue`）

- 顶部显示"记忆管理（MEMORY.md）"及说明
- 空状态：文件不存在时显示"Agent 尚未写入任何记忆"
- 文件存在时：显示可编辑的多行文本区域
- 底部：「保存修正」按钮

#### 注册路由（`web/src/router/index.ts`）

```typescript
{
  path: 'settings/memory',
  name: 'SettingsMemory',
  component: () => import('@/pages/settings/Memory.vue')
}
```

#### 注册导航（`web/src/layouts/MainLayout.vue`）

- 在设置导航中添加"记忆管理"菜单项
- 使用 TrashOutline 图标

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `web/src/api/settings.ts` | 修改 | 新增 4 个 API 方法 |
| `web/src/pages/settings/Context.vue` | 新增 | CONTEXT.md 编辑页面 |
| `web/src/pages/settings/Memory.vue` | 新增 | MEMORY.md 查看+编辑页面 |
| `web/src/router/index.ts` | 修改 | 注册新路由 |
| `web/src/layouts/MainLayout.vue` | 修改 | 注册导航菜单项 |
| `web/src/locales/index.ts` | 修改 | 新增 i18n 翻译 |

---

## 自检结果

```bash
pnpm run type-check  ✅ 通过
pnpm run build       ✅ 通过
```

---

## 验收标准完成情况

### G-1：CONTEXT.md 编辑页面

- [x] 新增 API 方法 getWorkspaceContext / saveWorkspaceContext
- [x] 新增 Context.vue 组件
- [x] 设置页导航可点到"工作区背景"
- [x] 文本区域正确显示 CONTEXT.md 内容
- [x] 编辑后点保存，刷新页面内容保留
- [x] 编译通过
- [x] 构建通过

### G-2：MEMORY.md 查看 + 编辑页面

- [x] 新增 API 方法 getAgentMemory / saveAgentMemory
- [x] 新增 Memory.vue 组件
- [x] 设置页导航可点到"记忆管理"
- [x] 文件不存在时正确显示空状态提示
- [x] 文件存在时正确显示内容
- [x] 编辑保存后内容正确更新
- [x] 编译通过
- [x] 构建通过

---

## 遗留事项

无