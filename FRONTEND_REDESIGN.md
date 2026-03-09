# GoPaw Frontend 重设计记录

> 本文档记录 GoPaw/web 前端重设计成果，供 gopaw-desk 迁移参考。

## 已完成功能（Phase A-E）

### Phase A：Tailwind CSS v4 + 样式系统
- 移除 Naive UI，改用 Tailwind CSS v4 + CSS 变量
- 深色主题：`#0f1117` 背景 + 紫色强调 `#7c6af7`
- 样式文件：`web/src/styles/globals.css`（CSS 变量 + @theme + Markdown 样式）
- 删除：`global.scss`、`variables.scss`、`design-system.scss`、`page-layout.scss`

### Phase B：三栏布局
- `MainLayout.vue`：TitleBar（38px）+ IconSidebar（48px）+ router-view
- `TitleBar.vue`：`-webkit-app-region: drag`，红绿灯 72px 占位
- `IconSidebar.vue`：Logo + Chat/Market/Settings 图标 + LLM 状态点
- 路由简化为：`/chat/:id?`、`/market`、`/settings`

### Phase C：Chat.vue 重设计
- 深色气泡（用户：紫色半透明，AI：无气泡+头像）
- 会话面板 220px：搜索框、重命名、删除
- Markdown 渲染 + hljs 代码高亮（dark theme）
- 三点动画思考指示器
- 工具调用卡片（calling/done/error 状态）

### Phase D：Settings.vue + 8 个 Tab
- 左侧 180px Tab 导航 + 右侧 defineAsyncComponent 懒加载
- 8 个 Tab：ProvidersTab / AgentTab / ContextTab / MemoryTab
  / ChannelsTab / SkillsTab / CronTab / LogsTab
- 所有 Tab 均已移除 Naive UI，改为原生 HTML + CSS 变量

### Phase E：Market.vue
- 技能市场（即将上线占位 / 卡片网格）
- 移除 Naive UI，改为原生样式

---

## 未实现（待 gopaw-desk 补充）

| 功能 | 说明 |
|------|------|
| PermissionsTab | Layer-2 权限管理（gopaw-desk 独有）|
| RuntimeTab | Python/Node.js 运行时管理（gopaw-desk 独有）|
| PermissionDialog | WebSocket 权限请求弹窗（gopaw-desk 独有）|
| QuickLauncher | ⌥Space 全局浮窗（gopaw-desk 独有）|
| RuntimeProgress | 安装进度条（gopaw-desk 独有）|
| Wails 事件 | navigate/new-chat/quick-launcher（gopaw-desk 独有）|
| Token 认证 | GoPaw/web 有，gopaw-desk 不需要 |

---

## 设计 Token（CSS 变量）

```css
--bg-app:        #0f1117;
--bg-sidebar:    #141418;
--bg-panel:      #18181f;
--bg-elevated:   #1e1e28;
--bg-overlay:    #252532;
--border:        #2a2a38;
--border-subtle: #1e1e2a;
--text-primary:  #f0f0f5;
--text-secondary:#8888a0;
--text-tertiary: #55556a;
--accent:        #7c6af7;
--accent-hover:  #8f7fff;
--accent-dim:    rgba(124,106,247,0.1);
--green:         #22c55e;
--red:           #ef4444;
--yellow:        #f59e0b;
```

## 注意事项

- GoPaw/web 有 Token 认证（`/api/auth/login`），gopaw-desk **不需要**（直连本地）
- GoPaw/web 的 Settings 只有 8 个 Tab，gopaw-desk 需要额外 2 个（Permissions + Runtime）
- 字体：`-apple-system, "SF Pro Text", system-ui, sans-serif`（13px 基础）
- 圆角：8px（小组件）/ 10-12px（卡片）/ 14px（气泡）
