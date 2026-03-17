// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

// Package agent implements the native Function Calling agent engine.
// The agent orchestrates LLM calls, tool execution, and memory management to
// produce responses to user messages.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gopaw/gopaw/internal/convlog"
	"github.com/gopaw/gopaw/internal/llm"
	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/internal/skill"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
	"go.uber.org/zap"
)

// timeNow is a variable to allow test overrides.
var timeNow = time.Now

// ReActAgent is the core agent implementation using native Function Calling.
type ReActAgent struct {
	llmClient      llm.Client
	toolRegistry   *tool.Registry
	toolExecutor   *tool.Executor
	skillManager   *skill.Manager
	memoryManager  *memory.Manager
	ltmStore       *memory.LTMStore // structured long-term memory (memories.db)
	sessionManager *SessionManager
	contextBuilder *ContextBuilder // dynamic context builder
	// defaultPrompt is used when agentMDPath is not set or the file cannot be read.
	defaultPrompt string
	// agentMDPath is the path to data/AGENT.md. When set, the system prompt is
	// read from this file on each request, enabling hot-reload via Web UI edits.
	agentMDPath    string
	// memoryNotesDir is the directory containing daily notes (memory/notes/).
	memoryNotesDir string
	maxSteps       int
	hooks          Hooks
	logger         *zap.Logger
	convlog        *convlog.Logger // conversation event logger (may be nil)
}

