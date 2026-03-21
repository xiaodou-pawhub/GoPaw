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
	// Coding Plan 系列
	{
		ID:      "baidu-qianfan-coding",
		Name:    "百度千帆 Coding Plan",
		BaseURL: "https://qianfan.baidubce.com/v2/coding/chat/completions",
		Models:  []string{"Kimi-K2.5", "Kimi-K2.5（默认）", "DeepSeek-V3.2", "GLM-5", "MiniMax-M2.5"},
	},
	{
		ID:      "ali-bailian-coding",
		Name:    "阿里百炼 Coding Plan",
		BaseURL: "https://coding.dashscope.aliyuncs.com/v1",
		Models:  []string{"qwen3.5-plus", "qwen3-max-2026-01-23", "qwen3-coder-next", "qwen3-coder-plus", "glm-5", "kimi-k2.5", "minimax-m2.5"},
	},
	{
		ID:      "volc-ark-coding",
		Name:    "方舟 Coding Plan",
		BaseURL: "https://ark.cn-beijing.volces.com/api/coding/v3",
		Models:  []string{"doubao-seed-2.0-code", "doubao-seed-2.0-pro", "doubao-seed-2.0-lite", "doubao-seed-code", "minimax-m2.5", "glm-4.7", "deepseek-v3.2", "kimi-k2.5"},
	},
	{
		ID:      "zhipu-coding",
		Name:    "智谱 Coding Plan",
		BaseURL: "https://open.bigmodel.cn/api/coding/paas/v4",
		Models:  []string{"GLM-5", "GLM-4.7", "GLM-4.6"},
	},
	{
		ID:      "tencent-coding",
		Name:    "腾讯 Coding Plan",
		BaseURL: "https://api.lkeap.cloud.tencent.com/coding/v3",
		Models:  []string{"tc-code-latest", "hunyuan-2.0-instruct", "hunyuan-2.0-thinking", "minimax-m2.5", "kimi-k2.5", "glm-5", "hunyuan-t1", "hunyuan-turbos"},
	},
	{
		ID:      "modelscope",
		Name:    "魔搭 ModelScope",
		BaseURL: "https://api-inference.modelscope.cn/v1/",
		Models:  []string{"ZhipuAI/GLM-5", "deepseek-ai/DeepSeek-Coder-V2-Instruct", "deepseek-ai/DeepSeek-V3", "Qwen/Qwen3-Coder-Next", "Qwen/Qwen3-235B-A22B-Instruct-2507"},
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
