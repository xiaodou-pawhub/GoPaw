package orchestration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gopaw/gopaw/internal/agent"
	"github.com/gopaw/gopaw/internal/agent/message"
	"github.com/gopaw/gopaw/internal/workflow"
	"github.com/gopaw/gopaw/pkg/types"
	"go.uber.org/zap"
)

// Engine 编排执行引擎
type Engine struct {
	DB          *sql.DB  // 公开字段，供 service 使用
	agentMgr    *agent.Manager
	agentRouter *agent.Router
	msgMgr      *message.Manager
	workflowEng *workflow.Engine
	executions  map[string]*ExecutionContext
	mu          sync.RWMutex
	logger      *zap.Logger
}

// ExecutionContext 执行上下文
type ExecutionContext struct {
	ID             string
	Orchestration  *Orchestration
	Status         string
	CurrentNodeID  string
	Variables      map[string]interface{}
	Messages       []ExecutionMessage
	StartTime      time.Time
	UpdateTime     time.Time
	HumanInputChan chan string // 人工输入通道
}

// NodeResult 节点执行结果
type NodeResult struct {
	Content string
	Status  string // success/failed/waiting
	Data    map[string]interface{}
}

// NewEngine 创建编排引擎
func NewEngine(
	db *sql.DB,
	agentMgr *agent.Manager,
	agentRouter *agent.Router,
	msgMgr *message.Manager,
	workflowEng *workflow.Engine,
	logger *zap.Logger,
) *Engine {
	return &Engine{
		DB:          db,
		agentMgr:    agentMgr,
		agentRouter: agentRouter,
		msgMgr:      msgMgr,
		workflowEng: workflowEng,
		executions:  make(map[string]*ExecutionContext),
		logger:      logger,
	}
}

// Execute 执行编排
func (e *Engine) Execute(ctx context.Context, orchID string, input string, variables map[string]interface{}) (*ExecutionContext, error) {
	// 1. 加载编排定义
	orch, err := e.loadOrchestration(orchID)
	if err != nil {
		return nil, fmt.Errorf("failed to load orchestration: %w", err)
	}

	// 2. 创建执行上下文
	execCtx := &ExecutionContext{
		ID:             generateExecutionID(),
		Orchestration:  orch,
		Status:         "running",
		Variables:      variables,
		Messages:       make([]ExecutionMessage, 0),
		StartTime:      time.Now(),
		UpdateTime:     time.Now(),
		HumanInputChan: make(chan string, 1),
	}

	if execCtx.Variables == nil {
		execCtx.Variables = make(map[string]interface{})
	}

	// 保存输入到变量
	execCtx.Variables["input"] = input

	// 3. 保存执行记录到数据库
	if err := e.saveExecution(execCtx); err != nil {
		return nil, err
	}

	// 4. 存储执行上下文
	e.mu.Lock()
	e.executions[execCtx.ID] = execCtx
	e.mu.Unlock()

	// 5. 异步执行编排
	go e.runExecution(context.Background(), execCtx, input)

	return execCtx, nil
}

// runExecution 运行编排执行
func (e *Engine) runExecution(ctx context.Context, execCtx *ExecutionContext, input string) {
	defer func() {
		// 只有在运行中或暂停状态时才标记为完成
		if execCtx.Status == "running" || execCtx.Status == "paused" {
			execCtx.Status = "completed"
		}
		execCtx.UpdateTime = time.Now()
		e.saveExecution(execCtx)
		close(execCtx.HumanInputChan)
	}()

	// 获取起始节点
	currentNode := e.getStartNode(execCtx.Orchestration)
	if currentNode == nil {
		execCtx.Status = "failed"
		execCtx.Variables["error"] = "no start node found"
		return
	}

	// 设置初始输入
	currentInput := input

	for currentNode != nil {
		execCtx.CurrentNodeID = currentNode.ID
		e.saveExecution(execCtx)

		// 执行当前节点
		result, err := e.executeNode(ctx, execCtx, currentNode, currentInput)
		if err != nil {
			execCtx.Status = "failed"
			execCtx.Variables["error"] = err.Error()
			e.logger.Error("node execution failed", zap.Error(err), zap.String("node_id", currentNode.ID))
			return
		}

		// 如果节点等待人工输入，暂停执行
		if result.Status == "waiting" {
			execCtx.Status = "paused"
			e.saveExecution(execCtx)
			
			// 等待人工输入
			select {
			case humanInput := <-execCtx.HumanInputChan:
				result.Content = humanInput
				result.Status = "success"
				execCtx.Status = "running"
			case <-ctx.Done():
				execCtx.Status = "failed"
				return
			}
		}

		// 保存节点输出到变量
		execCtx.Variables[currentNode.ID+"_output"] = result.Content
		currentInput = result.Content

		// 根据结果和连线决定下一个节点
		nextNode, err := e.determineNextNode(execCtx.Orchestration, currentNode, result, execCtx)
		if err != nil {
			execCtx.Status = "failed"
			execCtx.Variables["error"] = err.Error()
			return
		}

		if nextNode == nil {
			// 没有下一个节点，执行结束
			execCtx.Variables["output"] = result.Content
			break
		}

		currentNode = nextNode
	}
}

