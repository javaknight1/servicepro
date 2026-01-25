package middleware

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/config"
)

// CORSMiddleware returns a middleware that handles Cross-Origin Resource Sharing (CORS).
//
// CORS is a security mechanism that allows browsers to make cross-origin requests.
// Without proper CORS headers, browsers will block requests from different origins.
//
// Key behaviors:
//   - Handles preflight OPTIONS requests automatically
//   - Validates origins against the allowed list
//   - Blocks wildcard origins in production when credentials are enabled
//   - Caches preflight responses for the configured duration
func CORSMiddleware(cfg *config.CORSConfig, env string) gin.HandlerFunc {
	if cfg == nil || !cfg.Enabled {
		return func(c *gin.Context) {
			c.Next()
		}
	}

	// Validate configuration
	validateCORSConfig(cfg, env)

	// Pre-compute allowed origins map for O(1) lookup
	allowedOriginsMap := make(map[string]bool)
	allowAllOrigins := false
	for _, origin := range cfg.AllowedOrigins {
		if origin == "*" {
			allowAllOrigins = true
		} else {
			allowedOriginsMap[origin] = true
		}
	}

	// Pre-compute header values
	allowMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowHeaders := strings.Join(cfg.AllowedHeaders, ", ")
	exposeHeaders := strings.Join(cfg.ExposedHeaders, ", ")
	maxAge := strconv.Itoa(cfg.MaxAge)
	allowCredentials := strconv.FormatBool(cfg.AllowCredentials)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// No Origin header means same-origin request or non-browser client
		if origin == "" {
			c.Next()
			return
		}

		// Check if origin is allowed
		originAllowed := allowAllOrigins || allowedOriginsMap[origin]

		if !originAllowed {
			// Origin not allowed - don't set CORS headers
			// Browser will block the response
			if c.Request.Method == http.MethodOptions {
				// For preflight, return 403 to clearly indicate rejection
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			// For actual requests, proceed but browser will block due to missing headers
			c.Next()
			return
		}

		// Set CORS headers for allowed origins
		// Use the actual origin, not "*", when credentials are allowed
		if cfg.AllowCredentials || !allowAllOrigins {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			c.Header("Access-Control-Allow-Origin", "*")
		}

		// Vary header is important for caching
		// Tells caches that the response varies based on the Origin header
		c.Header("Vary", "Origin")

		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", allowCredentials)
		}

		// Handle preflight (OPTIONS) requests
		if c.Request.Method == http.MethodOptions {
			// Preflight request - browser is asking if actual request is allowed
			c.Header("Access-Control-Allow-Methods", allowMethods)
			c.Header("Access-Control-Allow-Headers", allowHeaders)

			if cfg.MaxAge > 0 {
				c.Header("Access-Control-Max-Age", maxAge)
			}

			// Return 204 No Content for successful preflight
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		// For actual requests, expose headers if configured
		if exposeHeaders != "" {
			c.Header("Access-Control-Expose-Headers", exposeHeaders)
		}

		c.Next()
	}
}

// validateCORSConfig checks for common misconfigurations
func validateCORSConfig(cfg *config.CORSConfig, env string) {
	isProduction := strings.ToLower(env) == "production" || strings.ToLower(env) == "prod"

	// Check for dangerous wildcard in production
	for _, origin := range cfg.AllowedOrigins {
		if origin == "*" && isProduction {
			if cfg.AllowCredentials {
				// This is a critical security issue
				log.Printf("SECURITY WARNING: CORS wildcard origin '*' with credentials enabled in production. " +
					"This is insecure and browsers will reject it anyway. " +
					"Set specific origins in CORS_ALLOWED_ORIGINS.")
			} else {
				log.Printf("CORS WARNING: Using wildcard origin '*' in production. " +
					"Consider restricting to specific origins for better security.")
			}
		}
	}

	// Warn about localhost in production
	if isProduction {
		for _, origin := range cfg.AllowedOrigins {
			if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
				log.Printf("CORS WARNING: Localhost origin '%s' configured in production. "+
					"This may be unintentional.", origin)
			}
		}
	}

	// Check for empty origins
	if len(cfg.AllowedOrigins) == 0 {
		log.Printf("CORS WARNING: No allowed origins configured. All cross-origin requests will be blocked.")
	}
}

// NewCORSConfig creates a CORSConfig from environment-based defaults
// This is useful for testing or when you need to create config programmatically
func NewCORSConfig(frontendURL string, env string) *config.CORSConfig {
	isProduction := strings.ToLower(env) == "production" || strings.ToLower(env) == "prod"

	var allowedOrigins []string
	if isProduction {
		// In production, only allow the configured frontend URL
		if frontendURL != "" {
			allowedOrigins = []string{frontendURL}
		}
	} else {
		// In development, allow common local development URLs
		allowedOrigins = []string{
			"http://localhost:3000",
			"http://localhost:5173",
			"http://127.0.0.1:3000",
			"http://127.0.0.1:5173",
		}
		if frontendURL != "" && !corsContains(allowedOrigins, frontendURL) {
			allowedOrigins = append(allowedOrigins, frontendURL)
		}
	}

	return &config.CORSConfig{
		Enabled:        true,
		AllowedOrigins: allowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-Request-ID",
			"X-CSRF-Token",
		},
		ExposedHeaders: []string{
			"X-Request-ID",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
		},
		AllowCredentials: true,
		MaxAge:           86400, // 24 hours
	}
}

// contains checks if a slice contains a string
func corsContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
