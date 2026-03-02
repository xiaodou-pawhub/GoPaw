# [feat] Web Console 前端全面重构

**报告日期**: 2026-03-02
**开发者**: 小G (AI Assistant)
**关联任务**: G-1, G-2, G-3, G-4, G-5
**涉及文件数**: 12 个

---

## 功能概述

本次重构将 GoPaw 的 Web 前端从初期的“骨架版本”提升为功能完整、界面美观的现代化 AI 助理工作台。

**核心价值：**
- **实时性**：接入 SSE 流式 API，提供极致的打字机对话体验。
- **易用性**：实现了频道密钥的图形化配置与定时任务的可视化管理。
- **专业感**：采用深色主题侧边栏与系统状态监控，提升了产品的整体质感。

---

## 实现说明

### 核心逻辑

1. **SSE 流式集成**：在 `Chat.vue` 中使用原生 `EventSource` 监听 `/api/agent/chat/stream`，实现逐字渲染。
2. **动态配置 (Plan C)**：通过 `api/settings.ts` 读写存储在 SQLite 中的频道密钥，并处理嵌套 JSON 字符串映射。
3. **任务调度管理**：新建 `Cron.vue`，配合后端 `/api/cron` 接口实现定时任务的 CRUD 与即时触发。
4. **统一类型系统**：建立 `types/index.ts` 作为全站数据模型的单一事实来源。

```typescript
// Chat.vue: SSE 核心处理逻辑
const eventSource = new EventSource(url)
eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data)
  if (data.delta) { 
    assistantMsg.content += data.delta // 追加流式内容
    scrollToBottom()
  }
}
```

**审查要点**：
- ✅ SSE 连接在 `done` 或 `error` 时均有明确的 `close()` 处理，防止连接泄漏。
- ✅ 配置保存时采用了 `JSON.stringify` 嵌套处理，符合后端 `channel_configs` 表的设计。
- ⚠️ `Chat.vue` 的会话加载 (TODO) 需后端提供更明细的消息历史接口。

### UI 变化

- **侧边栏**：改为深色调 (`#1a1a2e`)，增加图标。
- **Header**：增加 LLM 配置状态 (Tag) 与暗色模式切换。
- **对话流**：左/右对齐的气泡布局，助手消息支持 Markdown 与代码高亮。

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `web/src/types/index.ts` | 新增 | 统一 TypeScript 接口定义 |
| `web/src/api/agent.ts` | 新增 | 会话管理与流式 URL 生成 |
| `web/src/api/cron.ts` | 新增 | 定时任务 API 封装 |
| `web/src/api/settings.ts` | 修改 | 扩展频道配置读写与健康检查 |
| `web/src/layouts/MainLayout.vue` | 重构 | 全新侧边栏、Header 与状态指示 |
| `web/src/pages/Chat.vue` | 重构 | 实现真实流式对话与气泡 UI |
| `web/src/pages/Cron.vue` | 新增 | 定时任务管理页面 |
| `web/src/pages/settings/Channels.vue` | 重构 | 实现真实配置表单与健康监控 |
| `web/src/pages/settings/Providers.vue` | 优化 | 适配 12px 圆角与新样式 |
| `web/src/pages/settings/Agent.vue` | 优化 | 适配 12px 圆角与新样式 |
| `web/src/pages/Setup.vue` | 优化 | 视觉效果升级 |
| `web/src/locales/index.ts` | 修改 | 补充 Cron 及新功能词条 |

---

## 接口 / API 变更

| 接口 / 函数 | 变更类型 | 是否兼容 | 说明 |
|------------|---------|---------|------|
| `getChatStreamUrl` | 新增 | — | 获取 SSE 地址 |
| `getChannelConfig` | 优化 | ✅ | 增加内部 JSON 解析逻辑 |
| `getCronJobs` | 新增 | — | 获取定时任务列表 |

---

## 验收标准完成情况

- [x] 聊天：SSE 流式显示正常，无 Mock。
- [x] 频道配置：飞书/钉钉/Webhook 密钥可真实保存并回显。
- [x] 定时任务：支持创建、删除与立即触发。
- [x] UI 现代化：侧边栏深色化，内容区 12px 圆角，整体风格统一。
- [x] 响应式：侧边栏支持折叠。

---

## 遗留事项

- 需后端补充“获取特定会话历史消息”接口，以实现完整的历史记录回放。
- Webhook 频道配置页需展示更详细的文档说明。

---

## 审查清单

### 代码逻辑
- [ ] SSE 连接释放逻辑是否严密。
- [ ] 嵌套 JSON (config_json) 的解析是否健壮。

### 视觉与交互
- [ ] 中英双语注释是否覆盖所有新增逻辑。
- [ ] 不同分辨率下的布局适配情况。
- [ ] 保存配置时的 Loading 反馈是否及时。
