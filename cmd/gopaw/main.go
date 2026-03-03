// Command gopaw is the main entry point for the GoPaw AI assistant workbench.
// It provides a CLI with three sub-commands: init, start, and version.
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
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/gopaw/gopaw/internal/agent"
	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/internal/config"
	"github.com/gopaw/gopaw/internal/convlog"
	"github.com/gopaw/gopaw/internal/llm"
	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/internal/scheduler"
	"github.com/gopaw/gopaw/internal/server"
	"github.com/gopaw/gopaw/internal/settings"
	"github.com/gopaw/gopaw/internal/skill"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/internal/workspace"
	"github.com/gopaw/gopaw/pkg/types"
	"github.com/gopaw/gopaw/web"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	// Import built-in channel plugins so their init() functions register them.
	_ "github.com/gopaw/gopaw/internal/platform/console"
	_ "github.com/gopaw/gopaw/internal/platform/dingtalk"
	_ "github.com/gopaw/gopaw/internal/platform/feishu"
	_ "github.com/gopaw/gopaw/internal/platform/webhook"

	// Import built-in tools so their init() functions register them.
	_ "github.com/gopaw/gopaw/internal/tools"
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

// printUsage prints the CLI usage message.
func printUsage() {
	fmt.Fprintln(os.Stderr, `GoPaw - Lightweight AI Assistant Workbench

Usage:
  gopaw <command> [flags]

Commands:
  init      Generate a default config.yaml in the current directory
  start     Start the GoPaw server
  version   Print version information

Run "gopaw <command> --help" for more information about a command.`)
}

