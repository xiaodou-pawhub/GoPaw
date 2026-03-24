// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

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
	"github.com/gopaw/gopaw/internal/agent/message"
	"github.com/gopaw/gopaw/internal/alert"
	"github.com/gopaw/gopaw/internal/audit"
	"github.com/gopaw/gopaw/internal/auth"
	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/internal/config"
	"github.com/gopaw/gopaw/internal/cron"
	"github.com/gopaw/gopaw/internal/flow"
	"github.com/gopaw/gopaw/internal/knowledge"
	"github.com/gopaw/gopaw/internal/llm"
	"github.com/gopaw/gopaw/internal/mcp"
	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/internal/metrics"
	"github.com/gopaw/gopaw/internal/mode"
	"github.com/gopaw/gopaw/internal/queue"
	"github.com/gopaw/gopaw/internal/server/handlers"
	"github.com/gopaw/gopaw/internal/settings"
	"github.com/gopaw/gopaw/internal/skill"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/internal/trace"
	"github.com/gopaw/gopaw/internal/user"
	"github.com/gopaw/gopaw/internal/workspace"
	"go.uber.org/zap"
)

// Server bundles the HTTP server and all route handlers.
type Server struct {
	cfg       *config.Config
	mode      mode.Mode
	engine    *gin.Engine
	httpSrv   *http.Server
	wsHandler *WSHandler
	logger    *zap.Logger
}

