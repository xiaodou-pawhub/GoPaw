// Package agent implements the ReAct (Reasoning + Acting) agent engine.
// The agent orchestrates LLM calls, tool execution, and memory management to
// produce responses to user messages.
package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"

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
func parseReActOutput(text string) parsedReAct {
	// Check for Final Answer first.
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

// ReActAgent is the core agent implementation using the ReAct reasoning pattern.
type ReActAgent struct {
	llmClient      llm.Client
	toolRegistry   *tool.Registry
	toolExecutor   *tool.Executor
	skillManager   *skill.Manager
	memoryManager  *memory.Manager
	sessionManager *SessionManager
	basePrompt     string
	maxSteps       int
	logger         *zap.Logger
}

// Config holds the parameters needed to construct a ReActAgent.
type Config struct {
	BasePrompt string
	MaxSteps   int
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
		basePrompt:     cfg.BasePrompt,
		maxSteps:       cfg.MaxSteps,
		logger:         logger,
	}
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

	// Load conversation history from memory.
	history, err := a.memoryManager.GetContext(req.SessionID, 0)
	if err != nil {
		a.logger.Warn("agent: failed to load memory", zap.Error(err))
		history = nil
	}

	// Optionally compress memory if context is getting long.
	if memory.EstimateTokens(history) > 3000 {
		if cErr := a.memoryManager.Compress(ctx, req.SessionID); cErr != nil {
			a.logger.Warn("agent: memory compression failed", zap.Error(cErr))
		} else {
			// Reload after compression.
			history, _ = a.memoryManager.GetContext(req.SessionID, 0)
		}
	}

	// Build system prompt.
	tools := a.toolRegistry.All()
	systemPrompt := buildSystemPrompt(a.basePrompt, a.skillManager.SystemPromptFragment(), tools)

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

		// Execute the requested tool.
		a.logger.Info("agent: executing tool",
			zap.String("tool", parsed.Action),
			zap.String("args", parsed.ActionInput),
		)
		observation, execErr := a.toolExecutor.Execute(ctx, parsed.Action, parsed.ActionInput)
		if execErr != nil {
			observation = fmt.Sprintf("Tool error: %v", execErr)
		}

		// Append the ReAct step to the message list.
		messages = appendReActStep(messages, reply, observation)
	}

	if finalAnswer == "" {
		return nil, fmt.Errorf("agent: max steps (%d) reached without final answer", maxSteps)
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
