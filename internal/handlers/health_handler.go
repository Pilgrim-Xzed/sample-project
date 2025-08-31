package handlers

import (
	"net/http"
	"time"

	"budget-management/internal/database"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db          *gorm.DB
	redisClient *redis.Client
	logger      *logrus.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(db *gorm.DB, redisClient *redis.Client, logger *logrus.Logger) *HealthHandler {
	return &HealthHandler{
		db:          db,
		redisClient: redisClient,
		logger:      logger,
	}
}

// Health returns basic health status
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC(),
		"service":   "budget-management",
		"version":   "1.0.0",
	})
}

// Ready returns readiness status including database and Redis connectivity
func (h *HealthHandler) Ready(c *gin.Context) {
	checks := make(map[string]interface{})
	allHealthy := true

	// Check database connectivity
	if err := database.HealthCheck(h.db); err != nil {
		checks["database"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		allHealthy = false
		h.logger.Errorf("Database health check failed: %v", err)
	} else {
		checks["database"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	// Check Redis connectivity
	if err := database.RedisHealthCheck(h.redisClient); err != nil {
		checks["redis"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		allHealthy = false
		h.logger.Errorf("Redis health check failed: %v", err)
	} else {
		checks["redis"] = map[string]interface{}{
			"status": "healthy",
		}
	}

	status := "ready"
	statusCode := http.StatusOK
	
	if !allHealthy {
		status = "not ready"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, gin.H{
		"status":    status,
		"timestamp": time.Now().UTC(),
		"checks":    checks,
	})
}

// Live returns liveness status (basic application health)
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now().UTC(),
	})
}

// Metrics returns basic application metrics
func (h *HealthHandler) Metrics(c *gin.Context) {
	// Get database connection stats
	sqlDB, err := h.db.DB()
	var dbStats interface{}
	if err != nil {
		dbStats = map[string]interface{}{
			"error": err.Error(),
		}
	} else {
		stats := sqlDB.Stats()
		dbStats = map[string]interface{}{
			"open_connections":     stats.OpenConnections,
			"in_use":              stats.InUse,
			"idle":                stats.Idle,
			"wait_count":          stats.WaitCount,
			"wait_duration":       stats.WaitDuration.String(),
			"max_idle_closed":     stats.MaxIdleClosed,
			"max_idle_time_closed": stats.MaxIdleTimeClosed,
			"max_lifetime_closed": stats.MaxLifetimeClosed,
		}
	}

	// Get Redis pool stats
	redisStats := h.redisClient.PoolStats()

	c.JSON(http.StatusOK, gin.H{
		"timestamp": time.Now().UTC(),
		"database":  dbStats,
		"redis": map[string]interface{}{
			"hits":        redisStats.Hits,
			"misses":      redisStats.Misses,
			"timeouts":    redisStats.Timeouts,
			"total_conns": redisStats.TotalConns,
			"idle_conns":  redisStats.IdleConns,
			"stale_conns": redisStats.StaleConns,
		},
	})
}