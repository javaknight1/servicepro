package middleware

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/internal/services/permissions"
)

// AccessControlMiddleware handles route protection and resource access control
type AccessControlMiddleware struct {
	config      *config.AccessControlConfig
	permChecker *permissions.PermissionChecker
	redisClient *redis.Client
}

// NewAccessControlMiddleware creates a new access control middleware
func NewAccessControlMiddleware(
	cfg *config.AccessControlConfig,
	permChecker *permissions.PermissionChecker,
	redisClient *redis.Client,
) *AccessControlMiddleware {
	return &AccessControlMiddleware{
		config:      cfg,
		permChecker: permChecker,
		redisClient: redisClient,
	}
}

// RouteProtection applies general route protection (IP filtering, rate limiting, etc.)
func (acm *AccessControlMiddleware) RouteProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !acm.config.RouteProtection.Enabled {
			c.Next()
			return
		}

		start := time.Now()

		// 1. IP-based access control
		if !acm.checkIPAccess(c) {
			return // Response already sent in checkIPAccess
		}

		// 2. Geographic restrictions (if implemented)
		if !acm.checkGeographicAccess(c) {
			return // Response already sent in checkGeographicAccess
		}

		// 3. Request size validation
		if !acm.checkRequestSize(c) {
			return // Response already sent in checkRequestSize
		}

		// 4. Route-specific rate limiting
		if !acm.checkRouteRateLimit(c) {
			return // Response already sent in checkRouteRateLimit
		}

		// Log performance if check took too long
		duration := time.Since(start)
		if duration > 50*time.Millisecond {
			log.Printf("[PERF] Route protection took %v for %s %s", duration, c.Request.Method, c.Request.URL.Path)
		}

		c.Next()
	}
}

// RequireResourceAccess checks if user has access to a specific resource
// Usage: router.GET("/users/:id", middleware.RequireResourceAccess("users", "read"), handler)
func (acm *AccessControlMiddleware) RequireResourceAccess(resourceType, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !acm.config.ResourceAccess.Enabled {
			c.Next()
			return
		}

		start := time.Now()

		// Get user ID from context (set by RequireAuth)
		userID, exists := acm.getUserID(c)
		if !exists {
			acm.denyAccess(c, "unauthorized", "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Get resource rule
		rule, exists := acm.config.ResourceAccess.GetResourceRule(resourceType)
		if !exists {
			// No specific rule defined, allow if basic permission check passes
			log.Printf("[WARN] No resource rule defined for: %s", resourceType)
			c.Next()
			return
		}

		// 1. Check required permissions for the action
		if !acm.checkResourcePermissions(c, userID, rule, action) {
			return // Response already sent
		}

		// 2. Check ownership if required
		if !acm.checkResourceOwnership(c, userID, rule, action) {
			return // Response already sent
		}

		// 3. Check hierarchy requirements
		if !acm.checkHierarchyRequirement(c, userID, rule) {
			return // Response already sent
		}

		// 4. Audit sensitive resource access
		if acm.config.ResourceAccess.IsSensitiveResource(resourceType) {
			acm.auditResourceAccess(c, userID, resourceType, action, "granted")
		}

		// Log performance if check took too long
		duration := time.Since(start)
		if duration > 100*time.Millisecond {
			log.Printf("[PERF] Resource access check took %v for user %s on %s.%s",
				duration, userID, resourceType, action)
		}

		c.Next()
	}
}

// RequireResourceOwnership checks if user owns the resource or has admin privileges
// Usage: router.PUT("/users/:id", middleware.RequireResourceOwnership("users"), handler)
func (acm *AccessControlMiddleware) RequireResourceOwnership(resourceType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !acm.config.ResourceAccess.EnableOwnershipCheck {
			c.Next()
			return
		}

		userID, exists := acm.getUserID(c)
		if !exists {
			acm.denyAccess(c, "unauthorized", "User not authenticated", http.StatusUnauthorized)
			return
		}

		// Get resource ID from URL parameter
		resourceIDStr := c.Param("id")
		if resourceIDStr == "" {
			acm.denyAccess(c, "bad_request", "Resource ID not provided", http.StatusBadRequest)
			return
		}

		resourceID, err := uuid.Parse(resourceIDStr)
		if err != nil {
			acm.denyAccess(c, "bad_request", "Invalid resource ID format", http.StatusBadRequest)
			return
		}

		// Get resource rule
		rule, exists := acm.config.ResourceAccess.GetResourceRule(resourceType)
		if !exists {
			// No rule, proceed
			c.Next()
			return
		}

		// Check if user owns the resource
		isOwner := userID == resourceID

		// If not owner, check if admin override is allowed
		if !isOwner && rule.AdminOverride {
			ctx := context.Background()
			isAdmin, err := acm.permChecker.IsAdmin(ctx, userID)
			if err != nil {
				log.Printf("[ERROR] Failed to check admin status for user %s: %v", userID, err)
				acm.denyAccess(c, "internal_error", "Failed to verify access", http.StatusInternalServerError)
				return
			}

			if isAdmin {
				log.Printf("[SECURITY] Admin %s accessed resource %s/%s with override",
					userID, resourceType, resourceID)
				c.Next()
				return
			}
		}

		// If not owner and no admin override, deny
		if !isOwner {
			log.Printf("[SECURITY] User %s denied access to resource %s/%s (not owner)",
				userID, resourceType, resourceID)
			acm.denyAccess(c, "forbidden", "You do not have access to this resource", http.StatusForbidden)
			return
		}

		c.Next()
	}
}

