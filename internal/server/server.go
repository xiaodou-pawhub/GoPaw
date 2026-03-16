// Package server provides the HTTP server, WebSocket handler and middleware for GoPaw.
package server

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/agent"
	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/internal/config"
	"github.com/gopaw/gopaw/internal/cron"
	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/internal/server/handlers"
	"github.com/gopaw/gopaw/internal/settings"
	"github.com/gopaw/gopaw/internal/skill"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/internal/workspace"
	"go.uber.org/zap"
)

// Server bundles the HTTP server and all route handlers.
type Server struct {
	cfg       *config.Config
	engine    *gin.Engine
	httpSrv   *http.Server
	wsHandler *WSHandler
	logger    *zap.Logger
}

// New creates and configures the HTTP server without starting it.
// adminToken is the resolved access token (from config or auto-generated).
// staticFS is the embedded Vue frontend filesystem (pass nil to disable static serving).
func New(
	cfg *config.Config,
	adminToken string,
	agentInstance *agent.ReActAgent,
	memMgr *memory.Manager,
	ltmStore *memory.LTMStore,
	channelMgr *channel.Manager,
	skillMgr *skill.Manager,
	cronService *cron.CronService,
	cfgMgr *config.Manager,
	settingsStore *settings.Store,
	wp *workspace.Paths,
	staticFS fs.FS,
	logger *zap.Logger,
) *Server {
	if cfg.Log.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(Recovery(logger), ZapLogger(logger), CORS())

	s := &Server{
		cfg:       cfg,
		engine:    engine,
		wsHandler: NewWSHandler(agentInstance, logger),
		logger:    logger,
	}

	s.registerRoutes(adminToken, agentInstance, memMgr, ltmStore, channelMgr, skillMgr, cronService, cfgMgr, settingsStore, wp, staticFS)

	s.httpSrv = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	return s
}

