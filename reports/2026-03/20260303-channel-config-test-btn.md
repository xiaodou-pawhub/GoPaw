# [feat] 频道配置页面增加测试按钮

**报告日期**: 2026-03-03
**开发者**: 小M（AI 助手）
**涉及文件数**: 4 个

---

## 功能概述

在频道配置页面为每个频道（飞书、钉钉、Webhook）添加"测试"按钮，点击后调用后端 `/api/channels/:name/test` 接口，验证频道连通性并显示结果。

---

## 实现说明

### 前端改动

1. **API 函数**（`api/settings.ts`）
   ```typescript
   export async function testChannel(name: string): Promise<{ success: boolean, message: string }> {
     return await api.post(`/channels/${name}/test`)
   }
   ```

2. **状态管理**（`Channels.vue`）
   ```typescript
   const testing = ref<string | null>(null)

   async function testChannelConn(name: string) {
     testing.value = name
     try {
       const result = await testChannel(name)
       if (result.success) {
         message.success(result.message)
       } else {
         message.error(result.message)
       }
     } catch (error: any) {
       message.error(error?.message || t('common.error'))
     } finally {
       testing.value = null
     }
   }
   ```

3. **UI 按钮**（每个频道卡片）
   ```vue
   <n-button round :loading="testing === 'feishu'" @click="testChannelConn('feishu')">
     {{ t('settings.channels.test') }}
   </n-button>
   ```

4. **国际化**（`locales/index.ts`）
   - 中文：`test: '测试'`
   - 英文：`test: 'Test'`

### 后端已有接口

后端 `/api/channels/:name/test` 接口已实现，调用各插件的 `Test()` 方法：
- **飞书**：验证 access_token 是否有效
- **钉钉**：验证 access_token 是否有效
- **Webhook**：发送测试消息到配置的 callback_url

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `web/src/api/settings.ts` | 修改 | 添加 `testChannel` API 函数 |
| `web/src/pages/settings/Channels.vue` | 修改 | 添加测试按钮和 `testChannelConn` 函数 |
| `web/src/locales/index.ts` | 修改 | 添加 `settings.channels.test` 翻译 |

---

## 自检结果

```bash
pnpm run build   ✅ 通过
```

---

## 遗留事项

无

---

## 审查清单

### 功能验证

- [ ] 飞书频道测试按钮显示正确
- [ ] 钉钉频道测试按钮显示正确
- [ ] Webhook 频道测试按钮显示正确
- [ ] 测试成功时显示成功消息
- [ ] 测试失败时显示错误消息
- [ ] 按钮加载状态正确显示
