// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

// Command gopaw is the main entry point for the GoPaw AI assistant workbench.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gopaw/gopaw/internal/agent"
	"github.com/gopaw/gopaw/internal/agent/message"
	"github.com/gopaw/gopaw/internal/alert"
	"github.com/gopaw/gopaw/internal/audit"
	"github.com/gopaw/gopaw/internal/auth"
	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/internal/config"
	"github.com/gopaw/gopaw/internal/convlog"
	"github.com/gopaw/gopaw/internal/cron"
	"github.com/gopaw/gopaw/internal/flow"
	"github.com/gopaw/gopaw/internal/focus"
	"github.com/gopaw/gopaw/internal/knowledge"
	"github.com/gopaw/gopaw/internal/llm"
	"github.com/gopaw/gopaw/internal/mcp"
	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/internal/mode"
	"github.com/gopaw/gopaw/internal/sandbox"
	"github.com/gopaw/gopaw/internal/server"
	"github.com/gopaw/gopaw/internal/settings"
	"github.com/gopaw/gopaw/internal/skill"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/internal/tool/builtin"
	"github.com/gopaw/gopaw/internal/trace"
	"github.com/gopaw/gopaw/internal/tray"
	"github.com/gopaw/gopaw/internal/queue"
	"github.com/gopaw/gopaw/internal/metrics"
	"github.com/gopaw/gopaw/internal/permission"
	"github.com/gopaw/gopaw/internal/resource"
	"github.com/gopaw/gopaw/internal/user"
	"github.com/gopaw/gopaw/internal/workspace"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
	"github.com/gopaw/gopaw/web"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	_ "github.com/gopaw/gopaw/internal/platform/console"
	_ "github.com/gopaw/gopaw/internal/platform/dingtalk"
	_ "github.com/gopaw/gopaw/internal/platform/feishu"
	_ "github.com/gopaw/gopaw/internal/platform/webhook"
)

var appVersion = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "init":
		runInit()
	case "start":
		runStart()
	case "version":
		runVersion()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `GoPaw - Unified AI Assistant Workbench

Usage:
  gopaw <command> [flags]

Commands:
  init      Generate a default config.yaml
  start     Start the GoPaw server
  version   Print version info

Start flags:
  --config string   Path to config file (default "config.yaml")
  --mode  string    Deployment mode: solo|team|cloud (overrides config app.mode)
  --tray            Run as system tray app (requires -tags tray build)
  --no-browser      Suppress auto-opening browser on startup`)
}

func runInit() {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	output := fs.String("output", "config.yaml", "Output file path")
	fs.Parse(os.Args[2:])

	if _, err := os.Stat(*output); err == nil {
		fmt.Fprintf(os.Stderr, "config file %q already exists\n", *output)
		os.Exit(1)
	}

	const defaultConfig = `workspace:
  dir: ~/.gopaw
app:
  name: "GoPaw"
  language: zh
  timezone: Asia/Shanghai
  # mode: solo | team | cloud
  # solo  - single user, no login required (default)
  # team  - multi-user with JWT auth (≤50 users, admin-managed)
  # cloud - JWT auth + invite codes, open registration
  mode: solo
server:
  host: 0.0.0.0
  port: 16688
agent:
  max_steps: 20
  memory:
    context_limit: 4000
    history_limit: 50
log:
  level: info
  format: json
  output: stdout`
	os.WriteFile(*output, []byte(defaultConfig), 0o644)
	fmt.Printf("Generated config file: %s\n", *output)
}