// registerRoutes wires all API routes and the SPA static file handler.
func (s *Server) registerRoutes(
	adminToken string,
	agentInstance *agent.ReActAgent,
	memMgr *memory.Manager,
	ltmStore *memory.LTMStore,
	channelMgr *channel.Manager,
	skillMgr *skill.Manager,
	cronService *cron.CronService,
	cfgMgr *config.Manager,
	settingsStore *settings.Store,
	wp *workspace.Paths,
	staticFS fs.FS,
) {
	// WebSocket endpoint — protected by WebAuth (cookie must be valid).
	s.engine.GET("/ws", WebAuth(adminToken), s.wsHandler.Handle)

	// Approval WebSocket endpoint — for tool execution approval in Web Console.
	approvalHandler := NewApprovalWSHandler(tool.GlobalApprovalStore, s.logger)
	s.engine.GET("/ws/approval", WebAuth(adminToken), approvalHandler.Handle)
	_ = approvalHandler // avoid unused warning for now

	// /api/auth — public, no auth required
	authH := handlers.NewAuthHandler(adminToken)
	authG := s.engine.Group("/api/auth")
	{
		authG.POST("/login", authH.Login)
		authG.POST("/logout", authH.Logout)
		// /api/auth/status is behind WebAuth: 200 = logged in, 401 = not logged in
		authG.GET("/status", WebAuth(adminToken), authH.Status)
	}

	api := s.engine.Group("/api", WebAuth(adminToken))

	// /api/agent
	agentH := handlers.NewAgentHandler(agentInstance, memMgr, s.logger)
	uploadH := handlers.NewUploadHandler(s.logger)
	agentG := api.Group("/agent")
	{
		agentG.POST("/chat", agentH.Chat)
		agentG.GET("/chat/stream", agentH.ChatStream)
		agentG.POST("/chat/stream", agentH.ChatStreamPost) // 中文：POST 流式对话，支持大内容 / English: POST streaming, supports large content
		agentG.GET("/sessions", agentH.ListSessions)
		agentG.GET("/sessions/:id/messages", agentH.GetSessionMessages)
		agentG.DELETE("/sessions/:id", agentH.DeleteSession)
		agentG.PUT("/sessions/:id/name", agentH.UpdateSessionName)
		agentG.GET("/sessions/:id/stats", agentH.GetSessionStats)
		agentG.POST("/upload", uploadH.Upload)
	}

	// /api/config — static startup configuration (read-only)
	cfgH := handlers.NewConfigHandler(cfgMgr, s.logger)
	api.GET("/config", cfgH.Get)

	// /api/settings — runtime settings (LLM providers, channel secrets, agent persona)
	settingsH := handlers.NewSettingsHandler(settingsStore, wp.AgentMDFile, channelMgr, s.logger)
	settingsG := api.Group("/settings")
	{
		settingsG.GET("/setup-status", settingsH.SetupStatus)
		settingsG.GET("/providers", settingsH.ListProviders)
		settingsG.GET("/builtin-providers", settingsH.ListBuiltinProviders)
		settingsG.GET("/providers/health", settingsH.GetProvidersHealth)
		settingsG.POST("/providers", settingsH.SaveProvider)
		settingsG.PUT("/providers/:id/active", settingsH.SetActiveProvider)
		settingsG.DELETE("/providers/:id", settingsH.DeleteProvider)
		
		// New endpoints for priority-based model management
		settingsG.POST("/providers/:id/toggle", settingsH.ToggleProvider)
		settingsG.POST("/providers/reorder", settingsH.ReorderProviders)
		settingsG.GET("/providers/capable/:capability", settingsH.GetCapableProviders)
		
		settingsG.GET("/channels/:name", settingsH.GetChannelConfig)
		settingsG.PUT("/channels/:name", settingsH.SetChannelConfig)
		settingsG.GET("/agent", settingsH.GetAgentMD)
		settingsG.PUT("/agent", settingsH.SetAgentMD)
	}

	// /api/workspace — agent files (AGENT.md, PERSONA.md, CONTEXT.md, MEMORY.md)
	workspaceH := handlers.NewWorkspaceHandler(wp, s.logger)
	workspaceG := api.Group("/workspace")
	{
		workspaceG.GET("/agent", workspaceH.GetAgent)
		workspaceG.PUT("/agent", workspaceH.PutAgent)
		workspaceG.GET("/persona", workspaceH.GetPersona)
		workspaceG.PUT("/persona", workspaceH.PutPersona)
		workspaceG.GET("/context", workspaceH.GetContext)
		workspaceG.PUT("/context", workspaceH.PutContext)
		workspaceG.GET("/memory", workspaceH.GetMemory)
		workspaceG.PUT("/memory", workspaceH.PutMemory)
	}

	// /api/memories — structured long-term memory CRUD
	memH := handlers.NewMemoryHandler(ltmStore, s.logger)
	memG := api.Group("/memories")
	{
		memG.GET("", memH.List)
		memG.POST("", memH.Create)
		memG.PUT("/:id", memH.Update)
		memG.DELETE("/:id", memH.Delete)
		memG.GET("/stats", memH.Stats)
		memG.POST("/import-md", memH.ImportMD)
	}

	// /api/memory-files — MD file management (MEMORY.md, notes, archives)
	memFilesH := handlers.NewMemoryFilesHandler(wp, s.logger)
	memFilesG := api.Group("/memory-files")
	{
		memFilesG.GET("/memory", memFilesH.GetMemoryMD)
		memFilesG.PUT("/memory", memFilesH.PutMemoryMD)
		memFilesG.GET("/notes", memFilesH.ListNotes)
		memFilesG.GET("/notes/:date", memFilesH.GetNote)
		memFilesG.PUT("/notes/:date", memFilesH.PutNote)
		memFilesG.POST("/notes/:date/append", memFilesH.AppendNote)
		memFilesG.DELETE("/notes/:date", memFilesH.DeleteNote)
		memFilesG.GET("/archives", memFilesH.ListArchives)
		memFilesG.GET("/archives/:name", memFilesH.GetArchive)
	}

	// /api/skills
	skillsH := handlers.NewSkillsHandler(skillMgr, wp.SkillsDir, s.logger)
	skillsG := api.Group("/skills")
	{
		skillsG.GET("", skillsH.List)
		skillsG.POST("/reload", skillsH.Reload)
		skillsG.POST("/install", skillsH.Install)
		skillsG.PUT("/:name/enabled", skillsH.SetEnabled)
	}

	// /api/channels
	channelsH := handlers.NewChannelsHandler(channelMgr, s.logger)
	channelsG := api.Group("/channels")
	{
		channelsG.GET("/health", channelsH.Health)
		channelsG.POST("/:name/test", channelsH.Test)
	}

	// /api/cron
	cronH := handlers.NewCronHandler(cronService, s.logger)
	cronG := api.Group("/cron")
	{
		cronG.GET("", cronH.List)
		cronG.POST("", cronH.Create)
		cronG.PUT("/:id", cronH.Update)
		cronG.DELETE("/:id", cronH.Delete)
		cronG.POST("/:id/trigger", cronH.Trigger)
		cronG.GET("/:id/runs", cronH.ListRuns)
	}

	// /api/system — all behind WebAuth (already inherited from api group)
	sysH := handlers.NewSystemHandler(s.cfg, wp.LogFile)
	sysG := api.Group("/system")
	{
		sysG.GET("/health", sysH.Health)
		sysG.GET("/version", sysH.Version)
		sysG.GET("/logs", sysH.ListLogs) // WebAuth already guards the group
	}

	// Health check at root — public, for load balancers / uptime monitors.
	// /api/system/health is already registered inside the api group above (behind WebAuth).
	s.engine.GET("/health", sysH.Health)

	// DingTalk channel routes (no /api prefix).
	dingTalkH := handlers.NewDingTalkHandler(channelMgr)
	s.engine.POST("/dingtalk/event", dingTalkH.Event)

	// SPA static file serving — must be registered last so API routes take priority.
	if staticFS != nil {
		s.engine.NoRoute(spaHandler(staticFS, s.logger))
	}
}

// spaHandler returns a Gin handler that serves the embedded Vue SPA.
// For requests to known static assets (JS/CSS/images), it serves the file directly.
// For all other paths (Vue Router client-side routes), it falls back to index.html.
func spaHandler(staticFS fs.FS, logger *zap.Logger) gin.HandlerFunc {
	httpFS := http.FS(staticFS)
	fileServer := http.FileServer(httpFS)
	return func(c *gin.Context) {
		upath := strings.TrimPrefix(c.Request.URL.Path, "/")
		if upath == "" {
			upath = "index.html"
		}
		// Check whether the file actually exists in the embedded FS.
		f, err := staticFS.Open(upath)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		// Unknown path — serve index.html so Vue Router can handle it.
		c.FileFromFS("index.html", httpFS)
	}
}

// Start begins accepting HTTP connections.
func (s *Server) Start() error {
	s.logger.Info("http server starting", zap.String("addr", s.httpSrv.Addr))
	if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server: listen: %w", err)
	}
	return nil
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("http server shutting down")
	return s.httpSrv.Shutdown(ctx)
}