// runInit creates a default config.yaml in the current directory.
func runInit() {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	output := fs.String("output", "config.yaml", "Output file path")
	fs.Parse(os.Args[2:]) //nolint:errcheck

	if _, err := os.Stat(*output); err == nil {
		fmt.Fprintf(os.Stderr, "config file %q already exists. Use --output to specify a different path.\n", *output)
		os.Exit(1)
	}

	const defaultConfig = `# GoPaw 启动配置
# 此文件仅包含服务器启动所需的静态配置。
# LLM 提供商、频道密钥、Agent 设定请在启动后通过 Web UI 配置。

workspace:
  dir: ~/.gopaw

app:
  name: "GoPaw"
  language: zh
  timezone: Asia/Shanghai
  debug: false

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
  output: stdout
`
	if err := os.WriteFile(*output, []byte(defaultConfig), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Generated config file: %s\n", *output)
	fmt.Println("Next: edit config.yaml, then run: gopaw start")
	fmt.Println("After startup, visit http://localhost:8088 to configure LLM provider.")
}

// runVersion prints version and runtime information.
func runVersion() {
	fmt.Printf("GoPaw version %s\nGo %s %s/%s\n", appVersion, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

// runStart starts the GoPaw HTTP server.
func runStart() {
	fs := flag.NewFlagSet("start", flag.ExitOnError)
	cfgFile := fs.String("config", "config.yaml", "Path to config file")
	fs.Parse(os.Args[2:]) //nolint:errcheck

	// ---------- Pre-logger (config loading only) ----------
	// 仅用于加载配置阶段，配置读取后会用 config.Log 重建正式 logger
	// Only used during config loading; rebuilt from config.Log settings afterwards
	preLogCfg := zap.NewProductionConfig()
	preLogCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	preLogger, err := preLogCfg.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create pre-logger: %v\n", err)
		os.Exit(1)
	}

	// ---------- Config ----------
	cfgMgr, err := config.NewManager(*cfgFile, preLogger)
	if err != nil {
		preLogger.Fatal("failed to load config", zap.Error(err))
	}
	cfg := cfgMgr.Get()

	// ---------- Logger (config-driven) ----------
	// 读取 config.yaml 中的 log.level / log.format / log.output 重建 logger
	// Rebuild logger from config.yaml log.level / log.format / log.output
	logger, err := buildLogger(cfg.Log)
	if err != nil {
		preLogger.Fatal("failed to build logger from config", zap.Error(err))
	}
	defer logger.Sync() //nolint:errcheck
	preLogger.Sync()    //nolint:errcheck

	// Start hot-reload watcher.
	watcher := config.NewWatcher(cfgMgr, logger)
	watcher.Start()

	// ---------- Workspace ----------
	wp, err := workspace.Resolve(cfg.Workspace.Dir)
	if err != nil {
		logger.Fatal("failed to resolve workspace", zap.Error(err))
	}
	if err := workspace.EnsureDirs(wp); err != nil {
		logger.Fatal("failed to create workspace directories", zap.Error(err))
	}
	logger.Info("workspace initialized", zap.String("root", wp.Root))

	// ---------- Storage ----------
	store, err := memory.NewStore(wp.DBFile)
	if err != nil {
		logger.Fatal("failed to open database", zap.Error(err))
	}
	defer store.Close()

	// ---------- Settings Store ----------
	settingsStore := settings.NewStore(store.DB())

	// ---------- LLM Client ----------
	// LiveClient reads the active provider from the DB on each call,
	// enabling model switching via Web UI without restart.
	llmClient := llm.NewLiveClient(func() (string, string, string, int, int, error) {
		p, err := settingsStore.GetActiveProvider()
		if err != nil {
			return "", "", "", 0, 0, fmt.Errorf("读取 LLM 配置失败: %w", err)
		}
		if p == nil {
			return "", "", "", 0, 0, fmt.Errorf("LLM 未配置，请在 Web UI → 设置 → LLM 提供商 中添加配置")
		}
		return p.BaseURL, p.APIKey, p.Model, p.MaxTokens, p.TimeoutSec, nil
	}, logger)

	if settingsStore.IsSetupRequired() {
		logger.Warn("LLM provider not configured — visit the Web UI to set it up before chatting")
	}

	// ---------- Agent System Prompt ----------
	// Read from workspace/agent/AGENT.md.
	basePrompt, err := settings.ReadAgentMD(wp.AgentMDFile)
	if err != nil {
		logger.Warn("failed to read AGENT.md, using default prompt", zap.Error(err))
		basePrompt = settings.DefaultAgentPrompt
	}

	// ---------- Memory ----------
	memMgr := memory.NewManager(store, llmClient,
		cfg.Agent.Memory.ContextLimit,
		cfg.Agent.Memory.HistoryLimit,
		logger,
	)

	// ---------- Skills ----------
	toolReg := tool.Global()
	skillMgr := skill.NewManager(cfg.Skills.Dir, toolReg, logger)
	if err := skillMgr.Load(cfg.Skills.Enabled); err != nil {
		logger.Warn("skill load error (continuing)", zap.Error(err))
	}

	// ---------- Conversation Log ----------
	// Pass logger so convlog also prints simplified events to the console.
	convLogger, err := convlog.New(wp.ConvLogFile, logger)
	if err != nil {
		logger.Warn("failed to create conversation logger, continuing without", zap.Error(err))
	}

	// ---------- Agent ----------
	agentInstance := agent.New(
		llmClient,
		toolReg,
		skillMgr,
		memMgr,
		agent.Config{
			DefaultPrompt: basePrompt,
			AgentMDPath:   wp.AgentMDFile,
			MaxSteps:      cfg.Agent.MaxSteps,
			ConvLog:       convLogger,
		},
		logger,
	)

	// ---------- Channels ----------
	// All channel plugins are imported via _ import in main.go.
	// They register themselves via init() and are started automatically.
	channelMgr := channel.NewManager(channel.Global(), logger)

	// Load channel configs from DB (for plugins that need credentials).
	pluginCfgs := buildPluginConfigsFromDB(settingsStore, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := channelMgr.Start(ctx, pluginCfgs); err != nil {
		logger.Fatal("failed to start channels", zap.Error(err))
	}
	defer channelMgr.Stop()

	// ---------- Message dispatch loop ----------
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
					req := &types.Request{
						SessionID: m.SessionID,
						UserID:    m.UserID,
						Channel:   m.Channel,
						Content:   m.Content,
						MsgType:   m.MsgType,
						Files:     m.Files,
						Metadata:  m.Metadata,
					}
					resp, err := agentInstance.Process(ctx, req)
					if err != nil {
						logger.Error("agent processing error", zap.Error(err))
						return
					}
					reply := &types.Message{
						SessionID: m.SessionID,
						UserID:    m.UserID,
						Channel:   m.Channel,
						Content:   resp.Content,
						MsgType:   resp.MsgType,
						ReplyTo:   m.ID,
					}
					if sendErr := channelMgr.Send(reply); sendErr != nil {
						logger.Error("channel send error", zap.Error(sendErr))
					}
				}(msg)
			}
		}
	}()

	// ---------- Scheduler ----------
	jobStore := scheduler.NewJobStore(store.DB())
	schedMgr := scheduler.NewManager(
		jobStore,
		func(sCtx context.Context, req *types.Request) (*types.Response, error) {
			return agentInstance.Process(sCtx, req)
		},
		channelMgr.Send,
		logger,
	)
	if err := schedMgr.Start(ctx); err != nil {
		logger.Warn("scheduler start error", zap.Error(err))
	}
	defer schedMgr.Stop()

	// ---------- Admin Token ----------
	// 使用 config.yaml 中的 admin_token；若未配置则每次启动自动生成随机 token
	// Use admin_token from config.yaml; auto-generate a random token if not set
	adminToken := cfg.App.AdminToken
	if adminToken == "" {
		adminToken = generateToken()
		logger.Info("⚡ Web UI access token (set app.admin_token in config.yaml to fix it)",
			zap.String("token", adminToken),
		)
	}

	// ---------- HTTP Server ----------
	srv := server.New(cfg, adminToken, agentInstance, memMgr, channelMgr, skillMgr, schedMgr, cfgMgr, settingsStore, wp, web.FS(), logger)

	go func() {
		if err := srv.Start(); err != nil {
			logger.Error("server error", zap.Error(err))
		}
	}()

	logger.Info("GoPaw started",
		zap.String("version", appVersion),
		zap.String("addr", fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)),
	)
	if settingsStore.IsSetupRequired() {
		logger.Info("Setup required — open Web UI to configure LLM provider",
			zap.String("url", fmt.Sprintf("http://localhost:%d/api/settings/setup-status", cfg.Server.Port)),
		)
	}

	// ---------- Graceful Shutdown ----------
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", zap.Error(err))
	}
	logger.Info("GoPaw stopped")
}

