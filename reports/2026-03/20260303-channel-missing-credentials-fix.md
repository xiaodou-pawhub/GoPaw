# [fix] M-2 补充：频道插件凭据缺失错误处理

**报告日期**: 2026-03-03
**开发者**: 小M（AI 助手）
**关联任务**: M-2（ai/handoff-xiaom.md）
**涉及文件数**: 5 个

---

## 问题描述

M-2 任务要求在频道插件缺少必要凭据时，返回统一的 `ErrMissingCredentials` 错误，使得 channel manager 可以区分"未配置"和"配置错误"两种情况，分别打 Info 和 Warn 级别日志。

之前的实现只在 manager 层简化了插件启动逻辑，但没有实现错误类型区分。

---

## 根因分析

1. 缺少统一的 `ErrMissingCredentials` 错误类型定义
2. 各插件的 `Init()` 方法在凭据缺失时返回 `nil` 或普通错误，无法区分
3. `channel.Manager.Start()` 统一打 Warn 日志，无法区分"未配置"和"配置错误"

---

## 实现说明

### 1. 新增错误类型

```go
// pkg/plugin/errors.go
package plugin

import "errors"

// ErrMissingCredentials is returned by Init when required credentials are absent.
// The channel manager treats this as "unconfigured, skip silently" and logs at Info level.
var ErrMissingCredentials = errors.New("missing required credentials")
```

### 2. 修改 channel.Manager.Start()

```go
// internal/channel/manager.go
import "errors"

func (m *Manager) Start(ctx context.Context, pluginCfgs map[string]json.RawMessage) error {
    // ...
    if err := p.Init(cfg); err != nil {
        if errors.Is(err, plugin.ErrMissingCredentials) {
            m.logger.Info("channel not configured, skipping",
                zap.String("plugin", name))
        } else {
            m.logger.Warn("channel plugin init failed, skipping",
                zap.String("plugin", name), zap.Error(err))
        }
        continue
    }
    // ...
}
```

**审查要点**：
- ✅ 使用 `errors.Is()` 匹配错误类型，支持错误包装
- ✅ Info 级别日志表示"未配置，正常跳过"
- ✅ Warn 级别日志表示"配置有误，需要注意"

### 3. 修改各插件 Init() 方法

**飞书**（`internal/platform/feishu/plugin.go`）：
```go
func (p *Plugin) Init(cfg json.RawMessage) error {
    // ...
    if p.cfg.AppID == "" || p.cfg.AppSecret == "" {
        return plugin.ErrMissingCredentials
    }
    // ...
}
```

**钉钉**（`internal/platform/dingtalk/plugin.go`）：
```go
func (p *Plugin) Init(cfg json.RawMessage) error {
    // ...
    if p.cfg.ClientID == "" || p.cfg.ClientSecret == "" {
        return plugin.ErrMissingCredentials
    }
    // ...
}
```

**Webhook**（`internal/platform/webhook/plugin.go`）：
```go
func (p *Plugin) Init(cfg json.RawMessage) error {
    // ...
    if p.cfg.Token == "" {
        return plugin.ErrMissingCredentials
    }
    // ...
}
```

**Console**：无需修改，该插件不需要凭据，始终正常启动。

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `pkg/plugin/errors.go` | 新增 | 定义 `ErrMissingCredentials` |
| `internal/channel/manager.go` | 修改 | 区分错误类型，打不同级别日志 |
| `internal/platform/feishu/plugin.go` | 修改 | Init 返回 `ErrMissingCredentials` |
| `internal/platform/dingtalk/plugin.go` | 修改 | Init 返回 `ErrMissingCredentials` |
| `internal/platform/webhook/plugin.go` | 修改 | Init 返回 `ErrMissingCredentials` |

---

## 自检结果

```bash
go build ./...      ✅ 通过
go vet ./...        ✅ 通过
```

---

## 遗留事项

无

---

## 审查清单

### 代码逻辑

- [ ] `ErrMissingCredentials` 错误类型定义正确
- [ ] `errors.Is()` 匹配逻辑正确
- [ ] 各插件 Init() 在正确条件下返回该错误

### 功能验证

- [ ] 未配置飞书时，日志显示 Info 级别 "channel not configured, skipping"
- [ ] 配置错误时，日志显示 Warn 级别 "channel plugin init failed, skipping"
- [ ] Console 插件正常启动

### 测试覆盖

- [ ] 编译通过
- [ ] 静态检查通过
