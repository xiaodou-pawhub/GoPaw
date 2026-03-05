# GoPaw 工具重构实施路线图

## 一、Phase 1: 核心工具完善 (1-2 周) ⭐⭐⭐⭐⭐

### 任务清单

#### 1.1 网络工具增强
- [ ] `web_fetch` - 抓取网页内容 (High Priority)
  - 参考: PicoClaw web.go, CoPaw browser_control.py
  - 依赖: net/http, golang.org/x/net/html
  - 预估: 2 天

- [ ] `grep_search` - 内容搜索 (High Priority)
  - 参考: CoPaw file_search.py
  - 预估: 1 天

- [ ] `http_file_download` - 文件下载 (Medium Priority)
  - 预估: 1 天

#### 1.2 文件工具增强
- [ ] `send_file_to_user` - 发送文件 (High Priority)
  - 参考: CoPaw send_file.py
  - 预估: 2 天

- [ ] `file_append` - 追加文件 (Low Priority)
  - 预估: 0.5 天

- [ ] `file_delete` - 删除文件 (Low Priority)
  - 预估: 0.5 天

- [ ] `file_exists` - 检查文件存在 (Low Priority)
  - 预估: 0.5 天

#### 1.3 定时任务增强
- [ ] `cron_add` - 添加定时任务 (Medium Priority)
  - 预估: 2 天

- [ ] `cron_list` - 列出定时任务 (Medium Priority)
  - 预估: 1 天

- [ ] `cron_remove` - 删除定时任务 (Medium Priority)
  - 预估: 1 天

#### 1.4 进程与系统工具
- [ ] `process_execute` - 进程管理 (Medium Priority)
  - 预估: 3 天

- [ ] `get_current_time` - 获取当前时间 (Low Priority)
  - 预估: 0.5 天

---

## 二、Phase 2: 接口优化 (2 周) ⭐⭐⭐⭐

### 任务清单

#### 2.1 ToolResult 结构化
- [ ] 创建 `pkg/plugin/tool_result.go`
  - 预估: 1 天

- [ ] 更新所有现有工具
  - 预估: 3 天

- [ ] 更新 Executor
  - 预估: 2 天

#### 2.2 上下文支持
- [ ] 实现 `ContextualTool` 接口
  - 预估: 1 天

- [ ] 适配消息工具
  - 预估: 2 天

#### 2.3 异步工具支持
- [ ] 实现 `AsyncTool` 接口
  - 预估: 1 天

- [ ] 子 Agent 异步支持
  - 预估: 2 天

---

## 三、Phase 3: 浏览器自动化 (2-3 周) ⭐⭐⭐⭐⭐

### 任务清单

#### 3.1 浏览器核心功能
- [ ] `browser_use` - 浏览器控制 (High Priority)
  - 参考: CoPaw browser_control.py (2,231 行)
  - 预估: 10 天

- [ ] `browser_navigate` - 导航工具 (High Priority)
  - 预估: 1 天

#### 3.2 截图与媒体
- [ ] `desktop_screenshot` - 桌面截图 (Medium Priority)
  - 预估: 2 天

---

## 四、Phase 4: 文档处理 (2-3 周) ⭐⭐⭐⭐

### 任务清单

#### 4.1 PDF 处理
- [ ] `pdf_read` - PDF 读取 (Medium Priority)
  - 预估: 5 天

- [ ] `pdf_extract_images` - 提取图片 (Low Priority)
  - 预估: 2 天

#### 4.2 Word 处理
- [ ] `docx_read` - Word 读取 (Medium Priority)
  - 预估: 5 天

#### 4.3 Excel 处理
- [ ] `xlsx_read` - Excel 读取 (Medium Priority)
  - 预估: 5 天

---

## 五、Phase 5: 子 Agent 管理 (1 周) ⭐⭐⭐⭐

### 任务清单

- [ ] `subagent_list` - 列出子 Agent (Medium Priority)
  - 预估: 1 天

- [ ] `subagent_kill` - 终止子 Agent (Medium Priority)
  - 预估: 1 天

- [ ] `subagent_manage` - 管理子 Agent (Medium Priority)
  - 预估: 2 天

- [ ] `subagent_spawn` - 创建子 Agent (Medium Priority)
  - 预估: 3 天

---

## 六、Phase 6: 安全系统 (1 周) ⭐⭐⭐⭐⭐

### 任务清单

#### 6.1 执行审批
- [ ] `exec_approvals` - 执行审批系统 (High Priority)
  - 参考: OpenClaw exec-approvals.ts
  - 预估: 5 天

#### 6.2 权限控制
- [ ] `tool_permissions` - 工具权限 (Medium Priority)
  - 预估: 2 天

---

## 七、Phase 7: 集成与测试 (1 周) ⭐⭐⭐⭐⭐

### 任务清单

#### 7.1 测试覆盖
- [ ] 单元测试 (High Priority)
  - 预估: 3 天

- [ ] 集成测试 (High Priority)
  - 预估: 2 天

- [ ] E2E 测试 (Medium Priority)
  - 预估: 2 天

---

## 八、里程碑检查点

### Milestone 1 (Week 4): Phase 1+2 完成
```
✅ 核心工具 >= 20 个
✅ 工具接口统一 (ToolResult)
✅ 上下文支持 (ContextualTool)
✅ 异步工具 (AsyncTool)
```

### Milestone 2 (Week 8): Phase 1-3 完成
```
✅ browser_use 可用
✅ 核心文档工具 pdf/docx/xlsx
✅ 定时任务管理完整
```

### Milestone 3 (Week 12): Phase 1-5 完成
```
✅ 子 Agent 管理完整
✅ 执行审批系统上线
✅ 测试覆盖率 >70%
```

### Milestone 4 (Week 15): v0.2.0 Release
```
✅ 所有 P0/P1/P2 任务完成
✅ 文档完善
✅ E2E 测试通过
✅ 性能达标
```

---

## 九、风险管理

### 风险 1: 浏览器工具复杂度高 ⚠️⚠️⚠️
- **影响**: 预计 10 天，可能延长到 15 天
- **缓解**: 先实现核心功能，使用 Playwright Go Client

### 风险 2: 文档工具依赖外部库 ⚠️⚠️
- **影响**: PDF/Office 库稳定性
- **缓解**: 使用成熟库，完整单元测试

### 风险 3: 安全系统设计复杂 ⚠️⚠️⚠️
- **影响**: 执行审批需要完整权限模型
- **缓解**: 参考 OpenClaw，先实现白名单

### 风险 4: 测试覆盖不足 ⚠️⚠️
- **影响**: 代码质量不达标
- **缓解**: 每个工具至少 3 个测试用例

---

## 十、成功标准

### 代码质量
- [ ] 单元测试覆盖率 >70%
- [ ] Lint 无错误
- [ ] TypeCheck 无错误
- [ ] 性能测试通过

### 功能完整性
- [ ] 核心工具 >= 25 个
- [ ] 所有 P0/P1 任务完成
- [ ] 安全系统上线
- [ ] 文档完善

### 用户体验
- [ ] 工具文档完整
- [ ] 示例代码丰富
- [ ] 错误消息友好
- [ ] 性能达标 (P95 < 5s)

### 可维护性
- [ ] 代码结构清晰
- [ ] 注释完整
- [ ] 接口稳定
- [ ] 扩展性好
