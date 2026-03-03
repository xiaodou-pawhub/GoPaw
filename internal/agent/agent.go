// Package agent implements the ReAct (Reasoning + Acting) agent engine.
// The agent orchestrates LLM calls, tool execution, and memory management to
// produce responses to user messages.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/gopaw/gopaw/internal/convlog"
	"github.com/gopaw/gopaw/internal/llm"
	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/internal/skill"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
	"go.uber.org/zap"
)

// parsedReAct holds the structured output of one ReAct LLM response.
type parsedReAct struct {
	IsFinal     bool
	Answer      string // populated when IsFinal == true
	Action      string // tool name
	ActionInput string // raw JSON arguments string
}

var (
	// actionRe matches "Action: tool_name"
	actionRe = regexp.MustCompile(`(?m)^Action:\s*(.+)$`)
	// actionInputRe matches "Action Input: {...}"
	actionInputRe = regexp.MustCompile(`(?m)^Action Input:\s*(\{[\s\S]*?\})`)
	// finalAnswerRe matches "Final Answer: ..."
	finalAnswerRe = regexp.MustCompile(`(?ms)^Final Answer:\s*(.+)$`)
)

// parseReActOutput extracts structured fields from the LLM's raw text output.
// It supports both ReAct text format and JSON format.
func parseReActOutput(text string) parsedReAct {
	text = strings.TrimSpace(text)
	
	// Try JSON format first (some models output {"action": "...", "input": {...}})
	if strings.HasPrefix(text, "{") {
		if result, ok := parseJSONFormat(text); ok {
			return result
		}
	}
	
	// Check for Final Answer first (ReAct format).
	if m := finalAnswerRe.FindStringSubmatch(text); m != nil {
		return parsedReAct{IsFinal: true, Answer: strings.TrimSpace(m[1])}
	}

	var result parsedReAct
	if m := actionRe.FindStringSubmatch(text); m != nil {
		result.Action = strings.TrimSpace(m[1])
	}
	if m := actionInputRe.FindStringSubmatch(text); m != nil {
		result.ActionInput = strings.TrimSpace(m[1])
	}
	return result
}

// parseJSONFormat tries to parse the LLM output as JSON.
// Expected format: {"action": "tool_name", "input": {...}} or {"final_answer": "..."}
func parseJSONFormat(text string) (parsedReAct, bool) {
	text = strings.TrimSpace(text)
	
	// Try to unmarshal as a map
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return parsedReAct{}, false
	}
	
	// Check for final_answer field
	if answer, ok := data["final_answer"].(string); ok {
		return parsedReAct{IsFinal: true, Answer: answer}, true
	}
	
	// Check for action and input fields
	action, hasAction := data["action"].(string)
	if !hasAction {
		return parsedReAct{}, false
	}
	
	// Input can be a string (JSON) or an object
	var input string
	switch v := data["input"].(type) {
	case string:
		input = v
	case map[string]interface{}:
		// Marshal back to JSON string
		b, _ := json.Marshal(v)
		input = string(b)
	default:
		input = fmt.Sprintf("%v", v)
	}
	
	return parsedReAct{Action: action, ActionInput: input}, true
}

// ReActAgent is the core agent implementation using the ReAct reasoning pattern.
type ReActAgent struct {
	llmClient      llm.Client
	toolRegistry   *tool.Registry
	toolExecutor   *tool.Executor
	skillManager   *skill.Manager
	memoryManager  *memory.Manager
	sessionManager *SessionManager
	// defaultPrompt is used when agentMDPath is not set or the file cannot be read.
	defaultPrompt string
	// agentMDPath is the path to data/AGENT.md. When set, the system prompt is
	// read from this file on each request, enabling hot-reload via Web UI edits.
	agentMDPath string
	maxSteps    int
	logger      *zap.Logger
	convlog     *convlog.Logger // conversation event logger (may be nil)
}

// Config holds the parameters needed to construct a ReActAgent.
type Config struct {
	// DefaultPrompt is the fallback system prompt when AGENT.md is not found.
	DefaultPrompt string
	// AgentMDPath is the filesystem path to the AGENT.md persona file.
	// Leave empty to use DefaultPrompt only.
	AgentMDPath string
	MaxSteps    int
	// ConvLog is an optional conversation event logger.
	ConvLog *convlog.Logger
}

// New creates a ReActAgent.
func New(
	llmClient llm.Client,
	toolRegistry *tool.Registry,
	skillManager *skill.Manager,
	memoryManager *memory.Manager,
	cfg Config,
	logger *zap.Logger,
) *ReActAgent {
	return &ReActAgent{
		llmClient:      llmClient,
		toolRegistry:   toolRegistry,
		toolExecutor:   tool.NewExecutor(toolRegistry, logger),
		skillManager:   skillManager,
		memoryManager:  memoryManager,
		sessionManager: NewSessionManager(),
		defaultPrompt:  cfg.DefaultPrompt,
		agentMDPath:    cfg.AgentMDPath,
		maxSteps:       cfg.MaxSteps,
		logger:         logger,
		convlog:        cfg.ConvLog,
	}
}

