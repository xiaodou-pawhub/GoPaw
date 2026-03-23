// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package nodes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gopaw/gopaw/internal/flow"
)

// HTTPNodeExecutor HTTP 请求节点执行器
type HTTPNodeExecutor struct {
	flow.BaseNodeExecutor
	client *http.Client
}

// NewHTTPNodeExecutor 创建 HTTP 节点执行器
func NewHTTPNodeExecutor() *HTTPNodeExecutor {
	return &HTTPNodeExecutor{
		BaseNodeExecutor: flow.NewBaseNodeExecutor(flow.NodeTypeHTTP),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Execute 执行 HTTP 请求节点
func (e *HTTPNodeExecutor) Execute(ctx context.Context, node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	// 获取请求配置
	url, _ := node.Config["url"].(string)
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}

	method, _ := node.Config["method"].(string)
	if method == "" {
		method = "GET"
	}

	// 构建请求
	var reqBody io.Reader
	if body, ok := node.Config["body"]; ok {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	if headers, ok := node.Config["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	// 设置 Content-Type
	if contentType, ok := node.Config["content_type"].(string); ok {
		req.Header.Set("Content-Type", contentType)
	} else if method != "GET" {
		req.Header.Set("Content-Type", "application/json")
	}

	// 发送请求
	startTime := time.Now()
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	duration := time.Since(startTime)

	// 解析响应
	var respData interface{}
	if err := json.Unmarshal(respBody, &respData); err != nil {
		respData = string(respBody)
	}

	// 检查状态码
	statusCode := resp.StatusCode
	if statusCode >= 400 {
		return nil, fmt.Errorf("HTTP error: %d - %s", statusCode, string(respBody))
	}

	return map[string]interface{}{
		"status_code": statusCode,
		"body":        respData,
		"headers":     resp.Header,
		"duration":    duration.Milliseconds(),
	}, nil
}

// Validate 验证节点配置
func (e *HTTPNodeExecutor) Validate(node *flow.FlowNode) error {
	if err := e.BaseNodeExecutor.Validate(node); err != nil {
		return err
	}

	url, _ := node.Config["url"].(string)
	if url == "" {
		return fmt.Errorf("url is required")
	}

	method, _ := node.Config["method"].(string)
	if method != "" && method != "GET" && method != "POST" && method != "PUT" && method != "DELETE" && method != "PATCH" {
		return fmt.Errorf("invalid HTTP method: %s", method)
	}

	return nil
}

func init() {
	flow.MustRegisterNode(NewHTTPNodeExecutor())
}