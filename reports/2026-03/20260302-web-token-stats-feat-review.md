# Code Review 报告（标准）

**审核日期**: 2026-03-02
**审核者**: Codex (GPT-5)
**被审核报告**: `reports/2026-03/20260302-web-token-stats-feat.md`
**变更类型**: feat
**审核范围**: 会话 Token 统计接口与前端展示

---

## 📊 评分汇总

| 维度 | 得分 | 满分 |
|------|------|------|
| 一：代码质量与规范 | 15 | 25 |
| 二：业务逻辑与架构 | 13 | 25 |
| 三：安全性与数据隐私 | 12 | 15 |
| 四：可测试性 | 6 | 10 |
| 五：工程规范与兼容性 | 6 | 10 |
| 专项（feat）| 8 | 15 |
| **总分** | **60** | **100** |

**合并决策**：⚠️ 整改后合并

---

## 🔍 各维度详细评价

### 维度一：代码质量与规范（15/25）
- **1.1 命名规范（3/5）**：命名清晰，但类型定义出现重复。
- **1.2 可读性与复杂度（3/5）**：统计逻辑可读，仍与聊天流程耦合。
- **1.3 代码复用性 DRY（2/5）**：`SessionStats` 重复定义。  
  定位： [index.ts](/Users/luoxiaodou/workspace/projects/claw/GoPaw/web/src/types/index.ts:72)、[index.ts](/Users/luoxiaodou/workspace/projects/claw/GoPaw/web/src/types/index.ts:82)
- **1.4 错误处理与健壮性（4/5）**：前端失败路径有处理。
- **1.5 魔法值与硬编码（3/5）**：后端仍有 mock 常量返回。

### 维度二：业务逻辑与架构（13/25）
- **2.1 逻辑正确性（4/10）**：统计 UI 可工作，但后端 `GetSessionStats` 仍为 mock，业务正确性不足。  
  定位： [agent.go](/Users/luoxiaodou/workspace/projects/claw/GoPaw/internal/server/handlers/agent.go:152)
- **2.2 架构符合度（5/8）**：接口位置正确，数据层链路未打通。
- **2.3 性能与资源管理（4/7）**：刷新时机覆盖较全。

### 维度三：安全性与数据隐私（12/15）
- **3.1 输入校验（4/5）**：路径参数读取基础可用。
- **3.2 敏感信息处理（4/5）**：统计接口不含敏感字段。
- **3.3 权限控制（4/5）**：未引入新的明显越权面。

### 维度四：可测试性（6/10）
- **4.1 职责单一性（3/5）**：前端页面逻辑聚合偏多。
- **4.2 依赖可注入性（3/5）**：API 层有封装，但 `any` 使用较多。

### 维度五：工程规范与兼容性（6/10）
- **5.1 代码风格一致性（2/4）**：当前后端和前端构建通过。
- **5.2 新引入依赖评估（3/3）**：无新增依赖。
- **5.3 接口与 API 兼容性（1/3）**：接口存在但数据语义仍是 mock，影响对外预期一致性。

### 类型专项（8/15）
#### feat 专项：验收标准覆盖度
- **验收标准逐项确认（4/8）**：前端展示达成，后端“精确统计”尚未落地。
- **接口设计合理性（4/7）**：字段设计合理，需接入真实统计来源。

---

## 🚀 优化建议（按优先级）

### P0 — 必须修复（不得合并）
无。

### P1 — 强烈建议（整改后合并）
1. **[问题]**：统计接口返回 mock 数据，不能反映真实 token 使用。  
   **定位**： [agent.go](/Users/luoxiaodou/workspace/projects/claw/GoPaw/internal/server/handlers/agent.go:152)  
   **建议**：改为调用 `h.mem.GetSessionStats(sessionID)`，补齐错误处理。
2. **[问题]**：`SessionStats` 类型重复定义。  
   **定位**： [index.ts](/Users/luoxiaodou/workspace/projects/claw/GoPaw/web/src/types/index.ts:72)  
   **建议**：删除重复声明，保留单一定义。
3. **[问题]**：API 层大量 `any` 降低类型安全。  
   **定位**： [agent.ts](/Users/luoxiaodou/workspace/projects/claw/GoPaw/web/src/api/agent.ts:7)  
   **建议**：定义明确响应类型替换 `any`。

### P2 — 锦上添花（可遗留，记录到 tech-debt）
1. **[问题]**：`formatTokens` 只支持 `k`。  
   **建议**：扩展 `M` 级显示并统一 locale。

---

## 💡 代码亮点
- 统计刷新挂钩了会话切换与 SSE 完成时机，交互连贯性较好。