func runVersion() {
	fmt.Printf("GoPaw version %s\nGo %s %s/%s\n", appVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

func runStart() {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	cfgFile := fs.String("config", "config.yaml", "Path to config file")
	modeFlag := fs.String("mode", "", "Deployment mode: solo|team|cloud (overrides config app.mode)")
	trayFlag := fs.Bool("tray", false, "Run as system tray app (auto-opens browser, requires -tags tray build)")
	noBrowserFlag := fs.Bool("no-browser", false, "Suppress auto-opening browser on startup")
	fs.Parse(os.Args[2:])
	_ = trayFlag // systray integration handled below after logger is ready

	preLogger, _ := zap.NewProduction()
	cfgMgr, err := config.NewManager(*cfgFile, preLogger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}
	cfg := cfgMgr.Get()

	// --mode flag overrides config app.mode
	appMode := mode.Parse(cfg.App.Mode)
	if *modeFlag != "" {
		appMode = mode.Parse(*modeFlag)
	}

	wp, err := workspace.Resolve(cfg.Workspace.Dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving workspace: %v\n", err)
		os.Exit(1)
	}
	if err := workspace.EnsureDirs(wp); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating workspace directories: %v\n", err)
		os.Exit(1)
	}

	logger, _ := buildLogger(cfg.Log, wp.LogFile)
	defer logger.Sync()

	// ---------- Media Store ----------
	mediaBaseDir := filepath.Join(wp.Root, "media")
	mediaStore := channel.NewMediaStore(mediaBaseDir, 1*time.Hour, logger)
	defer mediaStore.Close()

	store, _ := memory.NewStore(wp.DBFile)
	defer store.Close()
	settingsStore := settings.NewStore(store.DB())

	llmClient := llm.NewFallbackClient(func() ([]llm.ProviderEntry, error) {
		providers, _ := settingsStore.GetProvidersByPriority()
		entries := make([]llm.ProviderEntry, len(providers))
		for i, p := range providers {
			entries[i] = llm.ProviderEntry{ID: p.ID, Name: p.Name, BaseURL: p.BaseURL, APIKey: p.APIKey, Model: p.Model, MaxTokens: p.MaxTokens, TimeoutSec: p.TimeoutSec}
		}
		return entries, nil
	}, logger)

	// In multi-agent mode, each agent has its own system prompt.
	// The default prompt here is only used as a fallback for legacy single-agent mode.
	basePrompt := "You are a helpful AI assistant."

	memMgr := memory.NewManager(store, llmClient, cfg.Agent.Memory.ContextLimit, cfg.Agent.Memory.HistoryLimit, logger)
	memMgr.SetArchiveDir(wp.MemoryArchDir)

	ltmStore, _ := memory.NewLTMStore(wp.MemoriesDBFile)
	defer ltmStore.Close()

	hygieneRunner := memory.NewHygieneRunner(store, ltmStore, wp.MemoryNotesDir, wp.MemoryArchDir, memory.HygieneConfig{}, logger)

	toolReg := tool.Global()
	toolReg.SetMediaStore(mediaStore)

	// Load MCP Servers if configured
	if len(cfg.MCPServers) > 0 {
		mcpConfigs := make([]tool.MCPServerConfig, len(cfg.MCPServers))
		for i, c := range cfg.MCPServers {
			mcpConfigs[i] = tool.MCPServerConfig{
				Name:    c.Name,
				Command: c.Command,
				Args:    c.Args,
			}
		}
		if err := toolReg.LoadMCPServers(context.Background(), mcpConfigs); err != nil {
			logger.Warn("failed to load some MCP servers", zap.Error(err))
		}
	}

	builtin.SetMemoryDir(wp.MemoryDir)
	builtin.SetLTMStore(ltmStore)
	builtin.SetMemoryNotesDir(wp.MemoryNotesDir)
	builtin.SetWorkspaceRoot(wp.Root)
	skillMgr := skill.NewManager(wp.SkillsDir, toolReg, nil, logger)
	skillMgr.Load(cfg.Skills.Enabled)
	builtin.SetSkillManager(skillMgr)

	convLogger, _ := convlog.New(wp.ConvLogFile, logger)

	// Initialize focus manager
	focusPath := filepath.Join(wp.Root, "FOCUS.md")
	focusMgr := focus.NewManager(focusPath, logger)
	if err := focusMgr.Load(); err != nil {
		logger.Warn("failed to load focus file", zap.Error(err))
	}

	// Set focus manager for the update_focus tool
	if focusTool, ok := toolReg.Get("update_focus"); ok {
		if ft, ok := focusTool.(*builtin.FocusUpdateTool); ok {
			ft.SetFocusManager(focusMgr)
		}
	}

	// Initialize trace manager
	traceDBPath := filepath.Join(wp.Root, "traces.db")
	traceMgr, err := trace.NewManager(traceDBPath, 7, logger) // 7 days retention
	if err != nil {
		logger.Warn("failed to initialize trace manager", zap.Error(err))
		traceMgr = nil
	}

	// Initialize sandbox manager
	sandboxRoot := filepath.Join(wp.Root, "sessions")
	sandboxMgr, err := sandbox.NewManager(sandboxRoot, logger)
	if err != nil {
		logger.Warn("failed to initialize sandbox manager", zap.Error(err))
		sandboxMgr = nil
	}

	agentInstance := agent.New(llmClient, toolReg, skillMgr, memMgr, agent.Config{
		DefaultPrompt:  basePrompt,
		AgentMDPath:    wp.AgentMDFile,
		LTMStore:       ltmStore,
		MemoryNotesDir: wp.MemoryNotesDir,
		MaxSteps:       cfg.Agent.MaxSteps,
		Hooks: agent.Hooks{
			PreReasoning: []agent.HookPreReasoning{agent.InjectCurrentTime()},
			PostTool:     []agent.HookPostTool{agent.AutoJournalHook(wp.MemoryNotesDir)},
		},
		ConvLog:        convLogger,
		FocusManager:   focusMgr,
		TraceManager:   traceMgr,
		SandboxManager: sandboxMgr,
	}, logger)

	builtin.SetSubAgentFn(func(ctx context.Context, req *types.Request) (string, error) {
		resp, _ := agentInstance.Process(ctx, req)
		return resp.Content, nil
	})

	channelMgr := channel.NewManager(channel.Global(), mediaStore, logger)
	pluginCfgs := buildPluginConfigsFromDB(settingsStore, logger)

	// --- Cron System Setup ---
	cronService := cron.NewCronService(wp.Root, logger)

	// Inject dependencies into built-in tools
	for _, t := range toolReg.All() {
		if ct, ok := t.(interface{ SetCronService(*cron.CronService) }); ok {
			ct.SetCronService(cronService)
		}
		if mt, ok := t.(interface{ SetMemoryManager(*memory.Manager) }); ok {
			mt.SetMemoryManager(memMgr)
		}
		if st, ok := t.(interface{ SetSettingsStore(*settings.Store) }); ok {
			st.SetSettingsStore(settingsStore)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hygieneRunner.Start(ctx)
	if err := cronService.Start(); err != nil {
		logger.Error("failed to start cron service", zap.Error(err))
	}
	defer cronService.Stop()

	channelMgr.Start(ctx, pluginCfgs)
	defer channelMgr.Stop()

	sessionLocker := channel.NewSessionLocker()
	coord := channel.NewCapabilityCoordinator(channelMgr, mediaStore)

	// Create Web Console approval UI
	webConsoleUI := server.NewWebConsoleApprovalUI(tool.GlobalApprovalStore, logger)

	// Create composite approval UI that supports both Feishu and Web Console
	compositeUI := server.NewCompositeApprovalUI(coord, webConsoleUI, logger)

	// Connect approval UI to tool executor
	agentInstance.SetApprovalUI(compositeUI)

	// Connect L2 notification callback to tool executor
	agentInstance.SetL2NotificationCallback(func(ctx context.Context, toolName string, args map[string]interface{}, channel, chatID, session string) {
		logger.Info("L2 tool executed",
			zap.String("tool", toolName),
			zap.String("channel", channel),
			zap.String("session", session),
		)
		// TODO: Send notification to user via WebSocket or Feishu
	})

	// Setup hot-reload for configuration files
	hotReloadCancels := setupHotReload(agentInstance, skillMgr, focusMgr, wp, logger)
	// Register cleanup on shutdown
	defer func() {
		for _, cancel := range hotReloadCancels {
			cancel()
		}
	}()

	// Connect immediate result callback to tool executor.
	// chatID is the platform-level chat room ID (e.g. Feishu oc_xxx) threaded from req.ChatID.
	agentInstance.Executor().SetResultCallback(func(ctx context.Context, channel, chatID, session, user string, result *plugin.ToolResult) {
		msg := &types.Message{
			Channel:   channel,
			ChatID:    chatID, // real platform chat ID, required by feishu receive_id_type=chat_id
			UserID:    user,
			Content:   result.UserOutput,
			MsgType:   types.MsgTypeText,
			SessionID: session,
		}

		if len(result.Media) > 0 {
			for _, ref := range result.Media {
				_, meta, err := mediaStore.ResolveWithMeta(ref)
				if err == nil {
					msg.Files = append(msg.Files, types.FileAttachment{
						Name:     meta.Filename,
						URL:      ref,
						MIMEType: meta.ContentType,
					})
				}
			}
		}

		if msg.Content != "" || len(msg.Files) > 0 {
			if err := channelMgr.Send(msg); err != nil {
				logger.Error("result callback: send failed", zap.Error(err))
			}
		}
	})

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-channelMgr.Messages():
				if !ok {
					return
				}
				go func(m *types.Message) {
					derivedSessionID := agent.DeriveSessionID(agent.StrategyPerSender, m.Channel, m.UserID, m.ChatID)
					unlock := sessionLocker.Lock(derivedSessionID)
					defer unlock()

					req := &types.Request{
						SessionID:   derivedSessionID,
						UserID:      m.UserID,
						ChatID:      m.ChatID,
						Channel:     m.Channel,
						Content:     m.Content,
						MsgType:     m.MsgType,
						Files:       m.Files,
						Metadata:    m.Metadata,
						IsMentioned: m.IsMentioned,
					}
					coord.PreProcess(ctx, m)
					resp, err := agentInstance.Process(ctx, req)
					if err != nil {
						logger.Error("agent processing error", zap.Error(err))
						return
					}
					reply := &types.Message{
						SessionID: derivedSessionID,
						UserID:    m.UserID,
						Channel:   m.Channel,
						Content:   resp.Content,
						MsgType:   resp.MsgType,
						ReplyTo:   m.ID,
						ChatID:    m.ChatID,
						ThreadID:  m.ThreadID,
					}
					_ = coord.PostProcess(ctx, m, reply)
				}(msg)
			}
		}
	}()

	adminToken := cfg.App.AdminToken
	if adminToken == "" {
		adminToken = generateToken()
		logger.Info("⚡ Admin token", zap.String("token", adminToken))
	}

	// --- Auth + User Services (team/cloud mode only) ---
	var authSvc *auth.Service
	var userSvc *user.Service
	if appMode.IsMultiUser() {
		authSvc = auth.NewService(auth.DefaultConfig())
		userStore, err := user.NewStore(store.DB())
		if err != nil {
			logger.Warn("failed to initialize user store", zap.Error(err))
		} else {
			userSvc = user.NewService(userStore)
			// Ensure there is at least one admin user on first launch.
			if _, err := userSvc.EnsureAdminExists("admin", adminToken); err != nil {
				logger.Warn("failed to ensure admin user exists", zap.Error(err))
			} else {
				logger.Info("team mode: admin user ready", zap.String("username", "admin"))
			}
		}
	}

	// Initialize agent manager for multi-agent support
	agentsDBPath := filepath.Join(wp.Root, "agents.db")
	agentsDir := filepath.Join(wp.Root, "agents")
	agentMgr, err := agent.NewManager(agentsDBPath, agentsDir, logger)
	if err != nil {
		logger.Warn("failed to initialize agent manager", zap.Error(err))
		agentMgr = nil
	}

	// Initialize MCP manager
	mcpMgr, err := mcp.NewManager(store.DB(), logger)
	if err != nil {
		logger.Warn("failed to initialize mcp manager", zap.Error(err))
		mcpMgr = nil
	} else {
		// Create builtin MCP servers
		if err := mcpMgr.CreateBuiltinServers(wp.Root); err != nil {
			logger.Warn("failed to create builtin mcp servers", zap.Error(err))
		}
	}

	// Initialize agent message manager
	agentMsgMgr, err := message.NewManager(store.DB(), logger)
	if err != nil {
		logger.Warn("failed to initialize agent message manager", zap.Error(err))
		agentMsgMgr = nil
	}

	// Initialize queue manager
	queueMgr, err := queue.NewManager(store.DB(), logger)
	if err != nil {
		logger.Warn("failed to initialize queue manager", zap.Error(err))
		queueMgr = nil
	}

	// Initialize audit manager
	if err := audit.InitSchema(store.DB()); err != nil {
		logger.Warn("failed to initialize audit schema", zap.Error(err))
	}
	auditMgr, err := audit.NewManager(store.DB(), logger)
	if err != nil {
		logger.Warn("failed to initialize audit manager", zap.Error(err))
		auditMgr = nil
	}

	// Initialize metrics service
	metricsService, err := metrics.NewService(store.DB(), logger)
	if err != nil {
		logger.Warn("failed to initialize metrics service", zap.Error(err))
		metricsService = nil
	}

	// Initialize knowledge base service
	var knowledgeService *knowledge.Service
	if err := knowledge.InitSchema(store.DB()); err != nil {
		logger.Warn("failed to initialize knowledge schema", zap.Error(err))
	} else {
		knowledgeService = knowledge.NewService(store.DB())
		// Register knowledge tools
		registerKnowledgeTools(toolReg, knowledgeService, logger)
		logger.Info("knowledge service initialized")
	}

	// Initialize agent factory and router for dynamic agent switching (after knowledge service)
	var agentRouter *agent.Router
	if agentMgr != nil {
		agentFactory := agent.NewFactory(agent.FactoryConfig{
			LLMClient:         llmClient,
			ToolRegistry:      toolReg,
			SkillManager:      skillMgr,
			MemoryManager:     memMgr,
			LTMStore:          ltmStore,
			SandboxManager:    sandboxMgr,
			TraceManager:      traceMgr,
			WorkspaceRoot:     wp.Root,
			Logger:            logger,
			GlobalKnowledgeFn: func(ctx context.Context) (string, error) {
				if knowledgeService != nil {
					return knowledgeService.GetGlobalInjectContent(ctx)
				}
				return "", nil
			},
		})

		sessionAgentDBPath := filepath.Join(wp.Root, "session_agents.db")
		agentRouter, err = agent.NewRouter(agentMgr, agentFactory, sessionAgentDBPath, logger)
		if err != nil {
			logger.Warn("failed to initialize agent router", zap.Error(err))
			agentRouter = nil
		}
	}

	// Configure cron handler now that agentRouter is available.
	cronService.SetHandler(func(job *cron.CronJob) (string, error) {
		// Use job name as session ID for readability in conversation history.
		sessionID := sanitizeCronName(job.Name)

		logger.Info("cron: executing job",
			zap.String("job", job.Name),
			zap.String("task", job.Task),
			zap.String("target", job.TargetID))

		// Select agent: prefer the agent that created this job (job.TargetID),
		// fall back to the default agent instance when not found.
		runAgent := agentInstance
		if job.TargetID != "" && agentRouter != nil {
			if a, routerErr := agentRouter.GetOrCreateAgent(job.TargetID); routerErr == nil {
				runAgent = a
			} else {
				logger.Warn("cron: agent not found, using default",
					zap.String("target_id", job.TargetID),
					zap.Error(routerErr))
			}
		}

		req := &types.Request{
			SessionID: sessionID,
			Channel:   job.Channel,
			ChatID:    job.TargetID,
			UserID:    "cron",
			Content:   job.Task,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		resp, err := runAgent.Process(ctx, req)
		if err != nil {
			logger.Error("cron: job execution failed", zap.Error(err))
			return "", err
		}

		if resp.Content != "" {
			msg := &types.Message{
				Channel:   job.Channel,
				ChatID:    job.TargetID,
				Content:   resp.Content,
				MsgType:   types.MsgTypeText,
				SessionID: sessionID,
			}
			if err := channelMgr.Send(msg); err != nil {
				logger.Error("cron: failed to send final answer", zap.Error(err))
				return fmt.Sprintf("Executed. Content: %s. Send Error: %v", resp.Content, err), nil
			}
		}

		return resp.Content, nil
	})

	// Initialize flow service (unified flow management)
	var flowService *flow.Service
	if store != nil && agentMgr != nil && agentMsgMgr != nil {
		flowService, err = flow.NewService(store.DB(), agentMgr, agentMsgMgr, logger)
		if err != nil {
			logger.Warn("failed to initialize flow service", zap.Error(err))
		} else {
			// Set agent router for execution engine
			if flowService != nil && agentRouter != nil {
				flowService.SetAgentRouter(agentRouter)
			}
			// Set cron service for triggers
			if flowService != nil && cronService != nil {
				flowService.SetCronService(cronService)
			}
			// Restore waiting executions from database (for Human/Webhook nodes)
			if err := flowService.RestoreWaitingExecutions(); err != nil {
				logger.Warn("failed to restore waiting executions", zap.Error(err))
			}
			logger.Info("flow service initialized")
		}
	}

	// Start metrics collection (every 5 minutes)
	if metricsService != nil {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := metricsService.Collect(); err != nil {
						logger.Warn("failed to collect metrics", zap.Error(err))
					}
				case <-ctx.Done():
					return
				}
			}
		}()
		// Initial collection
		metricsService.Collect()
	}

	// Create alert service
	var alertSvc *alert.Service
	if auditMgr != nil {
		var err error
		alertSvc, err = alert.NewService(store.DB(), logger)
		if err != nil {
			logger.Fatal("failed to create alert service", zap.Error(err))
		}
	}

	// Create resource service (team mode only)
	var resourceSvc *resource.Service
	if appMode.IsMultiUser() {
		resourceStore, err := resource.NewStore(store.DB(), logger)
		if err != nil {
			logger.Warn("failed to initialize resource store", zap.Error(err))
		} else {
			resourceSvc = resource.NewService(resourceStore, logger)
			logger.Info("resource service initialized")
		}
	}

	// Create permission checker (team mode only)
	var permChecker *permission.Checker
	if appMode.IsMultiUser() {
		permChecker = permission.NewChecker(store.DB(), logger)
		logger.Info("permission checker initialized")
	}

	srv := server.New(cfg, adminToken, appMode, authSvc, userSvc, agentInstance, memMgr, ltmStore, channelMgr, skillMgr, llmClient, cronService, cfgMgr, settingsStore, traceMgr, agentMgr, agentRouter, mcpMgr, agentMsgMgr, queueMgr, metricsService, knowledgeService, flowService, auditMgr, alertSvc, wp, resourceSvc, permChecker, web.FS(), logger)
	go srv.Start()

	// Auto-open browser in solo mode unless --no-browser is set.
	serverAddr := fmt.Sprintf("http://%s:%d", func() string {
		if cfg.Server.Host == "0.0.0.0" {
			return "localhost"
		}
		return cfg.Server.Host
	}(), cfg.Server.Port)
	if (appMode == mode.Solo || *trayFlag) && !*noBrowserFlag {
		go openBrowser(serverAddr, logger)
	}
	logger.Info("gopaw started",
		zap.String("mode", appMode.String()),
		zap.String("url", serverAddr),
		zap.Bool("tray", *trayFlag),
	)

	if *trayFlag {
		// Tray mode: systray.Run() blocks the main thread (required by macOS).
		// The quit channel is still monitored in a goroutine so SIGTERM works too.
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-quit
			cancel()
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer shutdownCancel()
			srv.Shutdown(shutdownCtx)
			os.Exit(0)
		}()
		tray.Run(tray.Config{
			AppURL: serverAddr,
			OnQuit: func() {
				cancel()
				shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer shutdownCancel()
				srv.Shutdown(shutdownCtx)
			},
			Logger: logger,
		})
		return // systray.Run exited
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
	shutdownCtx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	srv.Shutdown(shutdownCtx)

	// Close agent manager and router
	if agentRouter != nil {
		if err := agentRouter.Close(); err != nil {
			logger.Warn("failed to close agent router", zap.Error(err))
		}
	}
	if agentMgr != nil {
		if err := agentMgr.Close(); err != nil {
			logger.Warn("failed to close agent manager", zap.Error(err))
		}
	}

	// Close queue manager
	if queueMgr != nil {
		if err := queueMgr.Close(); err != nil {
			logger.Warn("failed to close queue manager", zap.Error(err))
		}
	}
}

