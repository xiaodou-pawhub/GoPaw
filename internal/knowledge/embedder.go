package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Embedder 文本向量化接口
type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	BatchEmbed(ctx context.Context, texts []string) ([][]float32, error)
	Dimension() int
}

// EmbedderConfig Embedding 配置
type EmbedderConfig struct {
	Provider string `json:"provider" yaml:"provider"` // ollama / openai
	Model    string `json:"model" yaml:"model"`
	BaseURL  string `json:"base_url" yaml:"base_url"`
	APIKey   string `json:"api_key" yaml:"api_key"`
}

// NewEmbedder 创建 Embedder
func NewEmbedder(config EmbedderConfig) (Embedder, error) {
	switch config.Provider {
	case "ollama":
		return NewOllamaEmbedder(config.BaseURL, config.Model), nil
	case "openai":
		return NewOpenAIEmbedder(config.BaseURL, config.APIKey, config.Model), nil
	default:
		return nil, fmt.Errorf("unsupported embedder provider: %s", config.Provider)
	}
}

// OllamaEmbedder Ollama Embedding 服务
type OllamaEmbedder struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewOllamaEmbedder 创建 Ollama Embedder
func NewOllamaEmbedder(baseURL, model string) *OllamaEmbedder {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "nomic-embed-text"
	}

	return &OllamaEmbedder{
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Embed 生成单个文本的向量
func (e *OllamaEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.BatchEmbed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding generated")
	}
	return embeddings[0], nil
}

// BatchEmbed 批量生成向量
func (e *OllamaEmbedder) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	var results [][]float32

	for _, text := range texts {
		reqBody := map[string]interface{}{
			"model": e.model,
			"prompt": text,
		}

		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}

		req, err := http.NewRequestWithContext(ctx, "POST",
			e.baseURL+"/api/embeddings",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := e.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("ollama embedding request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("ollama embedding failed with status: %d", resp.StatusCode)
		}

		var result struct {
			Embedding []float32 `json:"embedding"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode ollama response: %w", err)
		}

		results = append(results, result.Embedding)
	}

	return results, nil
}

// Dimension 返回向量维度
func (e *OllamaEmbedder) Dimension() int {
	switch e.model {
	case "nomic-embed-text":
		return 768
	case "mxbai-embed-large":
		return 1024
	default:
		return 768
	}
}

// OpenAIEmbedder OpenAI Embedding 服务
type OpenAIEmbedder struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// NewOpenAIEmbedder 创建 OpenAI Embedder
func NewOpenAIEmbedder(baseURL, apiKey, model string) *OpenAIEmbedder {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "text-embedding-3-small"
	}

	return &OpenAIEmbedder{
		baseURL: baseURL,
		apiKey:  apiKey,
		model:   model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Embed 生成单个文本的向量
func (e *OpenAIEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	embeddings, err := e.BatchEmbed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding generated")
	}
	return embeddings[0], nil
}

// BatchEmbed 批量生成向量
func (e *OpenAIEmbedder) BatchEmbed(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := map[string]interface{}{
		"model": e.model,
		"input": texts,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST",
		e.baseURL+"/embeddings",
		bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai embedding request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai embedding failed with status: %d", resp.StatusCode)
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode openai response: %w", err)
	}

	// 按索引排序
	embeddings := make([][]float32, len(texts))
	for _, item := range result.Data {
		embeddings[item.Index] = item.Embedding
	}

	return embeddings, nil
}

// Dimension 返回向量维度
func (e *OpenAIEmbedder) Dimension() int {
	switch e.model {
	case "text-embedding-3-small":
		return 1536
	case "text-embedding-3-large":
		return 3072
	case "text-embedding-ada-002":
		return 1536
	default:
		return 1536
	}
}
