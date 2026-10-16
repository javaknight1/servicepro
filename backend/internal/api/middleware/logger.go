package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger logs HTTP requests
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		startTime := time.Now()

		// Get request path
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)

		// Get status code
		statusCode := c.Writer.Status()

		// Get client IP
		clientIP := c.ClientIP()

		// Get method
		method := c.Request.Method

		// Get error message if any
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		// Query values (tokens, emails, codes) are never logged — only the
		// parameter names (T239)
		if keys := QueryKeys(raw); keys != "" {
			path = path + " query_keys=" + keys
		}

		// Log format: [timestamp] status latency client_ip method path error
		fmt.Printf("[%s] %d %v %s %s %s %s\n",
			time.Now().Format("2006-01-02 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
			errorMessage,
		)
	}
}

// StructuredLogger returns a structured logging middleware
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(startTime)

		// Build structured log
		logEntry := map[string]interface{}{
			"timestamp":  time.Now().Format(time.RFC3339),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"query_keys": QueryKeys(c.Request.URL.RawQuery), // names only, never values (T239)
			"status":     c.Writer.Status(),
			"latency_ms": latency.Milliseconds(),
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"request_id": c.GetString("request_id"),
		}

		// Add user info if authenticated
		if userID, exists := c.Get("userID"); exists {
			logEntry["user_id"] = userID
		}

		// Add error if present
		if len(c.Errors) > 0 {
			logEntry["error"] = c.Errors.String()
		}

		// In production, you would use a structured logger like zap or logrus
		// For now, we'll just print a formatted JSON-like output
		fmt.Printf("LOG: %+v\n", logEntry)
	}
}
