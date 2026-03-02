# [feat] 系统日志实时查看页面报告

**报告日期**: 2026-03-02
**开发者**: 小G (AI Assistant)
**关联任务**: P3-A
**涉及文件数**: 6 个

---

## 功能概述

为 Web Console 新增“系统日志”专用页面，实现在浏览器端直接监控后端运行状态的能力，减少对 SSH 命令行的依赖。

---

## 实现说明

### 核心逻辑

1. **后端读取**: 实现 `ListLogs` 处理器，使用 `bufio.Scanner` 流式扫描 `logs/gopaw.log`，通过切片操作仅提取末尾 100 行，确保内存开销极低。
2. **脱敏保护**: 在返回前扫描内容，对包含 `api_key`、`secret`、`token` 等关键字的日志行执行掩码处理。
3. **前端渲染**: 采用黑底白字的“代码编辑器风格”布局，符合运维直觉。
4. **实时轮询**: 默认开启 5 秒自动刷新，并实现了 `onUnmounted` 资源自动销毁。

```go
// 健壮的日志读取片段
scanner := bufio.NewScanner(file)
for scanner.Scan() {
    lastLines = append(lastLines, scanner.Text())
    if len(lastLines) > limit {
        lastLines = lastLines[1:] // 环形缓冲
    }
}
```

**审查要点**：
- ✅ 实现了高效的“尾部读取”逻辑，避免了大日志文件崩溃风险。
- ✅ 前端 ERROR 日志通过正则匹配自动实现红色高亮。

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/server/handlers/logs.go` | 新增 | 实现日志流式读取与脱敏 |
| `internal/server/server.go` | 修改 | 注册系统日志路由 |
| `web/src/api/system.ts` | 新增 | 封装日志查询 API |
| `web/src/pages/Logs.vue` | 新增 | 系统日志 UI 与自动刷新逻辑 |
| `web/src/router/index.ts` | 修改 | 注册 /logs 路由 |
| `web/src/layouts/MainLayout.vue` | 修改 | 侧边栏增加日志菜单项 |

---

## 验收标准完成情况

- [x] 后端支持读取最后 N 行日志。
- [x] 前端 ERROR 日志高亮。
- [x] 自动/手动刷新功能正常工作。

---

## 自检结果

```bash
go build ./...      ✅ 通过
pnpm run build      ✅ 通过
```