// Config holds the parameters needed to construct a ReActAgent.
type Config struct {
	// DefaultPrompt is the fallback system prompt when AGENT.md is not found.
	DefaultPrompt string
	// AgentMDPath is the filesystem path to the AGENT.md persona file.
	// Leave empty to use DefaultPrompt only.
	AgentMDPath string
	// LTMStore is the structured long-term memory store (memories.db).
	LTMStore *memory.LTMStore
	// MemoryNotesDir is the directory containing daily notes (memory/notes/).
	MemoryNotesDir string
	MaxSteps       int
	// Hooks contains optional callbacks invoked at key points in the agent loop.
	Hooks Hooks
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
	agent := &ReActAgent{
		llmClient:      llmClient,
		toolRegistry:   toolRegistry,
		toolExecutor:   tool.NewExecutor(toolRegistry, logger),
		skillManager:   skillManager,
		memoryManager:  memoryManager,
		ltmStore:       cfg.LTMStore,
		sessionManager: NewSessionManager(),
		defaultPrompt:  cfg.DefaultPrompt,
		agentMDPath:    cfg.AgentMDPath,
		memoryNotesDir: cfg.MemoryNotesDir,
		maxSteps:       cfg.MaxSteps,
		hooks:          cfg.Hooks,
		logger:         logger,
		convlog:        cfg.ConvLog,
	}

	// Initialize context builder with base persona
	persona := agent.currentBasePrompt()
	agent.contextBuilder = NewContextBuilder(
		persona,
		memoryManager,
		cfg.LTMStore,
		skillManager,
		cfg.MemoryNotesDir,
		2000, // Token budget for dynamic content
		logger,
	)

	return agent
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

// SetApprovalUI sets the approval UI handler for tool execution.
func (a *ReActAgent) SetApprovalUI(ui tool.ApprovalUI) {
	a.toolExecutor.SetApprovalUI(ui)
}

// SetL2NotificationCallback sets the callback for L2 tool execution notifications.
func (a *ReActAgent) SetL2NotificationCallback(cb tool.L2NotificationCallback) {
	a.toolExecutor.SetL2NotificationCallback(cb)
}

// ReloadPersona reloads the persona from AGENT.md and updates the context builder.
// This is called when the AGENT.md file changes (hot reload).
func (a *ReActAgent) ReloadPersona() {
	newPersona := a.currentBasePrompt()
	a.contextBuilder.SetPersona(newPersona)
	a.logger.Info("persona reloaded from file",
		zap.String("path", a.agentMDPath),
	)
}

// GetAgentMDPath returns the path to the AGENT.md file.
func (a *ReActAgent) GetAgentMDPath() string {
	return a.agentMDPath
}

// contextWarnTokens: log a warning when the prompt exceeds this many tokens.
const contextWarnTokens = 100_000

// contextHardLimitTokens: abort the agent loop when the prompt exceeds this many tokens.
// Most frontier models top out at 128K–200K context; 150K is a safe hard limit.
const contextHardLimitTokens = 150_000

// Tool step limit thresholds for progressive warnings.
const (
	toolStepWarnThreshold = 0.8  // Warn at 80% of max steps
	toolStepFinalWarning  = 0.96 // Final warning at 96% of max steps (maxSteps - 2)
)

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

// checkToolStepLimit checks if the current step has reached warning thresholds.
// Returns a warning message if applicable, or empty string if no warning needed.
func checkToolStepLimit(step, maxSteps int) string {
	if maxSteps <= 0 {
		maxSteps = 20
	}

	// 80% threshold warning
	warnThreshold := int(float64(maxSteps) * toolStepWarnThreshold)
	if step == warnThreshold {
		return fmt.Sprintf("⚠️ Tool call limit warning: %d/%d steps used (80%%). Consider wrapping up soon.", step, maxSteps)
	}

	// Final warning at maxSteps - 2
	finalThreshold := maxSteps - 2
	if step == finalThreshold && finalThreshold > warnThreshold {
		return fmt.Sprintf("🚨 Final warning: %d/%d steps used. Only 2 steps remaining! Please provide final answer.", step, maxSteps)
	}

	return ""
}

// ProgressEventType identifies a progress event kind.
type ProgressEventType string

const (
	ProgressToolCall   ProgressEventType = "tool_call"
	ProgressToolResult ProgressEventType = "tool_result"
)

// ProgressEvent is emitted during ProcessStream to report tool call progress.
// Handlers can serialize this to SSE so the frontend shows real-time tool execution status.
type ProgressEvent struct {
	Type     ProgressEventType `json:"type"`
	ToolName string            `json:"tool_name"`
	Args     string            `json:"args,omitempty"`
	Result   string            `json:"result,omitempty"`
	IsError  bool              `json:"is_error,omitempty"`
}

// toolCallResult pairs a tool call with its execution result.
type toolCallResult struct {
	call   llm.ToolCall
	output string
	err    error
}

// executeToolCallsParallel runs all tool calls from one LLM response concurrently.
func (a *ReActAgent) executeToolCallsParallel(ctx context.Context, calls []llm.ToolCall, detector *loopDetector, channel, chatID, session, user string) []toolCallResult {
	results := make([]toolCallResult, len(calls))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i, tc := range calls {
		wg.Add(1)
		go func(idx int, call llm.ToolCall) {
			defer wg.Done()

			mu.Lock()
			loopErr := detector.checkCall(call.Function.Name, call.Function.Arguments)
			mu.Unlock()

			if loopErr != nil {
				mu.Lock()
				results[idx] = toolCallResult{call: call, output: loopErr.Error(), err: loopErr}
				mu.Unlock()
				return
			}

			output, execErr := a.toolExecutor.Execute(ctx, call.Function.Name, call.Function.Arguments, channel, chatID, session, user)

			mu.Lock()
			if execErr != nil {
				detector.recordFailure()
				output = fmt.Sprintf("Tool error: %v", execErr)
			} else {
				detector.recordSuccess()
			}
			results[idx] = toolCallResult{call: call, output: output, err: execErr}
			mu.Unlock()
		}(i, tc)
	}

	wg.Wait()
	return results
}

// Process handles one user request through the native Function Calling loop.
func (a *ReActAgent) Process(ctx context.Context, req *types.Request) (*types.Response, error) {
	if req.SessionID == "" {
		req.SessionID = "default"
	}

	// Retrieve (or create) the session.
	s, err := a.sessionManager.GetOrCreate(req.SessionID, req.UserID, req.Channel)
	if err != nil {
		return nil, fmt.Errorf("agent: session: %w", err)
	}

	// [P0] Acquire session lock to prevent concurrent requests to the same session.
	if err := s.Acquire(); err != nil {
		return nil, err
	}
	defer s.Release()

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

	// Attempt to compress memory (if context is too long); async, non-blocking.
	a.memoryManager.MaybeCompress(req.SessionID)
	// Reload context after potential compression.
	history, _ = a.memoryManager.GetContext(req.SessionID, 0)

	// Build system prompt dynamically using ContextBuilder.
	// This includes persona, memories (Core + Daily + Relevant), skills, and time context.
	allTools := a.toolRegistry.All()
	var systemPrompt string
	if a.contextBuilder != nil {
		ctxResult, err := a.contextBuilder.Build(ctx, req.SessionID, req.Content)
		if err != nil {
			a.logger.Warn("failed to build dynamic context, using base persona", zap.Error(err))
			systemPrompt = a.currentBasePrompt() + "\n\n---\n" + buildCapabilityFragment(allTools)
		} else {
			systemPrompt = ctxResult.SystemPrompt + "\n\n---\n" + buildCapabilityFragment(allTools)
			a.logger.Debug("dynamic context built",
				zap.Int("memories_used", ctxResult.MemoriesUsed),
				zap.Int("skills_matched", ctxResult.SkillsMatched),
				zap.Duration("build_time", ctxResult.BuildTime),
			)
		}
	} else {
		// Fallback to base persona if context builder is not initialized
		systemPrompt = a.currentBasePrompt() + "\n\n---\n" + buildCapabilityFragment(allTools)
	}
	toolDefs := buildToolDefinitions(allTools)

	// Assemble the initial message list (req.Files triggers media manifest injection).
	messages := buildMessages(systemPrompt, history, req.Content, req.Files)

	maxSteps := a.maxSteps
	if maxSteps <= 0 {
		maxSteps = 20
	}

	detector := newLoopDetector(5, 3)

	var fullContent strings.Builder
	for step := 0; step < maxSteps; step++ {
		a.logger.Info("agent: fc step",
			zap.Int("step", step),
			zap.Int("max_steps", maxSteps),
			zap.String("session", req.SessionID),
		)

		// Check tool step limit and inject warning if needed
		if warning := checkToolStepLimit(step, maxSteps); warning != "" {
			a.logger.Warn("tool step limit warning",
				zap.String("warning", warning),
				zap.String("session", req.SessionID),
			)
			// Inject warning into messages for LLM to see
			messages = append(messages, llm.ChatMessage{
				Role:    llm.RoleSystem,
				Content: warning,
			})
		}

		// PreReasoning hooks: may modify messages (e.g. inject current time).
		hooked, hookErr := a.hooks.runPreReasoning(ctx, messages)
		if hookErr != nil {
			return nil, hookErr
		}

		resp, err := a.llmClient.Chat(ctx, llm.ChatRequest{
			Messages: hooked,
			Tools:    toolDefs,
		})
		if err != nil {
			return nil, fmt.Errorf("agent: llm call step %d: %w", step, err)
		}

		// Context Window Guard: check token usage reported by the provider.
		if pt := resp.Usage.PromptTokens; pt > 0 {
			if pt >= contextHardLimitTokens {
				return nil, fmt.Errorf("agent: context window exhausted (%d prompt tokens, limit %d) — please start a new session", pt, contextHardLimitTokens)
			}
			if pt >= contextWarnTokens {
				a.logger.Warn("agent: context window approaching limit",
					zap.Int("prompt_tokens", pt),
					zap.Int("warn_threshold", contextWarnTokens),
					zap.String("session", req.SessionID),
				)
			}
		}

		// Append the assistant message (may contain tool_calls) to the conversation.
		messages = append(messages, resp.Message)

		// Accumulate content if present in this turn.
		if resp.Message.Content != "" {
			if fullContent.Len() > 0 {
				fullContent.WriteString("\n")
			}
			fullContent.WriteString(resp.Message.Content)
		}

		// No tool calls → the model has produced its final answer.
		if len(resp.Message.ToolCalls) == 0 {
			a.logger.Info("agent: final answer",
				zap.Int("step", step),
				zap.String("finish_reason", resp.FinishReason),
				zap.String("session", req.SessionID),
			)
			break
		}

		// Execute all tool calls from this step (parallel).
		a.logger.Info("agent: executing tool calls",
			zap.Int("step", step),
			zap.Int("count", len(resp.Message.ToolCalls)),
			zap.String("session", req.SessionID),
		)

		results := a.executeToolCallsParallel(ctx, resp.Message.ToolCalls, detector, req.Channel, req.ChatID, req.SessionID, req.UserID)

		// Check failure streak after the batch.
		if streakErr := detector.checkFailureStreak(); streakErr != nil {
			return nil, streakErr
		}

		// Log, hook, and append tool result messages.
		for _, r := range results {
			a.logger.Info("agent: tool result",
				zap.Int("step", step),
				zap.String("tool", r.call.Function.Name),
				zap.String("result", truncate(r.output, 200)),
			)

			if a.convlog != nil {
				rawArgs := json.RawMessage(r.call.Function.Arguments)
				if err := a.convlog.LogToolCall(req.SessionID, r.call.Function.Name, rawArgs); err != nil {
					a.logger.Warn("agent: failed to log tool call", zap.Error(err))
				}
				var errPtr *string
				if r.err != nil {
					s := r.err.Error()
					errPtr = &s
				}
				if err := a.convlog.LogToolResult(req.SessionID, r.call.Function.Name, r.output, errPtr); err != nil {
					a.logger.Warn("agent: failed to log tool result", zap.Error(err))
				}
			}

			// PostTool hooks: fire-and-forget side effects (logging, metrics, etc.).
			a.hooks.runPostTool(ctx, r.call.Function.Name, r.output, r.err)

			// Record skill usage for learning (if tool belongs to a skill)
			if a.skillManager != nil {
				if skillName := a.skillManager.GetSkillByTool(r.call.Function.Name); skillName != "" {
					a.skillManager.RecordSkillUsage(skillName)
					a.logger.Debug("skill usage recorded",
						zap.String("tool", r.call.Function.Name),
						zap.String("skill", skillName),
					)
				}
			}

			messages = append(messages, llm.ChatMessage{
				Role:       llm.RoleTool,
				ToolCallID: r.call.ID,
				Name:       r.call.Function.Name,
				Content:    r.output,
			})
		}
	}

	// Max steps reached - add termination message and get final answer from LLM
	a.logger.Warn("agent: max steps reached, forcing final answer",
		zap.Int("max_steps", maxSteps),
		zap.String("session", req.SessionID),
	)

	// Inject system message to force final answer
	messages = append(messages, llm.ChatMessage{
		Role:    llm.RoleSystem,
		Content: fmt.Sprintf("🛑 Maximum tool calls (%d) reached. You must provide your final answer now based on the information gathered. Do not make any more tool calls.", maxSteps),
	})

	// Make one final LLM call to get the answer
	finalResp, err := a.llmClient.Chat(ctx, llm.ChatRequest{
		Messages: messages,
		Tools:    nil, // No tools - force text response
	})
	if err != nil {
		return nil, fmt.Errorf("agent: failed to get final answer after max steps: %w", err)
	}

	finalAnswer := fullContent.String()
	if finalResp.Message.Content != "" {
		finalAnswer = finalResp.Message.Content
	}
	if finalAnswer == "" {
		return nil, fmt.Errorf("agent: max steps reached without any content")
	}

	// Log agent reply.
	if a.convlog != nil {
		if err := a.convlog.LogAgentReply(req.SessionID, finalAnswer, nil); err != nil {
			a.logger.Warn("agent: failed to log agent reply", zap.Error(err))
		}
	}

	// Persist the exchange to memory.
	// We save the enriched user content (including media manifest) so that media://
	// references remain visible in conversation history for follow-up turns.
	userContentForMemory := req.Content
	if manifest := buildMediaManifest(req.Files); manifest != "" {
		userContentForMemory += manifest
	}
	if memErr := a.memoryManager.Add(req.SessionID, req.UserID, req.Channel, userContentForMemory, finalAnswer); memErr != nil {
		a.logger.Warn("agent: failed to save memory", zap.Error(memErr))
	}

	return &types.Response{
		Content: finalAnswer,
		MsgType: types.MsgTypeText,
	}, nil
}

// ProcessStream handles one user request through the native Function Calling loop.
// It uses blocking Chat() calls for each step (not streaming), keeping the logic
// simple and reliable. Once the final answer is ready, deltaFn is called once
// with the complete text. The frontend typewriter produces the output animation.
//
// progressFn is called before and after each tool execution; pass nil to disable.
// Returns the full final answer text and any error.
func (a *ReActAgent) ProcessStream(ctx context.Context, req *types.Request, progressFn func(ProgressEvent), deltaFn func(string)) (string, error) {
	if req.SessionID == "" {
		req.SessionID = "default"
	}

	s, err := a.sessionManager.GetOrCreate(req.SessionID, req.UserID, req.Channel)
	if err != nil {
		return "", fmt.Errorf("agent: session: %w", err)
	}

	// [P0] Acquire session lock to prevent concurrent requests to the same session.
	if err := s.Acquire(); err != nil {
		return "", err
	}
	defer s.Release()

	if a.convlog != nil {
		if err := a.convlog.LogUserMessage(req.SessionID, req.Content); err != nil {
			a.logger.Warn("agent: failed to log user message", zap.Error(err))
		}
	}

	history, err := a.memoryManager.GetContext(req.SessionID, 0)
	if err != nil {
		a.logger.Warn("agent: failed to load memory", zap.Error(err))
		history = nil
	}

	a.memoryManager.MaybeCompress(req.SessionID)
	history, _ = a.memoryManager.GetContext(req.SessionID, 0)

	// Build system prompt dynamically using ContextBuilder
	allTools := a.toolRegistry.All()
	var systemPrompt string
	if a.contextBuilder != nil {
		ctxResult, err := a.contextBuilder.Build(ctx, req.SessionID, req.Content)
		if err != nil {
			a.logger.Warn("failed to build dynamic context, using base persona", zap.Error(err))
			systemPrompt = a.currentBasePrompt() + "\n\n---\n" + buildCapabilityFragment(allTools)
		} else {
			systemPrompt = ctxResult.SystemPrompt + "\n\n---\n" + buildCapabilityFragment(allTools)
		}
	} else {
		systemPrompt = a.currentBasePrompt() + "\n\n---\n" + buildCapabilityFragment(allTools)
	}
	toolDefs := buildToolDefinitions(allTools)
	messages := buildMessages(systemPrompt, history, req.Content, req.Files)

	maxSteps := a.maxSteps
	if maxSteps <= 0 {
		maxSteps = 20
	}

	detector := newLoopDetector(5, 3)

	var fullContent strings.Builder

	for step := 0; step < maxSteps; step++ {
		a.logger.Info("agent: step",
			zap.Int("step", step),
			zap.String("session", req.SessionID),
		)

		// Check tool step limit and inject warning if needed
		if warning := checkToolStepLimit(step, maxSteps); warning != "" {
			a.logger.Warn("tool step limit warning",
				zap.String("warning", warning),
				zap.String("session", req.SessionID),
			)
			// Inject warning into messages for LLM to see
			messages = append(messages, llm.ChatMessage{
				Role:    llm.RoleSystem,
				Content: warning,
			})
		}

		// 显式推送当前步骤，触发前端视觉反馈
		if progressFn != nil {
			progressFn(ProgressEvent{
				Type:     "step",
				ToolName: fmt.Sprintf("Step %d", step+1),
			})
		}

		hooked, hookErr := a.hooks.runPreReasoning(ctx, messages)
		if hookErr != nil {
			return "", hookErr
		}

		resp, err := a.llmClient.Chat(ctx, llm.ChatRequest{
			Messages: hooked,
			Tools:    toolDefs,
		})
		if err != nil {
			return "", fmt.Errorf("agent: llm step %d: %w", step, err)
		}

		// Immediate streaming: call deltaFn if this turn produced text.
		if resp.Message.Content != "" && deltaFn != nil {
			deltaFn(resp.Message.Content)
			if fullContent.Len() > 0 {
				fullContent.WriteString("\n")
			}
			fullContent.WriteString(resp.Message.Content)
		}

		// Context window guard.
		if pt := resp.Usage.PromptTokens; pt > 0 {
			if pt >= contextHardLimitTokens {
				return "", fmt.Errorf("agent: context window exhausted (%d prompt tokens) — please start a new session", pt)
			}
			if pt >= contextWarnTokens {
				a.logger.Warn("agent: context window approaching limit",
					zap.Int("prompt_tokens", pt),
					zap.String("session", req.SessionID),
				)
			}
		}

		messages = append(messages, resp.Message)

		// No tool calls → reasoning cycle is complete.
		if len(resp.Message.ToolCalls) == 0 {
			a.logger.Info("agent: sequence complete",
				zap.Int("step", step),
				zap.String("session", req.SessionID),
			)
			break
		}

		a.logger.Info("agent: executing tool calls",
			zap.Int("step", step),
			zap.Int("count", len(resp.Message.ToolCalls)),
			zap.String("session", req.SessionID),
		)

		if progressFn != nil {
			for _, tc := range resp.Message.ToolCalls {
				progressFn(ProgressEvent{
					Type:     ProgressToolCall,
					ToolName: tc.Function.Name,
					Args:     tc.Function.Arguments,
				})
			}
		}

		results := a.executeToolCallsParallel(ctx, resp.Message.ToolCalls, detector, req.Channel, req.ChatID, req.SessionID, req.UserID)
		if streakErr := detector.checkFailureStreak(); streakErr != nil {
			return "", streakErr
		}

		for _, r := range results {
			a.logger.Info("agent: tool result",
				zap.Int("step", step),
				zap.String("tool", r.call.Function.Name),
				zap.String("result", truncate(r.output, 200)),
			)
			if a.convlog != nil {
				rawArgs := json.RawMessage(r.call.Function.Arguments)
				_ = a.convlog.LogToolCall(req.SessionID, r.call.Function.Name, rawArgs)
				var ep *string
				if r.err != nil {
					s := r.err.Error()
					ep = &s
				}
				_ = a.convlog.LogToolResult(req.SessionID, r.call.Function.Name, r.output, ep)
			}
			a.hooks.runPostTool(ctx, r.call.Function.Name, r.output, r.err)

			// Record skill usage for learning (if tool belongs to a skill)
			if a.skillManager != nil {
				if skillName := a.skillManager.GetSkillByTool(r.call.Function.Name); skillName != "" {
					a.skillManager.RecordSkillUsage(skillName)
					a.logger.Debug("skill usage recorded",
						zap.String("tool", r.call.Function.Name),
						zap.String("skill", skillName),
					)
				}
			}

			if progressFn != nil {
				progressFn(ProgressEvent{
					Type:     ProgressToolResult,
					ToolName: r.call.Function.Name,
					Result:   truncate(r.output, 300),
					IsError:  r.err != nil,
				})
			}

			messages = append(messages, llm.ChatMessage{
				Role:       llm.RoleTool,
				ToolCallID: r.call.ID,
				Name:       r.call.Function.Name,
				Content:    r.output,
			})
		}
	}

	// Max steps reached - add termination message and get final answer from LLM
	a.logger.Warn("agent: max steps reached, forcing final answer",
		zap.Int("max_steps", maxSteps),
		zap.String("session", req.SessionID),
	)

	// Inject system message to force final answer
	messages = append(messages, llm.ChatMessage{
		Role:    llm.RoleSystem,
		Content: fmt.Sprintf("🛑 Maximum tool calls (%d) reached. You must provide your final answer now based on the information gathered. Do not make any more tool calls.", maxSteps),
	})

	// Make one final LLM call to get the answer
	finalResp, err := a.llmClient.Chat(ctx, llm.ChatRequest{
		Messages: messages,
		Tools:    nil, // No tools - force text response
	})
	if err != nil {
		return "", fmt.Errorf("agent: failed to get final answer after max steps: %w", err)
	}

	finalAnswer := fullContent.String()
	if finalResp.Message.Content != "" {
		finalAnswer = finalResp.Message.Content
		// Stream the final answer if deltaFn is provided
		if deltaFn != nil {
			deltaFn(finalResp.Message.Content)
		}
	}
	if finalAnswer == "" {
		return "", fmt.Errorf("agent: max steps reached without any content")
	}

	if a.convlog != nil {
		_ = a.convlog.LogAgentReply(req.SessionID, finalAnswer, nil)
	}

	if memErr := a.memoryManager.Add(req.SessionID, req.UserID, req.Channel, req.Content, finalAnswer); memErr != nil {
		a.logger.Warn("agent: failed to save memory", zap.Error(memErr))
	}

	return finalAnswer, nil
}

// truncate shortens s to at most n runes, appending "…" if cut.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

// Sessions returns the session manager for use by the HTTP handlers.
func (a *ReActAgent) Sessions() *SessionManager {
	return a.sessionManager
}

// Executor returns the tool executor.
func (a *ReActAgent) Executor() *tool.Executor {
	return a.toolExecutor
}
