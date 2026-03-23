# 流程编排与工作流合并设计文档

## 一、背景与目标

### 1.1 问题分析

GoPaw 原本存在两个功能相似的概念：

| 功能 | 定位 | 问题 |
|------|------|------|
| **Workflow（工作流）** | 自动化任务执行 | 功能单一，仅支持线性步骤 |
| **Orchestration（编排）** | 多 Agent 协作对话 | 与工作流概念重叠，用户难以区分 |

**主要问题：**
1. 概念模糊：用户不清楚何时使用工作流、何时使用编排
2. 功能重复：两者都有节点、连线、执行引擎
3. 维护成本：两套代码需要分别维护
4. 学习成本：用户需要理解两套不同的概念

### 1.2 合并目标

1. **统一概念**：合并为单一的"流程"概念
2. **功能增强**：吸收两者的优点，形成更强大的功能
3. **简化使用**：降低用户学习成本
4. **便于扩展**：新节点类型可插拔

## 二、合并方案

### 2.1 核心设计

以 Orchestration 的图结构为基础，吸收 Workflow 的触发机制和执行能力，形成统一的"流程"概念。

```
┌─────────────────────────────────────────────────────────────┐
│                        Flow（流程）                          │
├─────────────────────────────────────────────────────────────┤
│  类型: conversation（对话流）| task（任务流）                │
│  触发: manual | cron | webhook | event                      │
│  节点: Start | Agent | Human | Condition | Parallel | ...   │
│  执行: 对话模式（有状态）| 任务模式（无状态）                 │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 流程类型

| 类型 | 说明 | 特点 |
|------|------|------|
| **conversation** | 对话流 | 支持人工介入、状态保持、多轮对话 |
| **task** | 任务流 | 自动化执行、支持触发器、无状态 |

### 2.3 与原功能对比

| 特性 | 原 Workflow | 原 Orchestration | 新 Flow |
|------|-------------|------------------|---------|
| 数据结构 | 线性步骤 + depends_on | 图结构（节点 + 边） | 图结构 |
| 人工介入 | ❌ | ✅ | ✅ |
| 条件分支 | 简单条件 | 多种条件类型 | 多种条件类型 |
| 并行执行 | ✅ | ❌ | ✅ |
| 循环执行 | ❌ | ❌ | ✅ |
| 触发器 | cron/webhook | ❌ | ✅ |
| 子流程嵌套 | ❌ | ✅ | ✅ |

## 三、数据结构设计

### 3.1 核心模型

```go
// Flow 流程定义
type Flow struct {
    ID          string
    Name        string
    Description string
    Type        FlowType       // conversation/task
    Definition  FlowDefinition
    Trigger     *TriggerConfig
    Status      FlowStatus
}

// FlowDefinition 流程定义结构
type FlowDefinition struct {
    Nodes       []FlowNode
    Edges       []FlowEdge
    Variables   map[string]Variable
    StartNodeID string
}

// FlowNode 流程节点
type FlowNode struct {
    ID       string
    Type     NodeType
    Name     string
    AgentID  string
    Role     string
    Prompt   string
    Config   map[string]interface{}
    Position Position
}

// FlowEdge 流程连线
type FlowEdge struct {
    ID         string
    Source     string
    Target     string
    Label      string
    Condition  *EdgeCondition
    Transform  *Transform
}
```

### 3.2 节点类型

| 节点类型 | 说明 | 分类 | 使用场景 |
|----------|------|------|----------|
| **Start** | 流程起点 | 基础 | 流程开始时执行 |
| **Agent** | 调用数字员工 | 基础 | 需要 AI 处理、工具调用、决策时 |
| **Human** | 等待人工输入 | 基础 | 需要人工审核、选择、补充信息时 |
| **Condition** | 条件分支 | 控制 | 意图识别、结果判断、状态检查 |
| **Parallel** | 并行执行 | 控制 | 多个独立任务需要并行处理 |
| **Loop** | 循环执行 | 控制 | 需要迭代处理、重试机制 |
| **SubFlow** | 子流程 | 高级 | 复用已有流程、模块化设计 |
| **Webhook** | 等待外部事件 | 高级 | 需要外部系统回调、异步等待 |
| **End** | 流程终点 | 基础 | 流程结束时执行 |

### 3.3 条件类型

| 条件类型 | 说明 | 示例 |
|----------|------|------|
| **expression** | 表达式判断 | `{{score}} > 80` |
| **intent** | 意图匹配 | 用户输入包含"查询"、"订单" |
| **llm** | LLM 判断 | 让 LLM 根据上下文判断分支 |

## 四、使用指南

### 4.1 典型场景

#### 场景一：客服对话流程

```
用户输入 → Agent(意图识别) → Condition(意图判断)
                                    ↓
                    ┌───────────────┼───────────────┐
                    ↓               ↓               ↓
              Agent(查询)     Agent(投诉处理)   Agent(闲聊)
                    ↓               ↓               ↓
                    └───────────────┴───────────────┘
                                    ↓
                              End(输出结果)
```

**适用：** 多意图分支的对话场景

#### 场景二：审批流程

```
用户提交 → Agent(初审) → Condition(是否需要人工)
                              ↓
                    ┌─────────┴─────────┐
                    ↓                   ↓
              Human(审批)          Agent(自动通过)
                    ↓                   ↓
                    └─────────┬─────────┘
                              ↓
                        Agent(结果通知)
                              ↓
                          End(结束)
```

**适用：** 需要人工介入的审批场景

#### 场景三：数据处理流程

```
开始 → Agent(数据获取) → Parallel(并行处理)
                              ↓
                    ┌─────────┼─────────┐
                    ↓         ↓         ↓
              Agent(清洗) Agent(分析) Agent(可视化)
                    ↓         ↓         ↓
                    └─────────┼─────────┘
                              ↓
                        Agent(汇总报告)
                              ↓
                          End(结束)
