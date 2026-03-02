# [fix] 针对会话管理、统计与日志页面的 CR 问题整改报告

**报告日期**: 2026-03-02
**开发者**: 小G (AI Assistant)
**涉及文件数**: 6 个

---

## 问题描述

在针对会话管理 (N-8)、Token 统计 (N-9) 和系统日志页面 (P3-A) 的复审中，共发现了 1 个 P0 级安全漏洞和 3 个 P1 级架构/逻辑问题：
1. **安全漏洞 (P0)**: `AdminAuth` 在未配置 Token 时会回落到默认硬编码密钥。
2. **错误泄露 (P1)**: 多个 Handler 直接将底层数据库或文件系统错误返回给前端。
3. **架构缺陷 (P1)**: 日志读取路径硬编码，未从配置系统注入。
4. **文档不符 (P1)**: 开发报告中关于“Mock”和“鉴权占位”的描述与实际已实现的真实逻辑不一致。

---

## 修复方案

### 1. 安全加固 (P0)
- 修改 `internal/server/handlers/system.go`: 移除默认密钥回退。若 `AdminToken` 为空，接口将直接返回 403 Forbidden 并记录告警，强制管理员在 `config.yaml` 中配置。

### 2. 错误信息脱敏 (P1)
- 统一重构 `agent.go` 和 `logs.go`: 所有面向客户端的响应均改为统一错误文案（如“获取统计失败”），详细错误仅在服务端通过 `zap.Logger` 记录。

### 3. 日志读取配置化 (P1)
- 将 `ListLogs` 重构为 `SystemHandler` 的方法，通过依赖注入获取 `*config.Config`。
- 动态从 `h.cfg.Log.File` 读取日志路径，增加系统的可维护性。

### 4. 前端交互与报告同步 (P1)
- 重构 `Chat.vue` 的删除逻辑，将职责拆分为更小的函数，并在成功提示中追加短 ID。
- 更新了 `20260302-web-token-stats-feat.md` 和 `20260302-web-system-logs-feat.md`，确保文档描述与代码实现的真实功能（真实统计、已实装鉴权）完全同步。

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/server/handlers/system.go` | 修改 | 修复 AdminAuth 安全漏洞 |
| `internal/server/handlers/agent.go` | 修改 | 错误响应脱敏 |
| `internal/server/handlers/logs.go` | 修改 | 支持配置化路径，错误脱敏 |
| `internal/server/server.go` | 修改 | 路由调用切换 |
| `web/src/pages/Chat.vue` | 修改 | 重构删除逻辑，优化显示 |
| `reports/2026-03/*.md` | 修改 | 同步开发报告描述 |

---

## 验收标准完成情况

- [x] `AdminAuth` 无默认 Token 回退。
- [x] 客户端不再接收到底层 `err.Error()`。
- [x] 日志路径支持 `config.yaml` 动态配置。
- [x] 前端删除交互更健壮，且具备失败兜底提示。

---

## 自检结果摘要

```bash
# 后端编译验证
> go build ./...
Backend build: SUCCESS

# 前端构建验证
> npx vite build
Frontend build: SUCCESS
```