// buildLogger constructs a zap.Logger from config.
// logFile is the workspace log file path (always {workspace}/logs/gopaw.log).
// Output values: "stdout" (default), "file" (logFile only), "both" (stdout + logFile).
func buildLogger(cfg config.LogConfig, logFile string) (*zap.Logger, error) {
	var level zapcore.Level
	_ = level.UnmarshalText([]byte(cfg.Level))

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encCfg)
	}

	var outputs []string
	switch cfg.Output {
	case "file":
		outputs = []string{logFile}
	case "both":
		outputs = []string{"stdout", logFile}
	default: // "stdout" or empty
		outputs = []string{"stdout"}
	}

	// Ensure log directory exists before opening file
	if cfg.Output == "file" || cfg.Output == "both" {
		_ = os.MkdirAll(filepath.Dir(logFile), 0o755)
	}

	sink, _, err := zap.Open(outputs...)
	if err != nil {
		sink, _, _ = zap.Open("stdout")
	}

	return zap.New(zapcore.NewCore(encoder, sink, zap.NewAtomicLevelAt(level)), zap.AddCaller()), nil
}

func generateToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func buildPluginConfigsFromDB(store *settings.Store, logger *zap.Logger) map[string]json.RawMessage {
	plugins := channel.Global().All()
	out := make(map[string]json.RawMessage, len(plugins))
	for _, p := range plugins {
		cfgJSON, _ := store.GetChannelConfig(p.Name())
		if cfgJSON == "" {
			cfgJSON = "{}"
		}
		out[p.Name()] = json.RawMessage(cfgJSON)
	}
	return out
}

