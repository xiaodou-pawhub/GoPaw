package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ToolDefinition 工具定义
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolHandler 工具处理函数
type ToolHandler func(args map[string]interface{}) (string, error)

// RegisterTools 注册知识库工具到 Agent Manager
func (s *Service) RegisterTools(registerFunc func(name, description string, handler interface{})) {
	// 注册知识库搜索工具
	registerFunc(
		"knowledge_search",
		"从知识库中搜索相关信息，用于回答用户关于特定领域的问题。",
		s.handleKnowledgeSearch,
	)

	// 注册知识库列表工具
	registerFunc(
		"knowledge_list",
		"列出所有可用的知识库，获取知识库ID用于搜索。",
		s.handleKnowledgeList,
	)
}

// GetToolDefinitions 获取工具定义列表
func (s *Service) GetToolDefinitions() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "knowledge_search",
			Description: "从知识库中搜索相关信息，用于回答用户关于特定领域的问题。",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"knowledge_base_id": map[string]string{
						"type":        "string",
						"description": "知识库ID，可通过 knowledge_list 获取",
					},
					"query": map[string]string{
						"type":        "string",
						"description": "搜索查询，描述你要查找的信息",
					},
					"top_k": map[string]string{
						"type":        "number",
						"description": "返回结果数量（默认5）",
					},
				},
				"required": []string{"knowledge_base_id", "query"},
			},
		},
		{
			Name:        "knowledge_list",
			Description: "列出所有可用的知识库，获取知识库ID用于搜索。",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}
}

// handleKnowledgeSearch 处理知识库搜索
func (s *Service) handleKnowledgeSearch(args map[string]interface{}) (string, error) {
	kbID, ok := args["knowledge_base_id"].(string)
	if !ok || kbID == "" {
		return "", fmt.Errorf("knowledge_base_id is required")
	}

	query, ok := args["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("query is required")
	}

	topK := 5
	if k, ok := args["top_k"].(float64); ok {
		topK = int(k)
	}

	// 执行搜索
	resp, err := s.Search(context.Background(), kbID, SearchRequest{
		Query:      query,
		TopK:       topK,
		SearchType: "hybrid",
	})
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(resp.Results) == 0 {
		return "未找到相关信息。", nil
	}

	// 格式化结果
	var result strings.Builder
	result.WriteString(fmt.Sprintf("找到 %d 条相关信息：\n\n", len(resp.Results)))

	for i, r := range resp.Results {
		result.WriteString(fmt.Sprintf("[%d] 来自《%s》：\n", i+1, r.DocumentName))
		result.WriteString(r.Content)
		result.WriteString("\n\n")
	}

	return result.String(), nil
}

// handleKnowledgeList 处理知识库列表
func (s *Service) handleKnowledgeList(args map[string]interface{}) (string, error) {
	bases, err := s.ListKnowledgeBases(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to list knowledge bases: %w", err)
	}

	if len(bases) == 0 {
		return "当前没有可用的知识库。", nil
	}

	var result strings.Builder
	result.WriteString("可用知识库列表：\n\n")

	for _, kb := range bases {
		result.WriteString(fmt.Sprintf("- ID: %s\n", kb.ID))
		result.WriteString(fmt.Sprintf("  名称: %s\n", kb.Name))
		result.WriteString(fmt.Sprintf("  描述: %s\n", kb.Description))
		result.WriteString(fmt.Sprintf("  文档数: %d, 块数: %d\n", kb.DocumentCount, kb.ChunkCount))
		result.WriteString("\n")
	}

	return result.String(), nil
}

// FormatSearchResultsForAgent 格式化搜索结果供 Agent 使用
func FormatSearchResultsForAgent(results []SearchResult) string {
	if len(results) == 0 {
		return "未找到相关信息。"
	}

	var context strings.Builder
	context.WriteString("基于以下参考资料回答问题：\n\n")

	for i, r := range results {
		context.WriteString(fmt.Sprintf("--- 参考 %d ---\n", i+1))
		context.WriteString(fmt.Sprintf("来源：%s\n", r.DocumentName))
		context.WriteString(fmt.Sprintf("内容：%s\n", r.Content))
		context.WriteString("\n")
	}

	return context.String()
}

// GetRelevantContext 获取相关上下文（用于直接注入到 Prompt）
func (s *Service) GetRelevantContext(ctx context.Context, kbID string, query string, topK int) (string, error) {
	resp, err := s.Search(ctx, kbID, SearchRequest{
		Query:      query,
		TopK:       topK,
		SearchType: "hybrid",
	})
	if err != nil {
		return "", err
	}

	return FormatSearchResultsForAgent(resp.Results), nil
}

// ToolCall 工具调用结构
type ToolCall struct {
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ExecuteTool 执行工具调用
func (s *Service) ExecuteTool(call ToolCall) (string, error) {
	switch call.Name {
	case "knowledge_search":
		return s.handleKnowledgeSearch(call.Parameters)
	case "knowledge_list":
		return s.handleKnowledgeList(call.Parameters)
	default:
		return "", fmt.Errorf("unknown tool: %s", call.Name)
	}
}

// MarshalToolDefinitions 序列化工具定义为 JSON
func (s *Service) MarshalToolDefinitions() (string, error) {
	defs := s.GetToolDefinitions()
	data, err := json.MarshalIndent(defs, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