// New creates and configures the HTTP server without starting it.
// adminToken is the resolved access token (from config or auto-generated).
// m is the deployment mode controlling authentication behaviour.
// authSvc and userSvc are only used in team mode (pass nil for solo).
// staticFS is the embedded Vue frontend filesystem (pass nil to disable static serving).
func New(
	cfg *config.Config,
	adminToken string,
	m mode.Mode,
	authSvc *auth.Service,
	userSvc *user.Service,
	agentInstance *agent.ReActAgent,
	memMgr *memory.Manager,
	ltmStore *memory.LTMStore,
	channelMgr *channel.Manager,
	skillMgr *skill.Manager,
	llmClient llm.Client,
	cronService *cron.CronService,
	cfgMgr *config.Manager,
	settingsStore *settings.Store,
	traceMgr *trace.Manager,
	agentMgr *agent.Manager,
	agentRouter *agent.Router,
	mcpMgr *mcp.Manager,
	agentMsgMgr *message.Manager,
	queueMgr *queue.Manager,
	metricsService *metrics.Service,
	knowledgeService *knowledge.Service,
	flowService *flow.Service,
	auditMgr *audit.Manager,
	alertSvc *alert.Service,
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
		mode:      m,
		engine:    engine,
		wsHandler: NewWSHandler(agentInstance, agentRouter, logger),
		logger:    logger,
	}

	s.registerRoutes(adminToken, m, authSvc, userSvc, agentInstance, memMgr, ltmStore, channelMgr, skillMgr, llmClient, cronService, cfgMgr, settingsStore, traceMgr, agentMgr, agentRouter, mcpMgr, agentMsgMgr, queueMgr, metricsService, knowledgeService, flowService, auditMgr, alertSvc, wp, staticFS)

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
	m mode.Mode,
	authSvc *auth.Service,
	userSvc *user.Service,
	agentInstance *agent.ReActAgent,
	memMgr *memory.Manager,
	ltmStore *memory.LTMStore,
	channelMgr *channel.Manager,
	skillMgr *skill.Manager,
	llmClient llm.Client,
	cronService *cron.CronService,
	cfgMgr *config.Manager,
	settingsStore *settings.Store,
	traceMgr *trace.Manager,
	agentMgr *agent.Manager,
	agentRouter *agent.Router,
	mcpMgr *mcp.Manager,
	agentMsgMgr *message.Manager,
	queueMgr *queue.Manager,
	metricsService *metrics.Service,
	knowledgeService *knowledge.Service,
	flowService *flow.Service,
	auditMgr *audit.Manager,
	alertSvc *alert.Service,
	wp *workspace.Paths,
	staticFS fs.FS,
) {
	// WebSocket endpoint — protected by WebAuth (cookie must be valid).
	s.engine.GET("/ws", WebAuth(adminToken, m, authSvc), s.wsHandler.Handle)

	// Approval WebSocket endpoint — for tool execution approval in Web Console.
	approvalHandler := NewApprovalWSHandler(tool.GlobalApprovalStore, s.logger)
	s.engine.GET("/ws/approval", WebAuth(adminToken, m, authSvc), approvalHandler.Handle)
	_ = approvalHandler // avoid unused warning for now

	// /api/mode — public, returns current deployment mode and auth requirements.
	s.engine.GET("/api/mode", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"mode":         s.mode.String(),
			"require_auth": s.mode.RequireAuth(),
			"multi_user":   s.mode.IsMultiUser(),
		})
	})

	// /api/auth — public, no auth required (except /status and /me)
	authH := handlers.NewAuthHandler(adminToken, m, authSvc, userSvc)
	authG := s.engine.Group("/api/auth")
	{
		authG.POST("/login", authH.Login)
		authG.POST("/logout", authH.Logout)
		// /api/auth/status and /me are behind WebAuth
		authG.GET("/status", WebAuth(adminToken, m, authSvc), authH.Status)
		authG.GET("/me", WebAuth(adminToken, m, authSvc), authH.Me)
	}

	api := s.engine.Group("/api", WebAuth(adminToken, m, authSvc))

	// /api/agent
	agentH := handlers.NewAgentHandler(agentInstance, agentRouter, memMgr, s.logger)
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

	// /api/settings — runtime settings (LLM providers, channel secrets)
	settingsH := handlers.NewSettingsHandler(settingsStore, channelMgr, s.logger)
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
	}

	// /api/workspace — agent files (PERSONA.md, MEMORY.md)
	workspaceH := handlers.NewWorkspaceHandler(wp, s.logger)
	workspaceG := api.Group("/workspace")
	{
		workspaceG.GET("/persona", workspaceH.GetPersona)
		workspaceG.PUT("/persona", workspaceH.PutPersona)
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
	skillsH := handlers.NewSkillsHandler(skillMgr, wp.SkillsDir, llmClient, s.logger)
	skillsG := api.Group("/skills")
	{
		skillsG.GET("", skillsH.List)
		skillsG.GET("/market", skillsH.MarketList)
		skillsG.POST("/reload", skillsH.Reload)
		skillsG.POST("/install", skillsH.Install)
		skillsG.POST("/import", skillsH.ImportZip)
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

	// /api/traces — execution traces
	traceH := handlers.NewTraceHandler(traceMgr, s.logger)
	traceG := api.Group("/traces")
	{
		traceG.GET("", traceH.List)
		traceG.GET("/stats", traceH.Stats)
		traceG.GET("/:id", traceH.Get)
	}

	// /api/agents — multi-agent management
	if agentMgr != nil {
		agentsH := handlers.NewAgentsHandler(agentMgr, s.logger)
		agentsG := api.Group("/agents")
		{
			agentsG.GET("", agentsH.List)
			agentsG.GET("/default", agentsH.GetDefault)
			agentsG.POST("", agentsH.Create)
			agentsG.GET("/:id", agentsH.Get)
			agentsG.PUT("/:id", agentsH.Update)
			agentsG.DELETE("/:id", agentsH.Delete)
			agentsG.POST("/:id/default", agentsH.SetDefault)
			agentsG.GET("/:id/config", agentsH.GetConfig)
			agentsG.PUT("/:id/config", agentsH.UpdateConfig)
			// 版本管理
			agentsG.GET("/:id/versions", agentsH.ListVersions)
			agentsG.GET("/:id/versions/stats", agentsH.GetVersionStats)
			agentsG.POST("/:id/versions", agentsH.CreateVersion)
			agentsG.GET("/:id/versions/:version", agentsH.GetVersion)
			agentsG.POST("/:id/versions/:version/rollback", agentsH.RollbackVersion)
			agentsG.DELETE("/:id/versions/:version", agentsH.DeleteVersion)
			// 性能分析
			agentsG.GET("/stats", agentsH.GetAllAgentsStats)
			agentsG.GET("/:id/stats", agentsH.GetAgentStats)
			agentsG.GET("/:id/stats/daily", agentsH.GetAgentDailyStats)
			agentsG.GET("/:id/stats/errors", agentsH.GetAgentErrorStats)
		}
	}

	// /api/mcp — MCP server management
	if mcpMgr != nil {
		mcpH := handlers.NewMCPHandler(mcpMgr, s.logger)
		mcpG := api.Group("/mcp")
		{
			mcpG.GET("/servers", mcpH.List)
			mcpG.POST("/servers", mcpH.Create)
			mcpG.GET("/servers/:id", mcpH.Get)
			mcpG.PUT("/servers/:id", mcpH.Update)
			mcpG.DELETE("/servers/:id", mcpH.Delete)
			mcpG.POST("/servers/:id/active", mcpH.SetActive)
			mcpG.GET("/servers/:id/tools", mcpH.GetTools)
			mcpG.GET("/tools", mcpH.GetAllTools)
			mcpG.POST("/test", mcpH.TestServer)
		}
	}

	// /api/agent-messages — agent-to-agent messaging
	if agentMsgMgr != nil {
		agentMsgH := handlers.NewAgentMessageHandler(agentMsgMgr, s.logger)
		agentMsgG := api.Group("/agent-messages")
		{
			agentMsgG.POST("", agentMsgH.SendMessage)
			agentMsgG.POST("/task", agentMsgH.SendTask)
			agentMsgG.POST("/response", agentMsgH.SendResponse)
			agentMsgG.POST("/notify", agentMsgH.SendNotify)
			agentMsgG.POST("/query", agentMsgH.SendQuery)
			agentMsgG.GET("/:id", agentMsgH.GetMessage)
			agentMsgG.GET("/agent/:agent_id", agentMsgH.ListMessages)
			agentMsgG.GET("/agent/:agent_id/sent", agentMsgH.ListSentMessages)
			agentMsgG.GET("/agent/:agent_id/pending", agentMsgH.GetPendingMessages)
			agentMsgG.GET("/agent/:agent_id/stats", agentMsgH.GetStats)
			agentMsgG.GET("/agent/:agent_id/conversations", agentMsgH.ListConversations)
			agentMsgG.GET("/conversation/:parent_id", agentMsgH.ListConversation)
			agentMsgG.PUT("/:id/status", agentMsgH.UpdateStatus)
		}
	}

	// Health check at root — public, for load balancers / uptime monitors.
	// /api/system/health is already registered inside the api group above (behind WebAuth).
	s.engine.GET("/health", sysH.Health)

	// /api/queues — message queue management
	if queueMgr != nil {
		queueH := handlers.NewQueueHandler(queueMgr, s.logger)
		queuesG := api.Group("/queues")
		{
			queuesG.GET("", queueH.ListQueues)
			queuesG.GET("/:name/stats", queueH.GetQueueStats)
			queuesG.GET("/:name/messages", queueH.ListMessages)
			queuesG.POST("/:name/messages", queueH.PublishMessage)
			queuesG.POST("/:name/pause", queueH.PauseQueue)
			queuesG.POST("/:name/resume", queueH.ResumeQueue)
			queuesG.POST("/:name/cleanup", queueH.CleanupQueue)
		}
		messagesG := api.Group("/messages")
		{
			messagesG.GET("/:id", queueH.GetMessage)
			messagesG.POST("/:id/retry", queueH.RetryMessage)
			messagesG.DELETE("/:id", queueH.DeleteMessage)
		}
	}

	// /api/metrics — metrics dashboard
	if metricsService != nil {
		metricsH := handlers.NewMetricsHandler(metricsService, s.logger)
		metricsG := api.Group("/metrics")
		{
			metricsG.GET("/dashboard", metricsH.GetDashboard)
			metricsG.GET("/activity", metricsH.GetRecentActivity)
			metricsG.POST("/collect", metricsH.Collect)
		}
	}

	// /api/knowledge — knowledge base management
	if knowledgeService != nil {
		knowledgeH := handlers.NewKnowledgeHandler(knowledgeService)
		knowledgeH.RegisterRoutes(api)
	}

	// /api/alert — alert management
	if alertSvc != nil {
		alertH := handlers.NewAlertHandler(alertSvc, s.logger)
		alertH.RegisterRoutes(api)
	}

	// /api/flows — unified flow management
	if flowService != nil {
		flowH := handlers.NewFlowHandler(flowService, s.logger)
		flowH.RegisterRoutes(api)
	}

	// /api/users — user management (team mode only, admin access)
	if userSvc != nil && m.IsMultiUser() {
		usersH := handlers.NewUsersHandler(userSvc)
		usersG := api.Group("/users")
		{
			usersG.GET("", usersH.List)
			usersG.POST("", usersH.Create)
			usersG.DELETE("/:id", usersH.Delete)
			usersG.PUT("/:id/active", usersH.SetActive)
			usersG.PUT("/:id/role", usersH.SetRole)
			usersG.PUT("/:id/password", usersH.ResetPassword)
		}
	}

	// /api/audit — audit log management (team mode only)
	if auditMgr != nil && m.IsMultiUser() {
		auditH := handlers.NewAuditHandler(auditMgr, s.logger)
		auditG := api.Group("/audit")
		{
			auditG.GET("/logs", auditH.ListAuditLogs)
			auditG.GET("/logs/recent", auditH.GetRecentAuditLogs)
			auditG.GET("/logs/:id", auditH.GetAuditLog)
			auditG.GET("/stats", auditH.GetAuditStats)
			auditG.POST("/export", auditH.ExportAuditLogs)
			auditG.POST("/cleanup", auditH.CleanupAuditLogs)
		}
	}

	// /api/resource-packages — resource package management (team mode only)
	if m.IsMultiUser() {
		// Resource handler will be initialized when resource package is implemented
		// resourceH := handlers.NewResourceHandler(resourceSvc, s.logger)
		// resourceG := api.Group("/resource-packages")
		// {
		// 	resourceG.POST("", resourceH.CreatePackage)
		// 	resourceG.GET("", resourceH.ListPackages)
		// 	resourceG.GET("/:id", resourceH.GetPackage)
		// 	resourceG.PUT("/:id", resourceH.UpdatePackage)
		// 	resourceG.DELETE("/:id", resourceH.DeletePackage)
		// 	resourceG.POST("/:id/items", resourceH.AddItem)
		// 	resourceG.DELETE("/:id/items/:type/:resource_id", resourceH.RemoveItem)
		// 	resourceG.POST("/:id/grant", resourceH.GrantToUser)
		// 	resourceG.DELETE("/:id/grant/:user_id", resourceH.RevokeGrant)
		// }
		//
		// // Agent permissions
		// agentPermG := api.Group("/agents")
		// {
		// 	agentPermG.POST("/:id/permissions", resourceH.SetAgentPermission)
		// 	agentPermG.PUT("/:id/visibility", resourceH.SetAgentVisibility)
		// 	agentPermG.GET("/:id/permission", resourceH.CheckAgentPermission)
		// }
		//
		// // User packages
		// userPackageG := api.Group("/users")
		// {
		// 	userPackageG.GET("/:id/packages", resourceH.GetUserPackages)
		// }
	}

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