// setupHotReload initializes file watchers for hot-reloading configuration.
// Returns cancel functions that should be called on shutdown to clean up resources.
func setupHotReload(agentInstance *agent.ReActAgent, skillMgr *skill.Manager, focusMgr *focus.Manager, wp *workspace.Paths, logger *zap.Logger) []context.CancelFunc {
	var cancels []context.CancelFunc

	// Watch AGENT.md for persona changes
	if agentMDPath := agentInstance.GetAgentMDPath(); agentMDPath != "" {
		cancel, err := config.WatchFile(agentMDPath, func() {
			logger.Info("AGENT.md changed, reloading persona",
				zap.String("path", agentMDPath),
			)
			agentInstance.ReloadPersona()
		}, logger)
		if err != nil {
			logger.Warn("failed to watch AGENT.md", zap.Error(err))
		} else {
			cancels = append(cancels, cancel)
		}
	}

	// Watch skills directory for skill changes
	if wp.SkillsDir != "" {
		cancel, err := config.WatchDir(wp.SkillsDir, func(path string) {
			logger.Info("skills directory changed, reloading skills",
				zap.String("path", path),
			)
			if err := skillMgr.Reload(); err != nil {
				logger.Error("failed to reload skills", zap.Error(err))
			}
		}, logger)
		if err != nil {
			logger.Warn("failed to watch skills directory", zap.Error(err))
		} else {
			cancels = append(cancels, cancel)
		}
	}

	// Watch FOCUS.md for focus changes
	if focusMgr != nil && focusMgr.GetPath() != "" {
		cancel, err := config.WatchFile(focusMgr.GetPath(), func() {
			logger.Info("FOCUS.md changed, reloading focus",
				zap.String("path", focusMgr.GetPath()),
			)
			if err := focusMgr.Load(); err != nil {
				logger.Error("failed to reload focus", zap.Error(err))
			}
		}, logger)
		if err != nil {
			logger.Warn("failed to watch FOCUS.md", zap.Error(err))
		} else {
			cancels = append(cancels, cancel)
		}
	}

	logger.Info("hot-reload watching started",
		zap.String("agent_md", agentInstance.GetAgentMDPath()),
		zap.String("skills_dir", wp.SkillsDir),
		zap.String("focus_file", focusMgr.GetPath()),
	)

	return cancels
}