```

**适用：** 需要并行处理的任务流

#### 场景四：重试流程

```
开始 → Loop(重试循环) → Agent(执行任务) → Condition(执行结果)
          ↑                                          ↓
          └────────────────────────────────── 失败 ─┤
                                                   ↓
                                              End(成功结束)
```

**适用：** 需要重试机制的任务

### 4.2 节点配置说明

#### Agent 节点

```yaml
节点名称: 意图识别
Agent: agent_intent
角色描述: 负责识别用户意图
Prompt 模板: 请分析以下用户输入的意图：{{input}}
```

#### Human 节点

```yaml
节点名称: 人工审批
提示模板: 请审批以下申请：{{content}}
快捷选项: [同意, 拒绝, 需要更多信息]
超时: 3600 秒
```

#### Condition 节点

```yaml
节点名称: 意图判断
条件类型: intent
意图关键词: 查询,订单,投诉,闲聊
```

#### Loop 节点

```yaml
节点名称: 重试循环
循环条件: {{success}} == false
最大循环次数: 3
```

## 五、技术实现

### 5.1 目录结构

```
internal/flow/
├── models.go      # 数据模型定义
├── service.go     # 服务层（CRUD）
├── engine.go      # 执行引擎
└── (未来扩展)
    ├── triggers/  # 触发器实现
    └── validators/ # 流程验证器

web/src/components/flow/
├── FlowDesigner.vue       # 可视化设计器
├── nodes/                 # 节点组件
│   ├── StartNode.vue
│   ├── AgentNode.vue
│   ├── HumanNode.vue
│   ├── ConditionNode.vue
│   ├── ParallelNode.vue
│   ├── LoopNode.vue
│   ├── SubFlowNode.vue
│   ├── WebhookNode.vue
│   └── EndNode.vue
└── properties/            # 属性面板
    └── NodeProperties.vue
```

### 5.2 API 接口

```
GET    /api/flows              # 列出流程
POST   /api/flows              # 创建流程
GET    /api/flows/:id          # 获取流程
PUT    /api/flows/:id          # 更新流程
DELETE /api/flows/:id          # 删除流程
POST   /api/flows/:id/execute  # 执行流程
POST   /api/flows/:id/activate # 激活流程
POST   /api/flows/:id/deactivate # 停用流程

GET    /api/flows/:id/executions      # 列出执行记录
GET    /api/flows/executions/:execId  # 获取执行记录
POST   /api/flows/executions/:execId/continue # 继续执行（人工节点后）

GET    /api/flows/node-types    # 获取节点类型列表
```

### 5.3 执行流程

```
┌─────────────┐
│  Execute()  │ ← 创建执行实例
└──────┬──────┘
       ↓
┌─────────────┐
│  runFlow()  │ ← 执行循环
└──────┬──────┘
       ↓
┌─────────────┐
│ executeNode │ ← 执行当前节点
└──────┬──────┘
       ↓
   ┌───┴───┐
   │ 节点类型?
   └───┬───┘
       │
   ┌───┼───────────────────────────────┐
   ↓   ↓           ↓                   ↓
Agent Human    Condition            Parallel
   │   │           │                   │
   │   ↓           │                   │
   │ 等待人工输入   │               并行执行
   │   │           │                   │
   └───┴───────────┴───────────────────┘
       ↓
┌─────────────┐
│ selectNext  │ ← 选择下一个节点
└──────┬──────┘
       ↓
   ┌───┴───┐
   │ 有下一个? │
   └───┬───┘
       │
   ┌───┼───┐
   ↓       ↓
  是      否
   │       │
   │       ↓
   │   ┌─────────────┐
   │   │  Completed  │
   │   └─────────────┘
   │
   └──→ 继续执行循环
```

## 六、迁移指南

### 6.1 数据迁移

原有数据可通过以下方式迁移：

```sql
-- Workflow → Flow
INSERT INTO flows (id, name, description, type, definition, status)
SELECT 
    id,
    name,
    description,
    'task',
    -- 转换 definition 格式
    json_object(
        'nodes', -- 转换 steps 为 nodes
        'edges', -- 根据 depends_on 生成 edges
        'start_node_id', 'start_1'
    ),
    status
FROM workflows;

-- Orchestration → Flow
INSERT INTO flows (id, name, description, type, definition, status)
SELECT 
    id,
    name,
    description,
    'conversation',
    definition,
    status
FROM orchestrations;
```

### 6.2 API 兼容

原有 API 保持兼容，通过重定向实现：

```javascript
// router/index.ts
{
  path: '/workflows',
  redirect: '/flows'
},
{
  path: '/orchestrations',
  redirect: '/flows'
}
```

## 七、未来规划

### 7.1 短期计划

- [ ] 流程版本管理
- [ ] 流程导入/导出
- [ ] 更多触发器类型（事件触发）
- [ ] 流程执行监控面板

### 7.2 长期计划

- [ ] 流程调试器（断点、单步执行）
- [ ] 流程性能分析
- [ ] AI 辅助流程设计
- [ ] 流程市场（分享和复用）

## 八、总结

本次合并实现了以下目标：

1. **概念统一**：用户只需理解"流程"一个概念
2. **功能增强**：支持对话流和任务流两种模式
3. **易于使用**：可视化设计器 + 预置模板
4. **便于扩展**：节点类型可插拔，触发器可扩展

---

**文档版本：** v1.0  
**更新日期：** 2026-03-23  
**作者：** GoPaw Team