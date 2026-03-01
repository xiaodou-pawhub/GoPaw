// Package server provides the HTTP server, WebSocket handler and middleware for GoPaw.
package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/agent"
	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/internal/config"
	"github.com/gopaw/gopaw/internal/scheduler"
	"github.com/gopaw/gopaw/internal/server/handlers"
	"github.com/gopaw/gopaw/internal/skill"
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
func New(
	cfg *config.Config,
	agentInstance *agent.ReActAgent,
	channelMgr *channel.Manager,
	skillMgr *skill.Manager,
	scheduler *scheduler.Manager,
	cfgMgr *config.Manager,
	logger *zap.Logger,
) *Server {
	if !cfg.App.Debug {
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

	s.registerRoutes(agentInstance, channelMgr, skillMgr, scheduler, cfgMgr)

	s.httpSrv = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      engine,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
	}

	return s
}

// registerRoutes wires all API routes.
func (s *Server) registerRoutes(
	agentInstance *agent.ReActAgent,
	channelMgr *channel.Manager,
	skillMgr *skill.Manager,
	sched *scheduler.Manager,
	cfgMgr *config.Manager,
) {
	// WebSocket endpoint.
	s.engine.GET("/ws", s.wsHandler.Handle)

	api := s.engine.Group("/api")

	// /api/agent
	agentH := handlers.NewAgentHandler(agentInstance, s.logger)
	agentG := api.Group("/agent")
	{
		agentG.POST("/chat", agentH.Chat)
		agentG.GET("/chat/stream", agentH.ChatStream)
		agentG.GET("/sessions", agentH.ListSessions)
	}

	// /api/config
	cfgH := handlers.NewConfigHandler(cfgMgr, s.logger)
	cfgG := api.Group("/config")
	{
		cfgG.GET("", cfgH.Get)
		cfgG.GET("/llm", cfgH.GetLLM)
		cfgG.GET("/agent", cfgH.GetAgent)
	}

	// /api/skills
	skillsH := handlers.NewSkillsHandler(skillMgr, s.logger)
	skillsG := api.Group("/skills")
	{
		skillsG.GET("", skillsH.List)
		skillsG.PUT("/:name/enabled", skillsH.SetEnabled)
	}

	// /api/channels
	channelsH := handlers.NewChannelsHandler(channelMgr, s.logger)
	channelsG := api.Group("/channels")
	{
		channelsG.GET("/health", channelsH.Health)
	}

	// /api/cron
	cronH := handlers.NewCronHandler(sched, s.logger)
	cronG := api.Group("/cron")
	{
		cronG.GET("", cronH.List)
		cronG.POST("", cronH.Create)
		cronG.DELETE("/:id", cronH.Delete)
		cronG.POST("/:id/trigger", cronH.Trigger)
	}

	// /api/system
	sysH := handlers.NewSystemHandler()
	sysG := api.Group("/system")
	{
		sysG.GET("/health", sysH.Health)
		sysG.GET("/version", sysH.Version)
	}

	// Health check at root for load balancers.
	s.engine.GET("/health", sysH.Health)
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