// buildLogger constructs a zap.Logger from the log configuration.
// Supports multiple outputs: "stdout", "stderr", a file path, or comma-separated outputs (e.g., "stdout,./logs/app.log")
// 必须在 config 加载完成后调用，以确保 log.level / log.format / log.output 生效
func buildLogger(cfg config.LogConfig) (*zap.Logger, error) {
	// Resolve log level
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	// Resolve encoder: "console" for human-readable, anything else → json
	var encoder zapcore.Encoder
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	if cfg.Format == "console" {
		encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encCfg)
	}

	// Resolve output: "stdout", "stderr", a file path, or comma-separated outputs
	outputs := strings.Split(cfg.Output, ",")
	var cores []zapcore.Core

	for _, output := range outputs {
		output = strings.TrimSpace(output)
		if output == "" {
			output = "stdout"
		}

		sink, closeOut, err := zap.Open(output)
		if err != nil {
			// Fall back to stderr if the configured output can't be opened
			sink, _, _ = zap.Open("stderr")
			closeOut = func() {}
		}
		_ = closeOut

		cores = append(cores, zapcore.NewCore(encoder, sink, zap.NewAtomicLevelAt(level)))
	}

	core := zapcore.NewTee(cores...)
	return zap.New(core, zap.AddCaller()), nil
}

// generateToken creates a cryptographically random 32-hex-character token.
func generateToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate random token: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// buildPluginConfigsFromDB reads configs for all registered channel plugins.
// Channels not yet configured in the DB receive an empty JSON object "{}".
func buildPluginConfigsFromDB(store *settings.Store, logger *zap.Logger) map[string]json.RawMessage {
	plugins := channel.Global().All()
	out := make(map[string]json.RawMessage, len(plugins))
	for _, p := range plugins {
		name := p.Name()
		cfgJSON, err := store.GetChannelConfig(name)
		if err != nil {
			cfgJSON = "{}"
		}
		out[name] = json.RawMessage(cfgJSON)
	}
	return out
}
