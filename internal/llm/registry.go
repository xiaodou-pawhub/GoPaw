package llm

import (
	"strings"
)

// BuiltinProvider 定义预置厂商的基础信息
type BuiltinProvider struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	BaseURL string   `json:"base_url"`
	Models  []string `json:"models"`
}

// BuiltinProviders 预置厂商库
var BuiltinProviders = []BuiltinProvider{
	{
		ID:      "openai",
		Name:    "OpenAI",
		BaseURL: "https://api.openai.com/v1",
		Models:  []string{"gpt-4o", "gpt-4o-mini", "o1-preview", "gpt-4-turbo"},
	},
	{
		ID:      "anthropic",
		Name:    "Anthropic",
		BaseURL: "https://api.anthropic.com/v1",
		Models:  []string{"claude-3-5-sonnet-latest", "claude-3-opus-20240229", "claude-3-haiku-20240307"},
	},
	{
		ID:      "deepseek",
		Name:    "DeepSeek",
		BaseURL: "https://api.deepseek.com",
		Models:  []string{"deepseek-chat", "deepseek-reasoner"},
	},
	{
		ID:      "aliyun",
		Name:    "阿里云百炼 (Qwen)",
		BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		Models:  []string{"qwen-max", "qwen-plus", "qwen-turbo", "qwen-long"},
	},
	{
		ID:      "google",
		Name:    "Google Gemini",
		BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai",
		Models:  []string{"gemini-1.5-pro", "gemini-1.5-flash"},
	},
	{
		ID:      "ollama",
		Name:    "Ollama (Local)",
		BaseURL: "http://localhost:11434/v1",
		Models:  []string{},
	},
}

// InferTags 根据模型名称推断能力标签
func InferTags(modelName string) []string {
	var tags []string
	m := strings.ToLower(modelName)

	// 推断 Function Calling (fc)
	// OpenAI, Anthropic, Qwen, Gemini, DeepSeek-Chat 均支持原生工具调用
	if strings.Contains(m, "gpt-4") || 
	   strings.Contains(m, "gpt-3.5") || 
	   strings.Contains(m, "claude-3") || 
	   strings.Contains(m, "qwen-") || 
	   strings.Contains(m, "gemini") || 
	   strings.Contains(m, "deepseek-chat") ||
	   strings.Contains(m, "mistral") {
		tags = append(tags, "fc")
	}

	// 推断 Vision
	if strings.Contains(m, "vision") || 
	   strings.Contains(m, "gpt-4o") || 
	   strings.Contains(m, "claude-3-5-sonnet") || 
	   strings.Contains(m, "gemini-1.5") {
		tags = append(tags, "vision")
	}

	// 推断 Reasoning / Thinking
	if strings.Contains(m, "reasoner") || 
	   strings.Contains(m, "r1") || 
	   strings.Contains(m, "o1-") || 
	   strings.Contains(m, "o3-") {
		tags = append(tags, "reasoning")
	}

	return tags
}