// registerKnowledgeTools 注册知识库工具
func registerKnowledgeTools(toolReg *tool.Registry, knowledgeService *knowledge.Service, logger *zap.Logger) {
	// 注册知识库搜索工具
	toolReg.Register(&knowledgeSearchTool{service: knowledgeService, logger: logger})
	
	// 注册知识库列表工具
	toolReg.Register(&knowledgeListTool{service: knowledgeService, logger: logger})
}

// knowledgeSearchTool 知识库搜索工具
type knowledgeSearchTool struct {
	service *knowledge.Service
	logger  *zap.Logger
}

func (t *knowledgeSearchTool) Name() string {
	return "knowledge_search"
}

func (t *knowledgeSearchTool) Description() string {
	return "从知识库中搜索相关信息，用于回答用户关于特定领域的问题。"
}

func (t *knowledgeSearchTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"knowledge_base_id": {
				Type:        "string",
				Description: "知识库ID，可通过 knowledge_list 获取",
			},
			"query": {
				Type:        "string",
				Description: "搜索查询，描述你要查找的信息",
			},
			"top_k": {
				Type:        "number",
				Description: "返回结果数量（默认5）",
			},
		},
		Required: []string{"knowledge_base_id", "query"},
	}
}

func (t *knowledgeSearchTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	kbID, _ := args["knowledge_base_id"].(string)
	query, _ := args["query"].(string)
	topK := 5
	if k, ok := args["top_k"].(float64); ok {
		topK = int(k)
	}

	if kbID == "" || query == "" {
		return &plugin.ToolResult{
			LLMOutput: "参数错误：knowledge_base_id 和 query 不能为空",
			IsError:   true,
		}
	}

	resp, err := t.service.Search(ctx, kbID, knowledge.SearchRequest{
		Query:      query,
		TopK:       topK,
		SearchType: "hybrid",
	})
	if err != nil {
		t.logger.Error("knowledge search failed", zap.Error(err))
		return &plugin.ToolResult{
			LLMOutput: fmt.Sprintf("搜索失败: %v", err),
			IsError:   true,
		}
	}

	if len(resp.Results) == 0 {
		return &plugin.ToolResult{
			LLMOutput: "未找到相关信息。",
		}
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("找到 %d 条相关信息：\n\n", len(resp.Results)))

	for i, r := range resp.Results {
		result.WriteString(fmt.Sprintf("[%d] 来自《%s》：\n", i+1, r.DocumentName))
		result.WriteString(r.Content)
		result.WriteString("\n\n")
	}

	return &plugin.ToolResult{
		LLMOutput: result.String(),
	}
}

