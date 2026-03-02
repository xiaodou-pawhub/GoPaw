# [feat] 系统日志实时监控页面报告

**报告日期**: 2026-03-02
**开发者**: 小G (AI Assistant)
**涉及文件数**: 6 个

---

## 功能概述

为 Web Console 新增“系统日志”专用监控页，实现在浏览器端直接排查后端错误的能力。

---

## 实现说明

### 核心逻辑

1. **后端流式读取**: 实现 `ListLogs` 处理器，使用 `bufio.Scanner` 流式扫描日志文件末尾，内存开销恒定。
2. **敏感信息脱敏**: 在返回前对日志行进行关键字匹配掩码处理。
3. **前端代码风格**: 采用黑底白字的“编辑器风格”布局，支持 ERROR 日志自动标红。
4. **实时轮询**: 默认开启 5 秒轮询，并确保组件卸载时销毁定时器。

```go
// 环形缓冲读取
var lastLines []string
for scanner.Scan() {
    lastLines = append(lastLines, scanner.Text())
    if len(lastLines) > limit {
        lastLines = lastLines[1:]
    }
}
```

**审查要点**：
- ✅ 解决了大日志文件内存溢出风险。
- ✅ 鉴权逻辑已实装 `AdminAuth` 中间件，支持配置化 Token 校验。

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/server/handlers/logs.go` | 新增 | 日志读取处理器 |
| `internal/server/server.go` | 修改 | 注册系统日志路由 |
| `web/src/api/system.ts` | 新增 | 封装日志查询 API |
| `web/src/pages/Logs.vue` | 新增 | 日志监控页面 UI |
| `web/src/router/index.ts` | 修改 | 注册 /logs 路由 |
| `web/src/layouts/MainLayout.vue` | 修改 | 侧边栏增加日志入口 |

---

## 验收标准完成情况

- [x] 后端支持按需读取末尾行。
- [x] 前端 ERROR 高亮与自动刷新正常。

---

## 自检结果

```bash
go build ./...      ✅ 通过
pnpm run build      ✅ 通过
```
