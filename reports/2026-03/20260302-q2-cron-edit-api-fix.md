# [fix] Cron 编辑 API 整改报告（Q-2）

**报告日期**: 2026-03-02
**开发者**: 小 Q (AI Assistant)
**任务**: Q-2: Cron 任务编辑 API（整改）
**关联 Review**: 20260302-q2-cron-edit-api-feat-review.md
**涉及文件**: 2 个

---

## 整改概述

针对 Code Review 提出的 2 个问题进行修复，使用标准的 `errors.Is()` + 哨兵错误模式。

---

## 修复内容

### 问题 1: Cron 表达式校验有副作用

**问题**: 使用 `m.cron.AddFunc()` 校验会产生真实调度 entry，有副作用。

**修复前**:
```go
if _, err := m.cron.AddFunc(*req.CronExpr, func() {}); err != nil {
    return fmt.Errorf("invalid cron expression: %w", err)
}
```

**修复后**:
```go
// 使用临时 cron 实例校验，避免副作用
tmpCron := cron.New()
if _, err := tmpCron.AddFunc(*req.CronExpr, func() {}); err != nil {
    return fmt.Errorf("%w: %q: %v", ErrInvalidCronExpr, *req.CronExpr, err)
}
```

**效果**: 校验不产生真实调度 entry，无副作用。

---

### 问题 2: 无效 Cron 返回 500 而非 400（使用 errors.Is）

**问题**: 通过字符串切片判断错误类型，存在误判/漏判风险。

**修复方案**: 使用 `errors.Is()` + 哨兵错误模式。

**Step 1**: 定义哨兵错误
```go
// internal/scheduler/manager.go
var ErrInvalidCronExpr = errors.New("scheduler: invalid cron expression")
```

**Step 2**: 使用哨兵错误包装
```go
if _, err := tmpCron.AddFunc(*req.CronExpr, func() {}); err != nil {
    return fmt.Errorf("%w: %q: %v", ErrInvalidCronExpr, *req.CronExpr, err)
}
```

**Step 3**: Handler 使用 `errors.Is()` 判断
```go
// internal/server/handlers/cron.go
import "errors"

if err := h.manager.UpdateJob(...); err != nil {
    if errors.Is(err, scheduler.ErrInvalidCronExpr) {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
}
```

**效果**: 使用标准的 `errors.Is()` + 哨兵错误模式，稳定可靠，无误判风险。

---

## 文件变更清单

| 文件 | 变更类型 | 说明 |
|------|---------|------|
| `internal/scheduler/manager.go` | 修改 | 新增哨兵错误 `ErrInvalidCronExpr`，使用临时 cron 校验 |
| `internal/server/handlers/cron.go` | 修改 | 使用 `errors.Is()` 判断错误类型 |

---

## 验收标准

- [x] `go build ./...` 编译通过
- [x] `go vet ./...` 检查通过
- [x] Cron 表达式校验无副作用
- [x] 使用 `errors.Is()` + 哨兵错误模式
- [x] 无效 Cron 返回 400，其他错误返回 500

---

## 自检结果

```bash
go build ./...      ✅ 通过
go vet ./...        ✅ 通过
```

---

## 审查清单

### 修复有效性
- [x] 是否修复了所有 P1 问题：是的，2 个问题均已覆盖
- [x] 修改范围是否最小化：是的，仅限相关 2 个文件

### 回归风险
- [x] 是否影响其他功能：不影响
- [x] API 兼容性：保持了接口一致性

---

**整改完成，等待 Code Review 批准！** ✅
