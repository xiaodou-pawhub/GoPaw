# [fix] Memory Token 计数修复

**报告日期**: 2026-03-02
**开发者**: 小M (AI 助手)
**关联任务**: M-3 / handoff-xiaom.md
**涉及文件数**: 1 个

---

## 问题描述

`Manager.Add()` 方法在保存消息时，`TokenCount` 字段始终为零值，导致：
1. 数据库 `messages.token_count` 列存储无效数据
2. `GetSessionStats()` API 返回的 token 统计数据不准确

### 根因分析

原实现在构建 `StoredMessage` 时没有计算和填充 `TokenCount` 字段：

```go
// 原代码（有 bug）
if err := m.store.AddMessage(StoredMessage{
    ID:        uuid.New().String(),
    SessionID: sessionID,
    Role:      "user",
    Content:   userMsg,
    // TokenCount 缺失，为零值
    CreatedAt: now,
}); err != nil { ... }
```

---

## 实现说明

### 修复方案

在 `Add()` 方法中调用已有的 `CountTokens()` 函数计算 token 数，填充到 `StoredMessage.TokenCount`。

```go
func (m *Manager) Add(sessionID, userID, channel, userMsg, assistantMsg string) error {
    if err := m.store.EnsureSession(sessionID, userID, channel); err != nil {
        return fmt.Errorf("memory: ensure session: %w", err)
    }

    now := time.Now().UnixMilli()
    
    // 中文：计算 token 数
    // English: Calculate token count
    userTokens := CountTokens([]MemoryMessage{{Role: "user", Content: userMsg}})
    assistTokens := CountTokens([]MemoryMessage{{Role: "assistant", Content: assistantMsg}})

    if err := m.store.AddMessage(StoredMessage{
        ID:         uuid.New().String(),
        SessionID:  sessionID,
        Role:       "user",
        Content:    userMsg,
        TokenCount: userTokens, // ← 修复：填充计算值
        CreatedAt:  now,
    }); err != nil {
        return fmt.Errorf("memory: add user message: %w", err)
    }

    if err := m.store.AddMessage(StoredMessage{
        ID:         uuid.New().String(),
        SessionID:  sessionID,
        Role:       "assistant",
        Content:    assistantMsg,
        TokenCount: assistTokens, // ← 修复：填充计算值
        CreatedAt:  now + 1,
    }); err != nil {
        return fmt.Errorf("memory: add assistant message: %w", err)
    }
    return nil
}
```

**审查要点**：
- ✅ `CountTokens()` 已在 `internal/memory/tokenizer.go` 实现，基于 tiktoken
- ✅ 分别计算 user 和 assistant 消息的 token 数，数据更精确
- ✅ 无并发问题：`CountTokens()` 是纯函数，无共享状态

### 依赖确认

修改前确认了以下基础设施已存在，无需额外开发：

| 组件 | 文件位置 | 状态 |
|------|---------|------|
| `StoredMessage.TokenCount int` 字段 | `internal/memory/store.go:180` | ✅ 已存在 |
| `messages.token_count` 列 | `internal/memory/store.go:66` DDL | ✅ 已存在 |
| `AddMessage()` INSERT 包含 token_count | `internal/memory/store.go:193-194` | ✅ 已存在 |
| `Store.GetSessionStats()` | `internal/memory/store.go:320` | ✅ 已存在 |
| `Manager.GetSessionStats()` | `internal/memory/manager.go:234` | ✅ 已存在 |
| API 端点 `GET /api/agent/sessions/:id/stats` | `internal/server/server.go:100` | ✅ 已注册 |

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/memory/manager.go` | 修改 | `Add()` 方法增加 token 计算逻辑 |

**变更统计**：新增 6 行 / 修改 4 行 / 删除 0 行

---

## 接口 / API 变更

无。仅修复内部实现，接口签名不变。

---

## 自检结果

```bash
go build ./...      ✅ 通过
go vet ./...        ✅ 通过
go test -race ./internal/memory/...  ✅ 通过（9 个用例全部通过）
```

测试输出：
```
=== RUN   TestStore_CRUD
--- PASS: TestStore_CRUD (0.01s)
=== RUN   TestStore_Summary
--- PASS: TestStore_Summary (0.00s)
=== RUN   TestStore_FilePersistence
--- PASS: TestStore_FilePersistence (0.01s)
=== RUN   TestCountTokens_Empty
--- PASS: TestCountTokens_Empty (0.06s)
=== RUN   TestCountTokens_SimpleEnglish
--- PASS: TestCountTokens_SimpleEnglish (0.00s)
=== RUN   TestCountTokens_MultipleMessages
--- PASS: TestCountTokens_MultipleMessages (0.00s)
=== RUN   TestCountTokens_Fallback
--- PASS: TestCountTokens_Fallback (0.00s)
=== RUN   TestEstimateTokens_Compatibility
--- PASS: TestEstimateTokens_Compatibility (0.00s)
=== RUN   TestCountTokens_Chinese
--- PASS: TestCountTokens_Chinese (0.00s)
PASS
ok      github.com/gopaw/gopaw/internal/memory  1.293s
```

---

## 验收标准完成情况

对照 `handoff-xiaom.md` M-3 任务：

- [x] `Manager.Add()` 填充 `TokenCount` 字段
- [x] 使用 `CountTokens()` 计算 token 数
- [x] API `GET /api/agent/sessions/:id/stats` 返回正确统计（已有实现，本修复使其数据准确）

---

## 遗留事项

无。

---

## 审查清单

### 代码逻辑

- [ ] `CountTokens()` 调用是否正确（参数类型、返回值处理）
- [ ] user 和 assistant 消息是否都正确计算了 token

### 并发 & 错误处理

- [ ] 无共享状态修改，无并发问题
- [ ] 错误处理保持原有链式传递

### 功能验证

- [ ] 新保存的消息 `token_count` 不再为零
- [ ] `GetSessionStats()` 返回的 token 统计数据准确

### 测试覆盖

- [ ] 现有测试是否覆盖 token 计数逻辑
- [ ] `go test -race` 是否通过
