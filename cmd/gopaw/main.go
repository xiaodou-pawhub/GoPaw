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
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/gopaw/gopaw/internal/agent"
	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/internal/config"
	"github.com/gopaw/gopaw/internal/convlog"
	"github.com/gopaw/gopaw/internal/cron"
	"github.com/gopaw/gopaw/internal/llm"
	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/internal/server"
	"github.com/gopaw/gopaw/internal/settings"
	"github.com/gopaw/gopaw/internal/skill"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/internal/tool/builtin"
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

const appVersion = "0.1.0"

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
	fmt.Fprintln(os.Stderr, `GoPaw - Lightweight AI Assistant Workbench

Usage:
  gopaw <command> [flags]

Commands:
  init      Generate a default config.yaml
  start     Start the GoPaw server
  version   Print version info`)
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
server:
  host: 0.0.0.0
  port: 8088
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
	fs.Parse(os.Args[2:])

	preLogger, _ := zap.NewProduction()
	cfgMgr, _ := config.NewManager(*cfgFile, preLogger)
	cfg := cfgMgr.Get()

	logger, _ := buildLogger(cfg.Log)
	defer logger.Sync()

	watcher := config.NewWatcher(cfgMgr, logger)
	watcher.Start()

	wp, _ := workspace.Resolve(cfg.Workspace.Dir)
	workspace.EnsureDirs(wp)

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

	basePrompt, _ := settings.ReadAgentMD(wp.AgentMDFile)
	if basePrompt == "" { basePrompt = settings.DefaultAgentPrompt }

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
	skillMgr := skill.NewManager(cfg.Skills.Dir, toolReg, logger)
	skillMgr.Load(cfg.Skills.Enabled)

	convLogger, _ := convlog.New(wp.ConvLogFile, logger)

	agentInstance := agent.New(llmClient, toolReg, skillMgr, memMgr, agent.Config{
		DefaultPrompt:  basePrompt,
		AgentMDPath:    wp.AgentMDFile,
		LTMStore:       ltmStore,
		MemoryNotesDir: wp.MemoryNotesDir,
		MaxSteps:       cfg.Agent.MaxSteps,
		Hooks: agent.Hooks{
			PreReasoning: []agent.HookPreReasoning{agent.InjectCurrentTime()},
			PostTool:      []agent.HookPostTool{agent.AutoJournalHook(wp.MemoryNotesDir)},
		},
		ConvLog: convLogger,
	}, logger)

	builtin.SetSubAgentFn(func(ctx context.Context, req *types.Request) (string, error) {
		resp, _ := agentInstance.Process(ctx, req)
		return resp.Content, nil
	})

	channelMgr := channel.NewManager(channel.Global(), mediaStore, logger)
	pluginCfgs := buildPluginConfigsFromDB(settingsStore, logger)

	// --- Cron System Setup ---
	cronService := cron.NewCronService(wp.Root, logger)
	cronService.SetHandler(func(job *cron.CronJob) (string, error) {
		// Create an isolated session for this execution
		sessionID := fmt.Sprintf("cron:%s", job.ID)
		
		logger.Info("cron: executing job", 
			zap.String("job", job.Name), 
			zap.String("task", job.Task),
			zap.String("target", job.TargetID))

		// Construct a request for the agent
		// Note: We use the job's TargetID as both UserID and ChatID to ensure tools
		// like send_to_user route messages back to the correct chat.
		req := &types.Request{
			SessionID: sessionID,
			Channel:   job.Channel,
			ChatID:    job.TargetID, // Critical: this must be the real ChatID for Feishu
			UserID:    "cron",       // Virtual user
			Content:   job.Task,
		}

		// Run the agent. We use a background context with a timeout to prevent stuck jobs.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		resp, err := agentInstance.Process(ctx, req)
		if err != nil {
			logger.Error("cron: job execution failed", zap.Error(err))
			return "", err
		}
		
		// If the agent produced a final textual answer (and didn't just use tools silently),
		// deliver it to the user.
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
				// We don't fail the job execution itself if sending fails, but we log it.
				// Returning the content so it's recorded in history.
				return fmt.Sprintf("Executed. Content: %s. Send Error: %v", resp.Content, err), nil
			}
		}
		
		return resp.Content, nil
	})

	// Inject CronService into built-in tools
	for _, t := range toolReg.All() {
		if ct, ok := t.(interface{ SetCronService(*cron.CronService) }); ok {
			ct.SetCronService(cronService)
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
	
	// Connect approval UI to tool executor
	agentInstance.Executor().SetApprovalUI(coord)

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
				if !ok { return }
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

	srv := server.New(cfg, adminToken, agentInstance, memMgr, ltmStore, channelMgr, skillMgr, cronService, cfgMgr, settingsStore, wp, web.FS(), logger)
	go srv.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	cancel()
	shutdownCtx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	srv.Shutdown(shutdownCtx)
}

func buildLogger(cfg config.LogConfig) (*zap.Logger, error) {
	var level zapcore.Level
	_ = level.UnmarshalText([]byte(cfg.Level))
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	var encoder zapcore.Encoder
	if cfg.Format == "console" { encoder = zapcore.NewConsoleEncoder(encCfg) } else { encoder = zapcore.NewJSONEncoder(encCfg) }
	sink, _, _ := zap.Open("stdout")
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
		if cfgJSON == "" { cfgJSON = "{}" }
		out[p.Name()] = json.RawMessage(cfgJSON)
	}
	return out
}
