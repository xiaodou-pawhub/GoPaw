// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/metrics"
	"go.uber.org/zap"
)

// MetricsHandler handles metrics-related HTTP requests.
type MetricsHandler struct {
	service *metrics.Service
	logger  *zap.Logger
}

// NewMetricsHandler creates a new metrics handler.
func NewMetricsHandler(service *metrics.Service, logger *zap.Logger) *MetricsHandler {
	return &MetricsHandler{
		service: service,
		logger:  logger.Named("metrics_handler"),
	}
}

// GetDashboard returns dashboard data.
func (h *MetricsHandler) GetDashboard(c *gin.Context) {
	data, err := h.service.GetDashboardData()
	if err != nil {
		h.logger.Error("failed to get dashboard data", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// GetRecentActivity returns recent activity.
func (h *MetricsHandler) GetRecentActivity(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	activities, err := h.service.GetRecentActivity(limit)
	if err != nil {
		h.logger.Error("failed to get recent activity", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, activities)
}

// Collect triggers metrics collection.
func (h *MetricsHandler) Collect(c *gin.Context) {
	if err := h.service.Collect(); err != nil {
		h.logger.Error("failed to collect metrics", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "metrics collected"})
}
