// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package nodes

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gopaw/gopaw/internal/flow"
)

// TransformNodeExecutor 数据转换节点执行器
// 用于数据格式转换、字段映射、模板渲染等
type TransformNodeExecutor struct {
	flow.BaseNodeExecutor
}

// NewTransformNodeExecutor 创建数据转换节点执行器
func NewTransformNodeExecutor() *TransformNodeExecutor {
	return &TransformNodeExecutor{
		BaseNodeExecutor: flow.NewBaseNodeExecutor(flow.NodeTypeTransform),
	}
}

// Execute 执行数据转换节点
func (e *TransformNodeExecutor) Execute(ctx context.Context, node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	// 获取转换类型
	transformType, _ := node.Config["type"].(string)
	if transformType == "" {
		transformType = "template"
	}

	var result map[string]interface{}
	var err error

	switch transformType {
	case "template":
		result, err = e.executeTemplate(node, exec)
	case "mapping":
		result, err = e.executeMapping(node, exec)
	case "jsonpath":
		result, err = e.executeJsonPath(node, exec)
	case "script":
		result, err = e.executeScript(node, exec)
	default:
		return nil, fmt.Errorf("unknown transform type: %s", transformType)
	}

	if err != nil {
		return nil, err
	}

	// 存储结果到上下文
	for k, v := range result {
		exec.Context[node.ID+"_"+k] = v
	}

	return result, nil
}

// executeTemplate 执行模板转换
func (e *TransformNodeExecutor) executeTemplate(node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	template, _ := node.Config["template"].(string)
	if template == "" {
		return nil, fmt.Errorf("template is required")
	}

	// 简单模板变量替换
	result := e.replaceVariables(template, exec.Variables, exec.Context)

	return map[string]interface{}{
		"result": result,
	}, nil
}

// executeMapping 执行字段映射
func (e *TransformNodeExecutor) executeMapping(node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	mapping, ok := node.Config["mapping"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("mapping is required")
	}

	result := make(map[string]interface{})
	for targetField, sourceExpr := range mapping {
		expr, _ := sourceExpr.(string)
		value := e.resolveValue(expr, exec.Variables, exec.Context)
		result[targetField] = value
	}

	return result, nil
}

// executeJsonPath 执行 JSONPath 提取
func (e *TransformNodeExecutor) executeJsonPath(node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	// 获取输入数据
	inputKey, _ := node.Config["input"].(string)
	if inputKey == "" {
		inputKey = "input"
	}

	inputData, ok := exec.Context[inputKey]
	if !ok {
		return nil, fmt.Errorf("input data not found: %s", inputKey)
	}

	// 获取 JSONPath 表达式
	jsonPath, _ := node.Config["path"].(string)
	if jsonPath == "" {
		return nil, fmt.Errorf("path is required")
	}

	// 简单 JSONPath 实现（仅支持 $.field 格式）
	result, err := e.extractJsonPath(inputData, jsonPath)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"result": result,
	}, nil
}

// executeScript 执行脚本转换
func (e *TransformNodeExecutor) executeScript(node *flow.FlowNode, exec *flow.Execution) (map[string]interface{}, error) {
	// 脚本执行需要安全沙箱，这里先返回错误
	return nil, fmt.Errorf("script transform not implemented yet")
}

// replaceVariables 替换模板变量
func (e *TransformNodeExecutor) replaceVariables(template string, variables, context map[string]interface{}) string {
	result := template

	// 替换 ${var} 格式的变量
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	result = re.ReplaceAllStringFunc(result, func(match string) string {
		varName := strings.TrimPrefix(strings.TrimSuffix(match, "}"), "${")
		if val, ok := variables[varName]; ok {
			return fmt.Sprintf("%v", val)
		}
		if val, ok := context[varName]; ok {
			return fmt.Sprintf("%v", val)
		}
		return match
	})

	return result
}

// resolveValue 解析值
func (e *TransformNodeExecutor) resolveValue(expr string, variables, context map[string]interface{}) interface{} {
	// 检查是否是变量引用
	if strings.HasPrefix(expr, "${") && strings.HasSuffix(expr, "}") {
		varName := strings.TrimPrefix(strings.TrimSuffix(expr, "}"), "${")
		if val, ok := variables[varName]; ok {
			return val
		}
		if val, ok := context[varName]; ok {
			return val
		}
	}

	// 尝试解析为数字
	if num, err := strconv.ParseFloat(expr, 64); err == nil {
		return num
	}

	// 尝试解析为布尔值
	if expr == "true" {
		return true
	}
	if expr == "false" {
		return false
	}

	// 返回字符串
	return expr
}

// extractJsonPath 从数据中提取 JSONPath
func (e *TransformNodeExecutor) extractJsonPath(data interface{}, path string) (interface{}, error) {
	// 简单实现，仅支持 $.field 格式
	path = strings.TrimPrefix(path, "$.")
	if path == "" {
		return data, nil
	}

	// 将数据转换为 map
	var dataMap map[string]interface{}
	switch v := data.(type) {
	case map[string]interface{}:
		dataMap = v
	case string:
		if err := json.Unmarshal([]byte(v), &dataMap); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported data type")
	}

	// 按点分割路径
	parts := strings.Split(path, ".")
	current := dataMap

	for _, part := range parts {
		if val, ok := current[part]; ok {
			if m, ok := val.(map[string]interface{}); ok {
				current = m
			} else {
				return val, nil
			}
		} else {
			return nil, fmt.Errorf("path not found: %s", part)
		}
	}

	return current, nil
}

// Validate 验证节点配置
func (e *TransformNodeExecutor) Validate(node *flow.FlowNode) error {
	if err := e.BaseNodeExecutor.Validate(node); err != nil {
		return err
	}

	transformType, _ := node.Config["type"].(string)
	if transformType == "" {
		transformType = "template"
	}

	switch transformType {
	case "template":
		if _, ok := node.Config["template"]; !ok {
			return fmt.Errorf("template is required for template transform")
		}
	case "mapping":
		if _, ok := node.Config["mapping"]; !ok {
			return fmt.Errorf("mapping is required for mapping transform")
		}
	case "jsonpath":
		if _, ok := node.Config["path"]; !ok {
			return fmt.Errorf("path is required for jsonpath transform")
		}
	}

	return nil
}

func init() {
	flow.MustRegisterNode(NewTransformNodeExecutor())
}