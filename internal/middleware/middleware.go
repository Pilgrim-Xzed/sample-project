package middleware

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// CORS returns a CORS middleware with default configuration
func CORS() gin.HandlerFunc {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-Request-ID"}
	config.ExposeHeaders = []string{"X-Request-ID"}
	config.AllowCredentials = true
	config.MaxAge = 12 * time.Hour

	return cors.New(config)
}

// RequestID adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		c.Header("X-Request-ID", requestID)
		c.Set("request_id", requestID)
		c.Next()
	}
}

// Logger creates a structured logging middleware
func Logger(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery
		
		// Process request
		c.Next()
		
		// Calculate latency
		latency := time.Since(start)
		
		// Get request ID
		requestID, _ := c.Get("request_id")
		
		// Build full path
		if raw != "" {
			path = path + "?" + raw
		}
		
		// Log request
		logger.WithFields(logrus.Fields{
			"request_id":   requestID,
			"status":       c.Writer.Status(),
			"method":       c.Request.Method,
			"path":         path,
			"ip":           c.ClientIP(),
			"user_agent":   c.Request.UserAgent(),
			"latency":      latency,
			"latency_ms":   latency.Milliseconds(),
			"body_size":    c.Writer.Size(),
		}).Info("HTTP Request")
		
		// Log errors if any
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				logger.WithFields(logrus.Fields{
					"request_id": requestID,
					"error":      err.Error(),
				}).Error("Request Error")
			}
		}
	}
}

// ErrorHandler handles panics and returns JSON errors
func ErrorHandler() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			c.JSON(500, gin.H{
				"error":   "Internal Server Error",
				"message": err,
			})
		} else {
			c.JSON(500, gin.H{
				"error":   "Internal Server Error",
				"message": "An unexpected error occurred",
			})
		}
		c.Abort()
	})
}

// RateLimit creates a simple rate limiting middleware
func RateLimit() gin.HandlerFunc {
	// This is a simple in-memory rate limiter
	// In production, you'd want to use Redis-based rate limiting
	clients := make(map[string][]time.Time)
	
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()
		
		// Clean old entries (older than 1 minute)
		if timestamps, exists := clients[clientIP]; exists {
			var validTimestamps []time.Time
			for _, timestamp := range timestamps {
				if now.Sub(timestamp) < time.Minute {
					validTimestamps = append(validTimestamps, timestamp)
				}
			}
			clients[clientIP] = validTimestamps
		}
		
		// Check rate limit (100 requests per minute)
		if len(clients[clientIP]) >= 100 {
			c.JSON(429, gin.H{
				"error":   "Rate Limit Exceeded",
				"message": "Too many requests",
			})
			c.Abort()
			return
		}
		
		// Add current request
		clients[clientIP] = append(clients[clientIP], now)
		c.Next()
	}
}

// Timeout creates a timeout middleware
func Timeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set timeout on the context
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// SecurityHeaders adds security headers to responses
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")
		c.Next()
	}
}

// JSONContentType ensures content type is application/json for API routes
func JSONContentType() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Type", "application/json")
		c.Next()
	}
}

// ValidateContentType validates that request content type is JSON for POST/PUT requests
func ValidateContentType() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "POST" || c.Request.Method == "PUT" || c.Request.Method == "PATCH" {
			contentType := c.GetHeader("Content-Type")
			if contentType != "application/json" && contentType != "application/json; charset=utf-8" {
				c.JSON(400, gin.H{
					"error":   "Invalid Content Type",
					"message": "Content-Type must be application/json",
				})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// Pagination adds pagination parameters to context
func Pagination() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Default pagination values
		offset := 0
		limit := 20
		
		// Parse offset
		if offsetStr := c.Query("offset"); offsetStr != "" {
			if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
				offset = parsedOffset
			}
		}
		
		// Parse limit (max 100)
		if limitStr := c.Query("limit"); limitStr != "" {
			if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
				if parsedLimit > 100 {
					limit = 100
				} else {
					limit = parsedLimit
				}
			}
		}
		
		c.Set("offset", offset)
		c.Set("limit", limit)
		c.Next()
	}
}

// HealthCheck creates a simple health check middleware
func HealthCheck(path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == path {
			c.JSON(200, gin.H{
				"status":    "ok",
				"timestamp": time.Now().UTC(),
			})
			c.Abort()
			return
		}
		c.Next()
	}
}