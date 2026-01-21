# Middleware Documentation

This document describes the access control and logging middleware components implemented in the ServicePro backend.

## Overview

Three middleware components have been implemented:

1. **access_control.go** - Route protection and resource access control
2. **access_logging.go** - Access logging and metrics collection
3. **config/access_control.go** - Configuration for both middleware components

## 1. Access Control Middleware

The `AccessControlMiddleware` provides comprehensive route protection and resource-level access control.

### Features

- IP-based access control (whitelist/blacklist)
- Geographic restrictions (framework in place)
- Request size validation
- Route-specific rate limiting
- Security headers (HSTS, CSP, etc.)
- CORS handling
- Resource ownership verification
- Hierarchical access control
- Admin override capabilities

### Usage

#### Basic Setup

```go
import (
    "github.com/javaknight1/servicepro/backend/config"
    "github.com/javaknight1/servicepro/backend/internal/api/middleware"
    "github.com/javaknight1/servicepro/backend/internal/services/permissions"
)

// Load configuration
cfg := config.LoadAccessControlConfig()

// Create middleware
accessControl := middleware.NewAccessControlMiddleware(
    cfg,
    permissionChecker,
    redisClient,
)
```

#### Apply Route Protection

```go
// Apply global route protection
router.Use(accessControl.RouteProtection())

// Apply security headers
router.Use(accessControl.ApplySecurityHeaders())

// Apply CORS
router.Use(accessControl.CORS())
```

#### Resource Access Control

```go
// Require specific resource access
router.GET("/users/:id",
    accessControl.RequireResourceAccess("users", "read"),
    handler,
)

// Require resource ownership
router.PUT("/users/:id",
    accessControl.RequireResourceOwnership("users"),
    handler,
)
```

### Configuration

Configure via environment variables in `config/access_control.go`:

```bash
# Route Protection
ACCESS_CONTROL_ENABLED=true
IP_WHITELIST=192.168.1.1,192.168.1.2
IP_BLACKLIST=10.0.0.100
DEFAULT_RATE_LIMIT=100
DEFAULT_RATE_WINDOW=1m
MAX_REQUEST_SIZE=10485760  # 10MB

# Security Headers
ENABLE_SECURITY_HEADERS=true
HSTS_MAX_AGE=31536000
ENABLE_CSP=true
CSP_DIRECTIVES="default-src 'self'; script-src 'self'"

# CORS
ENABLE_CORS=true
CORS_ALLOWED_ORIGINS=http://localhost:3000,https://example.com
CORS_ALLOWED_METHODS=GET,POST,PUT,PATCH,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Origin,Content-Type,Authorization
CORS_ALLOW_CREDENTIALS=true

# Resource Access
RESOURCE_ACCESS_ENABLED=true
ENABLE_OWNERSHIP_CHECK=true
ENABLE_HIERARCHY_CHECK=true
SENSITIVE_RESOURCES=users,roles,permissions
AUDIT_SENSITIVE_ACCESS=true
```

### Resource Rules

Define custom resource rules in code:

```go
cfg.ResourceAccess.Resources["custom_resource"] = config.ResourceRule{
    Name: "custom_resource",
    RequiredPermissions: map[string][]string{
        "read":   {"custom.read"},
        "write":  {"custom.write"},
        "delete": {"custom.delete"},
    },
    OwnerOnlyActions: []string{"delete"},
    AdminOverride:    true,
    MinHierarchy:     50,
}
```

### Middleware Methods

#### `RouteProtection()`

Applies general route protection including IP filtering, rate limiting, and request size validation.

#### `RequireResourceAccess(resourceType, action)`

Checks if user has permission to perform an action on a resource type.

#### `RequireResourceOwnership(resourceType)`

Verifies user owns the resource being accessed (based on URL parameter `:id`).

#### `ApplySecurityHeaders()`

Adds security headers to all responses (HSTS, CSP, X-Frame-Options, etc.).

#### `CORS()`

Handles CORS preflight requests and sets appropriate CORS headers.

## 2. Access Logging Middleware

The `AccessLogger` provides comprehensive request/response logging and metrics collection.

### Features

- Configurable log formats (JSON, text, Apache combined)
- Request/response body logging
- Sensitive data filtering
- Performance metrics tracking
- Slow request detection
- Audit trail for sensitive operations
- Log rotation support
- Real-time metrics

