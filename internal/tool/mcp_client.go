package tool

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/gopaw/gopaw/pkg/plugin"
)

// MCPClient implements the Model Context Protocol over stdio.
type MCPClient struct {
	serverName string
	command    string
	args       []string
	
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	
	idGen  atomic.Uint64
	mu     sync.Mutex
	calls  map[uint64]chan *jsonRPCResponse
	
	tools  []MCPToolInfo
}

type MCPToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type jsonRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      uint64      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      uint64          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewMCPClient creates a client for a specific MCP server.
func NewMCPClient(name, command string, args []string) *MCPClient {
	return &MCPClient{
		serverName: name,
		command:    command,
		args:       args,
		calls:      make(map[uint64]chan *jsonRPCResponse),
	}
}

// Start launches the MCP server and performs the handshake.
func (c *MCPClient) Start(ctx context.Context) error {
	c.cmd = exec.CommandContext(ctx, c.command, c.args...)
	
	var err error
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return err
	}
	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := c.cmd.Start(); err != nil {
		return err
	}

	go c.listen()

	// 1. Initialize
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]string{"name": "GoPaw-Agent", "version": "0.1.0"},
	}
	if _, err := c.call(ctx, "initialize", initParams); err != nil {
		return fmt.Errorf("mcp initialize failed: %w", err)
	}

	// 2. Notifications: initialized
	if err := c.notify("notifications/initialized", nil); err != nil {
		return err
	}

	// 3. List Tools
	res, err := c.call(ctx, "tools/list", nil)
	if err != nil {
		return fmt.Errorf("mcp list tools failed: %w", err)
	}

	var toolList struct {
		Tools []MCPToolInfo `json:"tools"`
	}
	if err := json.Unmarshal(res, &toolList); err != nil {
		return err
	}
	c.tools = toolList.Tools

	return nil
}

func (c *MCPClient) listen() {
	scanner := bufio.NewScanner(c.stdout)
	for scanner.Scan() {
		var resp jsonRPCResponse
		if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
			continue
		}

		c.mu.Lock()
		ch, ok := c.calls[resp.ID]
		if ok {
			delete(c.calls, resp.ID)
			ch <- &resp
		}
		c.mu.Unlock()
	}
}

func (c *MCPClient) call(ctx context.Context, method string, params interface{}) (json.RawMessage, error) {
	id := c.idGen.Add(1)
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	ch := make(chan *jsonRPCResponse, 1)
	c.mu.Lock()
	c.calls[id] = ch
	c.mu.Unlock()

	data, _ := json.Marshal(req)
	fmt.Fprintln(c.stdin, string(data))

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-ch:
		if resp.Error != nil {
			return nil, fmt.Errorf("mcp error (%d): %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	}
}

func (c *MCPClient) notify(method string, params interface{}) error {
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	data, _ := json.Marshal(req)
	_, err := fmt.Fprintln(c.stdin, string(data))
	return err
}

// GetTools returns the list of tools discovered from the MCP server.
func (c *MCPClient) GetTools() []MCPToolInfo {
	return c.tools
}

// ExecuteTool calls a specific tool on the MCP server.
func (c *MCPClient) ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) *plugin.ToolResult {
	params := map[string]interface{}{
		"name":      toolName,
		"arguments": args,
	}

	res, err := c.call(ctx, "tools/call", params)
	if err != nil {
		return plugin.ErrorResult(err.Error())
	}

	var callRes struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(res, &callRes); err != nil {
		return plugin.ErrorResult("failed to parse mcp result")
	}

	output := ""
	for _, c := range callRes.Content {
		if c.Type == "text" {
			output += c.Text
		}
	}

	return &plugin.ToolResult{
		LLMOutput: output,
		IsError:   callRes.IsError,
	}
}