// ApplySecurityHeaders applies security headers to responses
func (acm *AccessControlMiddleware) ApplySecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !acm.config.RouteProtection.EnableSecurityHeaders {
			c.Next()
			return
		}

		// HSTS (HTTP Strict Transport Security)
		c.Header("Strict-Transport-Security",
			fmt.Sprintf("max-age=%d; includeSubDomains", acm.config.RouteProtection.HSTSMaxAge))

		// Content Security Policy
		if acm.config.RouteProtection.EnableCSP {
			c.Header("Content-Security-Policy", acm.config.RouteProtection.CSPDirectives)
		}

		// Other security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// CORS applies CORS headers based on configuration
func (acm *AccessControlMiddleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !acm.config.RouteProtection.EnableCORS {
			c.Next()
			return
		}

		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range acm.config.RouteProtection.AllowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				break
			}
		}

		if !allowed && origin != "" {
			acm.denyAccess(c, "cors_error", "Origin not allowed", http.StatusForbidden)
			return
		}

		// Set CORS headers
		if origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", strings.Join(acm.config.RouteProtection.AllowedMethods, ", "))
		c.Header("Access-Control-Allow-Headers", strings.Join(acm.config.RouteProtection.AllowedHeaders, ", "))

		if len(acm.config.RouteProtection.ExposedHeaders) > 0 {
			c.Header("Access-Control-Expose-Headers", strings.Join(acm.config.RouteProtection.ExposedHeaders, ", "))
		}

		if acm.config.RouteProtection.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		c.Header("Access-Control-Max-Age", fmt.Sprintf("%d", acm.config.RouteProtection.MaxAge))

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Private helper methods

// checkIPAccess validates IP-based access control
func (acm *AccessControlMiddleware) checkIPAccess(c *gin.Context) bool {
	clientIP := c.ClientIP()

	// Parse IP to validate it's a real IP
	ip := net.ParseIP(clientIP)
	if ip == nil {
		log.Printf("[WARN] Invalid client IP: %s", clientIP)
		// Allow if we can't parse IP (could be behind proxy)
		return true
	}

	if !acm.config.RouteProtection.IsIPAllowed(clientIP) {
		log.Printf("[SECURITY] IP %s blocked by access control on %s %s",
			clientIP, c.Request.Method, c.Request.URL.Path)
		acm.denyAccess(c, "ip_blocked", "Access denied from your IP address", http.StatusForbidden)
		return false
	}

	return true
}

