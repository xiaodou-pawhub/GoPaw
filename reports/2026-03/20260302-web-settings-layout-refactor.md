# [refactor] 设置页面沉浸式布局与视觉重构报告

**报告日期**: 2026-03-02
**开发者**: 小G (AI Assistant)
**关联任务**: G-4
**涉及文件数**: 5 个

---

## 问题描述

重构前，GoPaw 的设置模块存在以下设计缺陷：
1. **导航混乱**: 设置子项直接平铺在主侧边栏，导致主导航臃肿。
2. **风格老旧**: 典型的“后台管理系统”风格，大量使用厚重的边框和标准的表单平铺，缺乏现代产品的精致感。
3. **空间利用率低**: 在宽屏下配置项分布散乱，视觉焦点不集中。

---

## 修复方案

### 1. 沉浸式二级布局 (Architecture)
- **`SettingsLayout.vue`**: 新增设置专用容器。左侧为浅灰色背景的二级菜单，右侧为纯白色内容区。
- **路由重组**: 在 `router/index.ts` 中将 `providers`, `channels`, `agent` 全部嵌套至 `SettingsLayout` 下。
- **主侧边栏瘦身**: 将主导航中的设置项收拢为一个“设置”单入口。

### 2. 视觉语言重塑 (Visual Design)
- **大排版**: 统一应用 32px 的 Extra-bold 标题，增加描述性文字，强化“配置实验室”的氛围。
- **卡片进化**: `Providers.vue` 废弃表格，改用带有 `transition` 动画和 `box-shadow` 呼吸感的磁贴布局。
- **品牌化**: `Channels.vue` 为每个平台设计了专属的渐变色图标块，提升辨识度。
- **交互回馈**: 为 Agent 编辑器增加了状态指示灯（Green/Orange），直观显示同步状态。

### 3. 微动效与细节
- **平滑切换**: 切换子路由时应用 `fade-slide` 动画。
- **圆角规范**: 全局应用 16px - 24px 的大圆角，符合现代审美。

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `web/src/layouts/SettingsLayout.vue` | 新增 | 设置模块二级导航容器 |
| `web/src/router/index.ts` | 修改 | 重组设置模块路由结构 |
| `web/src/layouts/MainLayout.vue` | 修改 | 精简主侧边栏菜单 |
| `web/src/pages/settings/Providers.vue` | 修改 | 重构为高级感卡片流布局 |
| `web/src/pages/settings/Agent.vue` | 修改 | 重构为文档编辑器风格 |
| `web/src/pages/settings/Channels.vue` | 修改 | 重构为品牌分栏布局 |

---

## 验收标准完成情况

- [x] 设置模块具备独立的二级导航。
- [x] 成功移除“中台式”陈旧设计，UI 具备高级感。
- [x] 所有配置项保存、删除逻辑回归正常。
- [x] 生产构建零错误。

---

## 自检结果摘要

```bash
# 构建验证
> npx vite build
✓ built in 3.92s
```