// currentBasePrompt returns the active system prompt.
// It reads from AGENT.md on each call so Web UI edits take effect immediately.
func (a *ReActAgent) currentBasePrompt() string {
	if a.agentMDPath != "" {
		data, err := os.ReadFile(a.agentMDPath)
		if err == nil {
			return string(data)
		}
	}
	return a.defaultPrompt
}

// Process handles one user request through the full ReAct loop.
func (a *ReActAgent) Process(ctx context.Context, req *types.Request) (*types.Response, error) {
	if req.SessionID == "" {
		req.SessionID = "default"
	}

	// Retrieve (or create) the session.
	_, err := a.sessionManager.GetOrCreate(req.SessionID, req.UserID, req.Channel)
	if err != nil {
		return nil, fmt.Errorf("agent: session: %w", err)
	}

	// Log user message.
	if a.convlog != nil {
		if err := a.convlog.LogUserMessage(req.SessionID, req.Content); err != nil {
			a.logger.Warn("agent: failed to log user message", zap.Error(err))
		}
	}

	// Load conversation history from memory.
	history, err := a.memoryManager.GetContext(req.SessionID, 0)
	if err != nil {
		a.logger.Warn("agent: failed to load memory", zap.Error(err))
		history = nil
	}

	// 尝试压缩内存（如果上下文过长）
	// MaybeCompress 内部会检查 token 数量，超过限制才压缩
	if err := a.memoryManager.MaybeCompress(ctx, req.SessionID); err != nil {
		a.logger.Warn("agent: memory compression failed", zap.Error(err))
	}
	// 重新加载上下文
	history, _ = a.memoryManager.GetContext(req.SessionID, 0)

	// Build system prompt (reads from AGENT.md on each call for hot-reload).
	tools := a.toolRegistry.All()
	systemPrompt := buildSystemPrompt(a.currentBasePrompt(), a.skillManager.SystemPromptFragment(), tools)

	// Convert tools to LLM tool definitions (optional — for providers that support function calling).
	_ = buildToolDefinitions(tools) // used by stream handler; plain ReAct uses text parsing

	// Assemble the initial message list.
	messages := buildMessages(systemPrompt, history, req.Content)

	maxSteps := a.maxSteps
	if maxSteps <= 0 {
		maxSteps = 20
	}

	var finalAnswer string
	for step := 0; step < maxSteps; step++ {
		a.logger.Debug("agent: react step", zap.Int("step", step), zap.String("session", req.SessionID))

		resp, err := a.llmClient.Chat(ctx, llm.ChatRequest{Messages: messages})
		if err != nil {
			return nil, fmt.Errorf("agent: llm call step %d: %w", step, err)
		}

		reply := resp.Message.Content
		parsed := parseReActOutput(reply)

		if parsed.IsFinal {
			finalAnswer = parsed.Answer
			break
		}

		if parsed.Action == "" {
			// LLM returned something unexpected; treat entire reply as the answer.
			finalAnswer = reply
			break
		}

		// Log tool call.
		if a.convlog != nil {
			if err := a.convlog.LogToolCall(req.SessionID, parsed.Action, json.RawMessage(parsed.ActionInput)); err != nil {
				a.logger.Warn("agent: failed to log tool call", zap.Error(err))
			}
		}

		// Execute the requested tool.
		a.logger.Info("agent: executing tool",
			zap.String("tool", parsed.Action),
			zap.String("args", parsed.ActionInput),
		)
		observation, execErr := a.toolExecutor.Execute(ctx, parsed.Action, parsed.ActionInput)
		if execErr != nil {
			observation = fmt.Sprintf("Tool error: %v", execErr)
		}

		// Log tool result.
		if a.convlog != nil {
			var errPtr *string
			if execErr != nil {
				errStr := execErr.Error()
				errPtr = &errStr
			}
			if err := a.convlog.LogToolResult(req.SessionID, parsed.Action, observation, errPtr); err != nil {
				a.logger.Warn("agent: failed to log tool result", zap.Error(err))
			}
		}

		// Append the ReAct step to the message list.
		messages = appendReActStep(messages, reply, observation)
	}

	if finalAnswer == "" {
		return nil, fmt.Errorf("agent: max steps (%d) reached without final answer", maxSteps)
	}

	// Log agent reply.
	if a.convlog != nil {
		if err := a.convlog.LogAgentReply(req.SessionID, finalAnswer, nil); err != nil {
			a.logger.Warn("agent: failed to log agent reply", zap.Error(err))
		}
	}

	// Persist the exchange to memory.
	if memErr := a.memoryManager.Add(req.SessionID, req.UserID, req.Channel, req.Content, finalAnswer); memErr != nil {
		a.logger.Warn("agent: failed to save memory", zap.Error(memErr))
	}

	return &types.Response{
		Content: finalAnswer,
		MsgType: types.MsgTypeText,
	}, nil
}

// buildToolDefinitions converts plugin.Tool slice to LLM ToolDefinition slice.
func buildToolDefinitions(tools []plugin.Tool) []llm.ToolDefinition {
	defs := make([]llm.ToolDefinition, 0, len(tools))
	for _, t := range tools {
		defs = append(defs, llm.ToolDefinition{
			Type: "function",
			Function: llm.FunctionDef{
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
			},
		})
	}
	return defs
}

// Sessions returns the session manager for use by the HTTP handlers.
func (a *ReActAgent) Sessions() *SessionManager {
	return a.sessionManager
}