// checkGeographicAccess validates geographic-based access control
func (acm *AccessControlMiddleware) checkGeographicAccess(c *gin.Context) bool {
	// This would require a GeoIP database/service integration
	// For now, we'll just return true
	// TODO: Implement GeoIP lookup if needed

	// Example implementation:
	// country := geoip.Lookup(c.ClientIP())
	// if !acm.config.RouteProtection.IsCountryAllowed(country) {
	//     acm.denyAccess(c, "geo_blocked", "Access denied from your location", http.StatusForbidden)
	//     return false
	// }

	return true
}

// checkRequestSize validates request size limits
func (acm *AccessControlMiddleware) checkRequestSize(c *gin.Context) bool {
	// Check content length
	if c.Request.ContentLength > acm.config.RouteProtection.MaxRequestSize {
		log.Printf("[SECURITY] Request too large: %d bytes from %s",
			c.Request.ContentLength, c.ClientIP())
		acm.denyAccess(c, "request_too_large",
			fmt.Sprintf("Request body too large (max: %d bytes)", acm.config.RouteProtection.MaxRequestSize),
			http.StatusRequestEntityTooLarge)
		return false
	}

	return true
}

// checkRouteRateLimit validates route-specific rate limits
func (acm *AccessControlMiddleware) checkRouteRateLimit(c *gin.Context) bool {
	// Skip rate limiting if Redis client is not configured
	if acm.redisClient == nil {
		return true
	}

	// Build route key
	routeKey := fmt.Sprintf("%s:%s", c.Request.Method, c.Request.URL.Path)

	// Check if there's a custom rate limit for this route
	customLimit, exists := acm.config.RouteProtection.CustomRateLimits[routeKey]

	var limit int
	var window time.Duration
	var limitKey string

	if exists {
		limit = customLimit.Limit
		window = customLimit.Window

		// Determine if we should limit by IP or user
		if customLimit.ByUser {
			userID, userExists := acm.getUserID(c)
			if userExists {
				limitKey = fmt.Sprintf("ratelimit:route:%s:user:%s", routeKey, userID)
			} else {
				// Fall back to IP if user not authenticated
				limitKey = fmt.Sprintf("ratelimit:route:%s:ip:%s", routeKey, c.ClientIP())
			}
		} else {
			limitKey = fmt.Sprintf("ratelimit:route:%s:ip:%s", routeKey, c.ClientIP())
		}
	} else {
		// Use default rate limit
		limit = acm.config.RouteProtection.DefaultRateLimit
		window = acm.config.RouteProtection.DefaultRateWindow
		limitKey = fmt.Sprintf("ratelimit:route:default:%s", c.ClientIP())
	}

	// Check rate limit using Redis
	ctx := context.Background()
	count, err := acm.redisClient.Incr(ctx, limitKey).Result()
	if err != nil {
		log.Printf("[ERROR] Failed to check rate limit: %v", err)
		// Allow request if Redis fails
		return true
	}

	// Set expiry on first request
	if count == 1 {
		acm.redisClient.Expire(ctx, limitKey, window)
	}

	// Check if limit exceeded
	if count > int64(limit) {
		ttl, _ := acm.redisClient.TTL(ctx, limitKey).Result()
		log.Printf("[SECURITY] Rate limit exceeded for %s from %s", routeKey, c.ClientIP())
		acm.denyAccess(c, "rate_limit_exceeded",
			fmt.Sprintf("Too many requests. Please try again in %v", ttl.Round(time.Second)),
			http.StatusTooManyRequests)
		return false
	}

	return true
}

// checkResourcePermissions validates required permissions for resource action
func (acm *AccessControlMiddleware) checkResourcePermissions(
	c *gin.Context,
	userID uuid.UUID,
	rule config.ResourceRule,
	action string,
) bool {
	requiredPerms, exists := rule.RequiredPermissions[action]
	if !exists || len(requiredPerms) == 0 {
		// No specific permissions required for this action
		return true
	}

	ctx := context.Background()
	hasPermission, err := acm.permChecker.CheckAnyPermission(ctx, userID, requiredPerms...)
	if err != nil {
		log.Printf("[ERROR] Permission check failed for user %s: %v", userID, err)
		acm.denyAccess(c, "internal_error", "Failed to verify permissions", http.StatusInternalServerError)
		return false
	}

	if !hasPermission {
		log.Printf("[SECURITY] User %s denied %s.%s - missing permissions: %v",
			userID, rule.Name, action, requiredPerms)
		acm.denyAccess(c, "forbidden",
			fmt.Sprintf("You do not have permission to %s %s", action, rule.Name),
			http.StatusForbidden)
		return false
	}

	return true
}