// executeNode 执行节点
func (e *Engine) executeNode(ctx context.Context, execCtx *ExecutionContext, node *OrchestrationNode, input string) (*NodeResult, error) {
	switch node.Type {
	case "agent":
		return e.executeAgentNode(ctx, execCtx, node, input)
	case "human":
		return e.executeHumanNode(ctx, execCtx, node, input)
	case "condition":
		return e.executeConditionNode(ctx, execCtx, node, input)
	case "workflow":
		return e.executeWorkflowNode(ctx, execCtx, node, input)
	case "end":
		return &NodeResult{Content: input, Status: "success"}, nil
	default:
		return nil, fmt.Errorf("unknown node type: %s", node.Type)
	}
}

// executeAgentNode 执行 Agent 节点
func (e *Engine) executeAgentNode(ctx context.Context, execCtx *ExecutionContext, node *OrchestrationNode, input string) (*NodeResult, error) {
	// 1. 获取或创建 Agent 实例
	agentInstance, err := e.agentRouter.GetOrCreateAgent(node.AgentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent: %w", err)
	}

	// 2. 构建输入（结合角色 Prompt 和上下文）
	prompt := node.Prompt
	if prompt != "" {
		input = prompt + "\n\n" + input
	}

	// 3. 调用 Agent 处理
	response, err := agentInstance.Process(ctx, &types.Request{
		Content:   input,
		SessionID: execCtx.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("agent processing failed: %w", err)
	}

	// 4. 记录消息
	e.recordMessage(execCtx, node.ID, "", "response", response.Content)

	return &NodeResult{
		Content: response.Content,
		Status:  "success",
	}, nil
}

// executeHumanNode 执行人工节点
func (e *Engine) executeHumanNode(ctx context.Context, execCtx *ExecutionContext, node *OrchestrationNode, input string) (*NodeResult, error) {
	// 构建提示信息
	prompt := node.Prompt
	if prompt != "" {
		prompt = e.replaceVariables(prompt, execCtx.Variables)
	} else {
		prompt = input
	}

	// 记录等待人工输入的消息
	e.recordMessage(execCtx, node.ID, "", "human_wait", prompt)

	// 返回等待状态
	return &NodeResult{
		Content: prompt,
		Status:  "waiting",
		Data: map[string]interface{}{
			"prompt": prompt,
		},
	}, nil
}

// executeConditionNode 执行条件节点
func (e *Engine) executeConditionNode(ctx context.Context, execCtx *ExecutionContext, node *OrchestrationNode, input string) (*NodeResult, error) {
	// 条件节点本身不处理内容，只是作为分支判断的标记
	return &NodeResult{
		Content: input,
		Status:  "success",
		Data: map[string]interface{}{
			"input": input,
		},
	}, nil
}

// executeWorkflowNode 执行工作流节点
func (e *Engine) executeWorkflowNode(ctx context.Context, execCtx *ExecutionContext, node *OrchestrationNode, input string) (*NodeResult, error) {
	// 获取工作流 ID
	workflowID, ok := node.Config["workflow_id"].(string)
	if !ok || workflowID == "" {
		return nil, fmt.Errorf("workflow_id not configured")
	}

	// 构建输入
	var workflowInput map[string]interface{}
	if mapping, ok := node.Config["input_mapping"].(map[string]interface{}); ok {
		workflowInput = make(map[string]interface{})
		for key, value := range mapping {
			if strValue, ok := value.(string); ok {
				workflowInput[key] = e.replaceVariables(strValue, execCtx.Variables)
			} else {
				workflowInput[key] = value
			}
		}
	} else {
		workflowInput = map[string]interface{}{"input": input}
	}

	// 执行工作流
	execution, err := e.workflowEng.Execute(workflowID, workflowInput, "orchestration")
	if err != nil {
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	// 等待工作流完成（简化实现，实际应该异步）
	// 这里简化处理，直接返回
	return &NodeResult{
		Content: fmt.Sprintf("Workflow executed: %s", execution.ID),
		Status:  "success",
		Data: map[string]interface{}{
			"workflow_execution_id": execution.ID,
		},
	}, nil
}

// determineNextNode 决定下一个节点
func (e *Engine) determineNextNode(orch *Orchestration, currentNode *OrchestrationNode, result *NodeResult, execCtx *ExecutionContext) (*OrchestrationNode, error) {
	// 获取从当前节点出发的所有连线
	edges := e.getOutgoingEdges(orch, currentNode.ID)

	for _, edge := range edges {
		// 检查条件
		if edge.Condition != nil {
			match, err := e.evaluateCondition(edge.Condition, result, execCtx)
			if err != nil {
				return nil, err
			}
			if !match {
				continue // 条件不匹配，尝试下一条连线
			}
		}

		// 找到匹配的连线，返回目标节点
		return e.getNodeByID(orch, edge.Target), nil
	}

	// 没有匹配的连线，执行结束
	return nil, nil
}

// evaluateCondition 评估条件
func (e *Engine) evaluateCondition(cond *EdgeCondition, result *NodeResult, execCtx *ExecutionContext) (bool, error) {
	switch cond.Type {
	case "expression":
		return e.evaluateExpression(cond.Expression, result, execCtx)
	case "intent":
		return e.matchIntent(cond.Intent, result.Content)
	case "llm":
		return e.llmJudge(cond.LLMQuery, result.Content)
	default:
		return true, nil // 无条件，默认通过
	}
}

// evaluateExpression 评估表达式
func (e *Engine) evaluateExpression(expr string, result *NodeResult, execCtx *ExecutionContext) (bool, error) {
	// 简单的表达式评估，支持 {{variable}} 语法
	expr = e.replaceVariables(expr, execCtx.Variables)

	// 支持简单的比较：==、!=、contains
	if strings.Contains(expr, "==") {
		parts := strings.SplitN(expr, "==", 2)
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			return left == right, nil
		}
	}

	if strings.Contains(expr, "!=") {
		parts := strings.SplitN(expr, "!=", 2)
		if len(parts) == 2 {
			left := strings.TrimSpace(parts[0])
			right := strings.TrimSpace(parts[1])
			return left != right, nil
		}
	}

	if strings.Contains(expr, "contains") {
		re := regexp.MustCompile(`(.+)\s+contains\s+(.+)`)
		matches := re.FindStringSubmatch(expr)
		if len(matches) == 3 {
			return strings.Contains(matches[1], matches[2]), nil
		}
	}

	// 默认返回 true
	return true, nil
}

// matchIntent 匹配意图
func (e *Engine) matchIntent(intent, content string) (bool, error) {
	content = strings.ToLower(content)
	intent = strings.ToLower(intent)

	switch intent {
	case "确认", "同意", "是的", "对":
		return strings.Contains(content, "确认") || strings.Contains(content, "同意") ||
			strings.Contains(content, "是的") || strings.Contains(content, "对"), nil
	case "取消", "拒绝", "不", "否":
		return strings.Contains(content, "取消") || strings.Contains(content, "拒绝") ||
			strings.Contains(content, "不") || strings.Contains(content, "否"), nil
	case "咨询", "询问", "问题":
		return strings.Contains(content, "?") || strings.Contains(content, "？") ||
			strings.Contains(content, "咨询") || strings.Contains(content, "询问"), nil
	default:
		return strings.Contains(content, intent), nil
	}
}

// llmJudge LLM 判断
func (e *Engine) llmJudge(query, content string) (bool, error) {
	// 简化实现，实际应该调用 LLM 进行判断
	// 这里使用简单的关键词匹配作为示例
	return true, nil
}

// replaceVariables 替换变量
func (e *Engine) replaceVariables(template string, variables map[string]interface{}) string {
	re := regexp.MustCompile(`\{\{(\w+)\}\}`)
	return re.ReplaceAllStringFunc(template, func(match string) string {
		key := match[2 : len(match)-2] // 去掉 {{ 和 }}
		if value, ok := variables[key]; ok {
			return fmt.Sprintf("%v", value)
		}
		return match
	})
}

// getStartNode 获取起始节点
func (e *Engine) getStartNode(orch *Orchestration) *OrchestrationNode {
	if orch.Definition.StartNodeID != "" {
		return e.getNodeByID(orch, orch.Definition.StartNodeID)
	}
	// 如果没有指定起始节点，返回第一个节点
	if len(orch.Definition.Nodes) > 0 {
		return &orch.Definition.Nodes[0]
	}
	return nil
}

// getNodeByID 根据 ID 获取节点
func (e *Engine) getNodeByID(orch *Orchestration, nodeID string) *OrchestrationNode {
	for i := range orch.Definition.Nodes {
		if orch.Definition.Nodes[i].ID == nodeID {
			return &orch.Definition.Nodes[i]
		}
	}
	return nil
}

// getOutgoingEdges 获取从节点出发的所有连线
func (e *Engine) getOutgoingEdges(orch *Orchestration, nodeID string) []OrchestrationEdge {
	var edges []OrchestrationEdge
	for _, edge := range orch.Definition.Edges {
		if edge.Source == nodeID {
			edges = append(edges, edge)
		}
	}
	return edges
}

// recordMessage 记录消息
func (e *Engine) recordMessage(execCtx *ExecutionContext, fromNodeID, toNodeID, msgType, content string) {
	msg := ExecutionMessage{
		ID:          generateMessageID(),
		ExecutionID: execCtx.ID,
		FromNodeID:  fromNodeID,
		ToNodeID:    toNodeID,
		MessageType: msgType,
		Content:     content,
		CreatedAt:   time.Now(),
	}
	execCtx.Messages = append(execCtx.Messages, msg)

	// 保存到数据库
	e.saveMessage(&msg)
}

// SubmitHumanInput 提交人工输入
func (e *Engine) SubmitHumanInput(executionID string, input string) error {
	e.mu.RLock()
	execCtx, ok := e.executions[executionID]
	e.mu.RUnlock()

	if !ok {
		return fmt.Errorf("execution not found: %s", executionID)
	}

	if execCtx.Status != "paused" {
		return fmt.Errorf("execution is not waiting for human input")
	}

	execCtx.HumanInputChan <- input
	return nil
}

// GetExecution 获取执行上下文
func (e *Engine) GetExecution(executionID string) (*ExecutionContext, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if execCtx, ok := e.executions[executionID]; ok {
		return execCtx, nil
	}

	// 从数据库加载
	return e.loadExecutionFromDB(executionID)
}

// loadOrchestration 加载编排定义
func (e *Engine) loadOrchestration(id string) (*Orchestration, error) {
	var orch Orchestration
	var defStr string

	err := e.DB.QueryRow(`
		SELECT id, name, description, status, definition, created_at, updated_at
		FROM orchestrations WHERE id = ?
	`, id).Scan(&orch.ID, &orch.Name, &orch.Description, &orch.Status, &defStr, &orch.CreatedAt, &orch.UpdatedAt)

	if err != nil {
		return nil, err
	}

	if defStr != "" {
		json.Unmarshal([]byte(defStr), &orch.Definition)
	}

	return &orch, nil
}

// saveExecution 保存执行记录
func (e *Engine) saveExecution(execCtx *ExecutionContext) error {
	variables, _ := json.Marshal(execCtx.Variables)

	_, err := e.DB.Exec(`
		INSERT OR REPLACE INTO orchestration_executions
		(id, orchestration_id, status, input, output, variables, current_node_id, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, execCtx.ID, execCtx.Orchestration.ID, execCtx.Status,
		execCtx.Variables["input"], execCtx.Variables["output"],
		string(variables), execCtx.CurrentNodeID, execCtx.StartTime, execCtx.UpdateTime)

	return err
}

// saveMessage 保存消息
func (e *Engine) saveMessage(msg *ExecutionMessage) error {
	_, err := e.DB.Exec(`
		INSERT INTO execution_messages (id, execution_id, from_node_id, to_node_id, message_type, content, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, msg.ID, msg.ExecutionID, msg.FromNodeID, msg.ToNodeID, msg.MessageType, msg.Content, msg.CreatedAt)

	return err
}

// loadExecutionFromDB 从数据库加载执行记录
func (e *Engine) loadExecutionFromDB(executionID string) (*ExecutionContext, error) {
	var execCtx ExecutionContext
	var variablesStr string
	var inputStr, outputStr string

	err := e.DB.QueryRow(`
		SELECT id, orchestration_id, status, input, output, variables, current_node_id, started_at, completed_at
		FROM orchestration_executions WHERE id = ?
	`, executionID).Scan(&execCtx.ID, &execCtx.Orchestration.ID, &execCtx.Status,
		&inputStr, &outputStr,
		&variablesStr, &execCtx.CurrentNodeID, &execCtx.StartTime, &execCtx.UpdateTime)

	if err != nil {
		return nil, err
	}

	// 初始化 Variables map
	execCtx.Variables = make(map[string]interface{})
	execCtx.Variables["input"] = inputStr
	execCtx.Variables["output"] = outputStr

	if variablesStr != "" {
		json.Unmarshal([]byte(variablesStr), &execCtx.Variables)
	}

	return &execCtx, nil
}

// generateExecutionID 生成执行 ID
func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}

// generateMessageID 生成消息 ID
func generateMessageID() string {
	return fmt.Sprintf("msg_%d", time.Now().UnixNano())
}
