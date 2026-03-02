# [feat] 系统日志查看页面报告

**报告日期**: 2026-03-02
**开发者**: 小G (AI Assistant)
**关联任务**: P3-A
**涉及文件数**: 6 个

---

## 功能概述

为 Web Console 增加“系统日志”页面，允许管理员在不通过 SSH 的情况下直接查看后端运行日志，极大提升了排障效率。

---

## 实现说明

### 核心逻辑

1. **后端读取**: 实现 `ListLogs` 处理器，读取 `logs/gopaw.log` 的最后 100 行，并进行逆序处理（最新日志在前）。
2. **前端渲染**: 采用黑底白字的“代码编辑器风格”布局。
3. **级别过滤**: 实现基于正则匹配的日志级别探测，自动为 ERROR 日志添加红色高亮背景。
4. **自动刷新**: 默认开启 5 秒轮询，支持手动开关。

```go
// 后端日志读取核心代码
scanner := bufio.NewScanner(file)
for scanner.Scan() {
    lines = append(lines, LogEntry{Raw: scanner.Text()})
}
// 返回最后 100 行，最新优先
```

**审查要点**：
- ✅ 前端引入了 `onUnmounted` 确保轮询定时器被销毁。
- ✅ 日志页面适配了全屏滚动，解决了长日志布局溢出问题。

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/server/handlers/logs.go` | 新增 | 日志读取处理器 |
| `internal/server/server.go` | 修改 | 注册系统日志路由 |
| `web/src/api/system.ts` | 新增 | 封装日志查询 API |
| `web/src/pages/Logs.vue` | 新增 | 日志查看页面 UI |
| `web/src/router/index.ts` | 修改 | 注册 /logs 路由 |
| `web/src/layouts/MainLayout.vue` | 修改 | 侧边栏增加日志菜单项 |

---

## 验收标准完成情况

- [x] 后端支持读取最后 N 行日志。
- [x] 前端 ERROR 日志高亮显示。
- [x] 自动刷新功能正常工作。

---

## 自检结果

```bash
go build ./...      ✅ 通过
pnpm run build      ✅ 通过
```