// checkResourceOwnership validates resource ownership
func (acm *AccessControlMiddleware) checkResourceOwnership(
	c *gin.Context,
	userID uuid.UUID,
	rule config.ResourceRule,
	action string,
) bool {
	// Check if this is an owner-only action
	isOwnerOnly := false
	for _, ownerAction := range rule.OwnerOnlyActions {
		if ownerAction == action {
			isOwnerOnly = true
			break
		}
	}

	if !isOwnerOnly {
		return true
	}

	// Get resource ID from URL
	resourceIDStr := c.Param("id")
	if resourceIDStr == "" {
		// No resource ID in URL, can't check ownership
		return true
	}

	resourceID, err := uuid.Parse(resourceIDStr)
	if err != nil {
		acm.denyAccess(c, "bad_request", "Invalid resource ID", http.StatusBadRequest)
		return false
	}

	// Check if user owns the resource
	isOwner := userID == resourceID

	// If not owner, check admin override
	if !isOwner && rule.AdminOverride {
		ctx := context.Background()
		isAdmin, err := acm.permChecker.IsAdmin(ctx, userID)
		if err == nil && isAdmin {
			log.Printf("[SECURITY] Admin %s overriding ownership for %s.%s on resource %s",
				userID, rule.Name, action, resourceID)
			return true
		}
	}

	if !isOwner {
		log.Printf("[SECURITY] User %s denied %s.%s on resource %s - not owner",
			userID, rule.Name, action, resourceID)
		acm.denyAccess(c, "forbidden", "You can only perform this action on your own resources", http.StatusForbidden)
		return false
	}

	return true
}

// checkHierarchyRequirement validates minimum hierarchy level
func (acm *AccessControlMiddleware) checkHierarchyRequirement(
	c *gin.Context,
	userID uuid.UUID,
	rule config.ResourceRule,
) bool {
	if !acm.config.ResourceAccess.EnableHierarchyCheck || rule.MinHierarchy == 0 {
		return true
	}

	ctx := context.Background()
	role, err := acm.permChecker.GetUserHighestRole(ctx, userID)
	if err != nil {
		log.Printf("[ERROR] Failed to get role for user %s: %v", userID, err)
		acm.denyAccess(c, "internal_error", "Failed to verify access level", http.StatusInternalServerError)
		return false
	}

	if role.HierarchyLevel < rule.MinHierarchy {
		log.Printf("[SECURITY] User %s denied %s - insufficient hierarchy: %d < %d",
			userID, rule.Name, role.HierarchyLevel, rule.MinHierarchy)
		acm.denyAccess(c, "forbidden", "Your access level is insufficient for this action", http.StatusForbidden)
		return false
	}

	return true
}

// auditResourceAccess logs resource access for audit trail
func (acm *AccessControlMiddleware) auditResourceAccess(
	c *gin.Context,
	userID uuid.UUID,
	resourceType, action, result string,
) {
	log.Printf("[AUDIT] User: %s, Resource: %s, Action: %s, Result: %s, IP: %s, Path: %s",
		userID, resourceType, action, result, c.ClientIP(), c.Request.URL.Path)
}

// denyAccess sends an access denied response
func (acm *AccessControlMiddleware) denyAccess(c *gin.Context, errorCode, message string, statusCode int) {
	c.JSON(statusCode, models.ErrorResponse{
		Error:   errorCode,
		Message: message,
	})
	c.Abort()
}

// getUserID extracts the user ID from gin context
func (acm *AccessControlMiddleware) getUserID(c *gin.Context) (uuid.UUID, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}

	userID, ok := value.(uuid.UUID)
	if !ok {
		return uuid.Nil, false
	}

	return userID, true
}