// knowledgeListTool 知识库列表工具
type knowledgeListTool struct {
	service *knowledge.Service
	logger  *zap.Logger
}

func (t *knowledgeListTool) Name() string {
	return "knowledge_list"
}

func (t *knowledgeListTool) Description() string {
	return "列出所有可用的知识库，获取知识库ID用于搜索。"
}

func (t *knowledgeListTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type:       "object",
		Properties: map[string]plugin.ToolProperty{},
	}
}

func (t *knowledgeListTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	bases, err := t.service.ListKnowledgeBases(ctx)
	if err != nil {
		t.logger.Error("failed to list knowledge bases", zap.Error(err))
		return &plugin.ToolResult{
			LLMOutput: fmt.Sprintf("获取知识库列表失败: %v", err),
			IsError:   true,
		}
	}

	if len(bases) == 0 {
		return &plugin.ToolResult{
			LLMOutput: "当前没有可用的知识库。",
		}
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

	return &plugin.ToolResult{
		LLMOutput: result.String(),
	}
}

// sanitizeCronName converts a cron job name into a safe session ID string.
// Spaces become hyphens; non-alphanumeric characters are dropped; result is lowercased.
func sanitizeCronName(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	result := b.String()
	if result == "" {
		return "cron-unnamed"
	}
	return "cron-" + result
}

// openBrowser opens the default web browser to the given URL after a short delay
// to allow the HTTP server time to start.
func openBrowser(url string, logger *zap.Logger) {
	time.Sleep(800 * time.Millisecond)
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	if err := exec.Command(cmd, args...).Start(); err != nil {
		logger.Warn("failed to open browser", zap.String("url", url), zap.Error(err))
	}
}
