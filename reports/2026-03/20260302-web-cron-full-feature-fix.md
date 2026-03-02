# [fix] 定时任务 (Cron) 复审问题闭环整改报告

**报告日期**: 2026-03-02
**开发者**: 小G (AI Assistant)
**涉及文件数**: 3 个

---

## 问题描述

在针对定时任务联调的复审 (Score 76/100) 中，存在以下核心缺陷：
1. **逻辑矛盾 (P1)**: 活跃时间窗口名义上可选，但默认值 `09:00-18:00` 导致所有新任务被强制限时执行。
2. **校验缺陷 (P1)**: `handleSubmit` 未执行 `form.validate()`，导致提交逻辑未形成闭环。
3. **架构瑕疵 (P2)**: API 层仍残留 `any` 类型，且存在多处硬编码文案。

---

## 修复方案

### 1. 时间窗口可选逻辑闭环 (P1)
- **状态重置**: 将 `active_from/active_until` 初始值改为 `null`。
- **动态提交**: 在 `handleSubmit` 中增加转换逻辑，仅当用户显式通过时间选择器设定值时才发送，否则传空字符串告知后端不限时。

### 2. 强制表单校验 (P1)
- **同步拦截**: 在提交流程首行调用 `await formRef.value?.validate()`，校验失败时立即中断请求。

### 3. 错误透传与类型安全 (P1/P2)
- **排障优化**: 改进 `catch` 块，优先展示后端返回的 JSON 错误描述。
- **消除 any**: 为 `getCronJobs`、`getCronRunHistory` 等 API 补全了精确的泛型接口定义。
- **i18n 标准化**: 补全 `locales/index.ts` 中的词条，将页面文案完全国际化。

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `web/src/api/cron.ts` | 修改 | 彻底消除 any，补全响应类型 |
| `web/src/locales/index.ts` | 修改 | 补全 cron 模块国际化词条 |
| `web/src/pages/Cron.vue` | 修改 | 修复校验逻辑、时间窗默认值及文案硬编码 |

---

## 验收标准完成情况

- [x] 表单提交前强制执行校验。
- [x] 新建任务不再默认携带活跃时间窗。
- [x] 成功透传后端错误详情。
- [x] API 层实现 TypeScript 类型闭环。

---

## 自检结果摘要

```bash
# 后端编译验证
> go build ./...
Backend build: SUCCESS

# 前端构建验证
> pnpm run build
✓ built in 3.9s
Frontend build: SUCCESS
```