### Usage

#### Basic Setup

```go
import (
    "github.com/javaknight1/servicepro/backend/config"
    "github.com/javaknight1/servicepro/backend/internal/api/middleware"
)

// Load configuration
cfg := config.LoadAccessControlConfig()

// Create logger
accessLogger, err := middleware.NewAccessLogger(&cfg.AccessLogging)
if err != nil {
    log.Fatalf("Failed to create access logger: %v", err)
}

// Apply middleware
router.Use(accessLogger.AccessLogging())
```

#### Audit Logging

```go
// Log sensitive operations
func deleteUserHandler(c *gin.Context) {
    userID := c.Param("id")

    // ... delete user logic ...

    accessLogger.AuditLog(
        c,
        "delete",        // action
        "users",         // resource
        userID,          // resource ID
        "success",       // result
        "User deleted",  // details
    )

    c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
```

#### Metrics

```go
// Get current metrics
metrics := accessLogger.GetMetrics()
fmt.Printf("Total Requests: %d\n", metrics.TotalRequests)
fmt.Printf("Avg Response Time: %v\n",
    metrics.TotalResponseTime / time.Duration(metrics.TotalRequests))

// Get formatted metrics
formatted := accessLogger.FormatMetrics()
fmt.Println(formatted)

// Reset metrics
accessLogger.ResetMetrics()
```

### Configuration

Configure via environment variables:

```bash
# Logging
ACCESS_LOGGING_ENABLED=true
LOG_SUCCESSFUL_REQUESTS=true
LOG_FAILED_REQUESTS=true
LOG_DENIED_ACCESS=true
LOG_SLOW_REQUESTS=true
SLOW_REQUEST_THRESHOLD=1s

# Sensitive Data
FILTER_SENSITIVE_DATA=true
SENSITIVE_HEADERS=Authorization,Cookie,X-API-Key
SENSITIVE_PARAMS=password,token,secret,api_key

# What to Log
LOG_REQUEST_BODY=false
LOG_RESPONSE_BODY=false
LOG_HEADERS=true
LOG_QUERY_PARAMS=true
MAX_BODY_LOG_SIZE=1024

# Metrics
ENABLE_METRICS=true
METRICS_PREFIX=servicepro
TRACK_RESPONSE_TIME=true
TRACK_REQUEST_SIZE=true
TRACK_RESPONSE_SIZE=true

# Format and Output
LOG_FORMAT=json  # json, text, or combined
INCLUDE_USER_INFO=true
INCLUDE_IP_INFO=true
LOG_TO_FILE=true
LOG_FILE_PATH=logs/access.log
LOG_TO_STDOUT=true

# Log Rotation
LOG_ROTATION_ENABLED=true
LOG_MAX_SIZE=100      # MB
LOG_MAX_BACKUPS=10
LOG_MAX_AGE=30        # days
LOG_COMPRESS=true

# Audit Trail
ENABLE_AUDIT_TRAIL=true
AUDIT_TRAIL_PATH=logs/audit.log
AUDIT_SENSITIVE_OPS=delete,update_permissions,update_role
```

### Log Formats

#### JSON Format

```json
{
  "timestamp": "2024-01-15T10:30:45Z",
  "request_id": "123e4567-e89b-12d3-a456-426614174000",
  "method": "GET",
  "path": "/api/users/123",
  "status_code": 200,
  "response_time_ms": 45,
  "client_ip": "192.168.1.100",
  "user_id": "user-uuid",
  "user_email": "user@example.com"
}
```

#### Text Format

```
[2024-01-15 10:30:45] 123e4567-e89b-12d3-a456-426614174000 | 200 | 45ms | 192.168.1.100 | GET /api/users/123 | User: user-uuid
```

#### Apache Combined Format

```
192.168.1.100 - user-uuid [15/Jan/2024:10:30:45 -0700] "GET /api/users/123 HTTP/1.1" 200 1234 "-" "Mozilla/5.0" 45ms
```

### Metrics Structure

The metrics collection tracks:

- **Total Requests**: Count of all requests
- **Successful Requests**: 2xx and 3xx responses
- **Failed Requests**: 4xx and 5xx responses
- **Denied Requests**: 403 responses
- **Slow Requests**: Requests exceeding threshold
- **Response Time**: Total and average
- **Request/Response Sizes**: Total bytes
- **Requests by Method**: Breakdown by HTTP method
- **Requests by Path**: Breakdown by endpoint
- **Requests by Status**: Breakdown by status code

## 3. Middleware Chaining

Recommended middleware order:

```go
// 1. Access logging (first to capture everything)
router.Use(accessLogger.AccessLogging())

// 2. Security headers
router.Use(accessControl.ApplySecurityHeaders())

// 3. CORS
router.Use(accessControl.CORS())

// 4. Route protection
router.Use(accessControl.RouteProtection())

// 5. Authentication (from existing middleware)
router.Use(permissionMiddleware.RequireAuth())

// 6. Resource-specific middleware on routes
router.GET("/users/:id",
    accessControl.RequireResourceAccess("users", "read"),
    getUserHandler,
)

router.PUT("/users/:id",
    accessControl.RequireResourceOwnership("users"),
    updateUserHandler,
)
```

## 4. Performance Considerations

### Caching

- Permission checks are cached in Redis (5-minute TTL)
- Rate limiting uses Redis for distributed rate limiting
- Metrics are stored in memory with mutex protection

### Logging

- Log rotation prevents disk space issues
- Configurable log levels reduce I/O overhead
- Sensitive data filtering happens before writing
- Body logging is optional and size-limited

### Metrics

- In-memory metrics with minimal overhead
- Lock-free reads for metrics retrieval
- Path tracking limited to prevent unbounded growth

## 5. Security Best Practices

### Production Configuration

```bash
# Strict security headers
ENABLE_SECURITY_HEADERS=true
HSTS_MAX_AGE=31536000
ENABLE_CSP=true

# Rate limiting
DEFAULT_RATE_LIMIT=100
DEFAULT_RATE_WINDOW=1m

# Request limits
MAX_REQUEST_SIZE=10485760  # 10MB
MAX_HEADER_SIZE=1048576    # 1MB

# Logging
FILTER_SENSITIVE_DATA=true
LOG_REQUEST_BODY=false
LOG_RESPONSE_BODY=false
ENABLE_AUDIT_TRAIL=true

# CORS - be specific
CORS_ALLOWED_ORIGINS=https://yourdomain.com
CORS_ALLOW_CREDENTIALS=true
```

### Sensitive Operations

Always audit sensitive operations:

```go
// Before performing sensitive action
accessLogger.AuditLog(c, "delete_user", "users", userID, "attempt", "")

// After action
if err != nil {
    accessLogger.AuditLog(c, "delete_user", "users", userID, "failure", err.Error())
} else {
    accessLogger.AuditLog(c, "delete_user", "users", userID, "success", "")
}
```

## 6. Testing

Comprehensive tests are provided:

```bash
# Run all middleware tests
go test ./internal/api/middleware/... -v

# Run specific test
go test ./internal/api/middleware -run TestAccessLogging_SuccessfulRequest -v

# Run benchmarks
go test ./internal/api/middleware -bench=. -benchmem
```

## 7. Troubleshooting

### Common Issues

**Issue**: Rate limit Redis errors

```
Solution: Check Redis connection, middleware will allow requests if Redis fails
```

**Issue**: Logs not being written

```
Solution: Check log directory permissions, ensure LOG_FILE_PATH is writable
```

**Issue**: Metrics showing zero

```
Solution: Ensure ENABLE_METRICS=true and middleware is applied before routes
```

**Issue**: CORS errors

```
Solution: Add origin to CORS_ALLOWED_ORIGINS, check CORS middleware is applied
```

### Debug Mode

Enable debug logging:

```go
log.SetLevel(log.DebugLevel)
```

This will show cache hits/misses, permission checks, and performance warnings.

## 8. Examples

See `internal/api/routes/routes.go` for integration examples with the existing application.

## 9. Migration Guide

To integrate these middleware components into an existing application:

1. Add configuration to `.env`
2. Create access logger in `main.go`
3. Apply middleware in order shown above
4. Update route handlers to use resource access middleware
5. Add audit logging to sensitive operations
6. Configure log rotation
7. Set up monitoring for metrics endpoint

## Support

For issues or questions, please refer to the test files or open an issue in the project repository.
