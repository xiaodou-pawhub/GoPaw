# Code Review 报告（轻量）

**审核日期**: 2026-03-02
**审核者**: Codex (GPT-5)
**被审核报告**: `reports/2026-03/20260302-memory-token-count-fix.md`
**审核范围**: `Manager.Add()` 为消息写入 `TokenCount` 的修复

---

## P0 问题（不得合并）

无。

---

## P1 问题（建议修复后合并）

1. **缺少针对修复点的回归测试，后续重构时容易再次退化。**
   目前测试主要覆盖 `Store` 与 `tokenizer`，但没有直接验证 `Manager.Add()` 写入后的 `messages.token_count` 非零且与 `GetSessionStats()` 一致。
   **定位**：`/Users/luoxiaodou/workspace/projects/claw/GoPaw/internal/memory/manager.go:50`、`/Users/luoxiaodou/workspace/projects/claw/GoPaw/internal/memory/store_test.go:9`
   **建议**：新增 `manager_test.go`，至少包含 1 个用例：调用 `Manager.Add()` 写入 user/assistant 后，断言 `GetSessionStats()` 的 `total/user/assist` 均大于 0，且 `total = user + assist`。

2. **历史脏数据（旧消息 token_count=0）未处理，会导致存量会话统计仍不准确。**
   本次修复仅覆盖“新写入消息”，对历史数据无修复动作。
   **定位**：`/Users/luoxiaodou/workspace/projects/claw/GoPaw/internal/memory/manager.go:59`
   **建议**：补一个一次性回填脚本/迁移任务，按 `messages.content` 重新计算并更新 `token_count`；至少在发布说明中明确“仅对修复后新增消息生效”。

---

## 💡 代码亮点

- 修复点精准落在根因：`Manager.Add()` 在入库前分别计算 user/assistant token，并写入 `StoredMessage.TokenCount`，与现有 `GetSessionStats()` 汇总逻辑直接闭环，改动小且风险可控。

---

## 结论

⚠️ **整改后合并** — 存在 P1 问题，建议补充回归测试与历史数据处理说明后合并。
