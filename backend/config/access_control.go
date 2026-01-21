package config

import (
	"regexp"
	"time"
)

// AccessControlConfig holds configuration for access control and logging
type AccessControlConfig struct {
	RouteProtection RouteProtectionConfig
	AccessLogging   AccessLoggingConfig
	ResourceAccess  ResourceAccessConfig
}

// RouteProtectionConfig defines route protection rules
type RouteProtectionConfig struct {
	// Enable route protection globally
	Enabled bool

	// IP-based access control
	IPWhitelist []string
	IPBlacklist []string

	// Geographic restrictions
	AllowedCountries []string
	BlockedCountries []string

	// Rate limiting per route
	DefaultRateLimit  int           // requests per window
	DefaultRateWindow time.Duration // time window
	CustomRateLimits  map[string]RouteRateLimit

	// Request size limits
	MaxRequestSize     int64 // in bytes
	MaxHeaderSize      int64 // in bytes
	MaxMultipartMemory int64 // in bytes

	// Security headers
	EnableSecurityHeaders bool
	HSTSMaxAge            int // in seconds
	EnableCSP             bool
	CSPDirectives         string

	// CORS settings
	EnableCORS       bool
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int // in seconds
}

// RouteRateLimit defines custom rate limit for a specific route
type RouteRateLimit struct {
	Path   string
	Method string
	Limit  int
	Window time.Duration
	ByIP   bool // rate limit by IP address
	ByUser bool // rate limit by authenticated user
}

// ResourceAccessConfig defines resource-level access control rules
type ResourceAccessConfig struct {
	// Enable resource access checking
	Enabled bool

	// Resource ownership validation
	EnableOwnershipCheck bool

	// Hierarchical access rules
	EnableHierarchyCheck bool

	// Resource-specific rules
	Resources map[string]ResourceRule

	// Sensitive resources that require additional checks
	SensitiveResources []string

	// Audit sensitive resource access
	AuditSensitiveAccess bool
}

// ResourceRule defines access rules for a specific resource type
type ResourceRule struct {
	Name string

	// Required permissions for each action
	RequiredPermissions map[string][]string // action -> permissions

	// Owner-only actions
	OwnerOnlyActions []string

	// Admin override - admins can bypass ownership checks
	AdminOverride bool

	// Hierarchy requirement - minimum hierarchy level needed
	MinHierarchy int

	// Custom validation function name (for extensibility)
	CustomValidator string
}

// AccessLoggingConfig defines logging and metrics configuration
type AccessLoggingConfig struct {
	// Enable access logging
	Enabled bool

	// Log levels
	LogSuccessfulRequests bool
	LogFailedRequests     bool
	LogDeniedAccess       bool
	LogSlowRequests       bool
	SlowRequestThreshold  time.Duration

	// Sensitive data filtering
	FilterSensitiveData bool
	SensitiveHeaders    []string
	SensitiveParams     []string

	// Request/Response logging
	LogRequestBody  bool
	LogResponseBody bool
	LogHeaders      bool
	LogQueryParams  bool
	MaxBodyLogSize  int // in bytes

	// Performance metrics
	EnableMetrics     bool
	MetricsPrefix     string
	TrackResponseTime bool
	TrackRequestSize  bool
	TrackResponseSize bool

	// Log format
	LogFormat       string // "json", "text", "combined"
	IncludeUserInfo bool
	IncludeIPInfo   bool

	// Log output
	LogToFile   bool
	LogFilePath string
	LogToStdout bool
	LogRotation LogRotationConfig

	// Audit trail
	EnableAuditTrail  bool
	AuditTrailPath    string
	AuditSensitiveOps []string // operations to always audit
	AuditByResource   map[string]bool
}

// LogRotationConfig defines log rotation settings
type LogRotationConfig struct {
	Enabled    bool
	MaxSize    int // in megabytes
	MaxBackups int
	MaxAge     int // in days
	Compress   bool
}

// LoadAccessControlConfig loads access control configuration with defaults
func LoadAccessControlConfig() *AccessControlConfig {
	return &AccessControlConfig{
		RouteProtection: RouteProtectionConfig{
			Enabled:               getEnvAsBool("ACCESS_CONTROL_ENABLED", true),
			IPWhitelist:           getEnvAsSlice("IP_WHITELIST", []string{}),
			IPBlacklist:           getEnvAsSlice("IP_BLACKLIST", []string{}),
			AllowedCountries:      getEnvAsSlice("ALLOWED_COUNTRIES", []string{}),
			BlockedCountries:      getEnvAsSlice("BLOCKED_COUNTRIES", []string{}),
			DefaultRateLimit:      getEnvAsInt("DEFAULT_RATE_LIMIT", 100),
			DefaultRateWindow:     getEnvAsDuration("DEFAULT_RATE_WINDOW", "1m"),
			CustomRateLimits:      make(map[string]RouteRateLimit),
			MaxRequestSize:        getEnvAsInt64("MAX_REQUEST_SIZE", 10*1024*1024),     // 10MB
			MaxHeaderSize:         getEnvAsInt64("MAX_HEADER_SIZE", 1*1024*1024),       // 1MB
			MaxMultipartMemory:    getEnvAsInt64("MAX_MULTIPART_MEMORY", 32*1024*1024), // 32MB
			EnableSecurityHeaders: getEnvAsBool("ENABLE_SECURITY_HEADERS", true),
			HSTSMaxAge:            getEnvAsInt("HSTS_MAX_AGE", 31536000), // 1 year
			EnableCSP:             getEnvAsBool("ENABLE_CSP", true),
			CSPDirectives:         getEnv("CSP_DIRECTIVES", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'"),
			EnableCORS:            getEnvAsBool("ENABLE_CORS", true),
			AllowedOrigins:        getEnvAsSlice("CORS_ALLOWED_ORIGINS", []string{"http://localhost:3000"}),
			AllowedMethods:        getEnvAsSlice("CORS_ALLOWED_METHODS", []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}),
			AllowedHeaders:        getEnvAsSlice("CORS_ALLOWED_HEADERS", []string{"Origin", "Content-Type", "Authorization"}),
			ExposedHeaders:        getEnvAsSlice("CORS_EXPOSED_HEADERS", []string{}),
			AllowCredentials:      getEnvAsBool("CORS_ALLOW_CREDENTIALS", true),
			MaxAge:                getEnvAsInt("CORS_MAX_AGE", 3600),
		},
		AccessLogging: AccessLoggingConfig{
			Enabled:               getEnvAsBool("ACCESS_LOGGING_ENABLED", true),
			LogSuccessfulRequests: getEnvAsBool("LOG_SUCCESSFUL_REQUESTS", true),
			LogFailedRequests:     getEnvAsBool("LOG_FAILED_REQUESTS", true),
			LogDeniedAccess:       getEnvAsBool("LOG_DENIED_ACCESS", true),
			LogSlowRequests:       getEnvAsBool("LOG_SLOW_REQUESTS", true),
			SlowRequestThreshold:  getEnvAsDuration("SLOW_REQUEST_THRESHOLD", "1s"),
			FilterSensitiveData:   getEnvAsBool("FILTER_SENSITIVE_DATA", true),
			SensitiveHeaders:      getEnvAsSlice("SENSITIVE_HEADERS", []string{"Authorization", "Cookie", "X-API-Key"}),
			SensitiveParams:       getEnvAsSlice("SENSITIVE_PARAMS", []string{"password", "token", "secret", "api_key"}),
			LogRequestBody:        getEnvAsBool("LOG_REQUEST_BODY", false),
			LogResponseBody:       getEnvAsBool("LOG_RESPONSE_BODY", false),
			LogHeaders:            getEnvAsBool("LOG_HEADERS", true),
			LogQueryParams:        getEnvAsBool("LOG_QUERY_PARAMS", true),
			MaxBodyLogSize:        getEnvAsInt("MAX_BODY_LOG_SIZE", 1024), // 1KB
			EnableMetrics:         getEnvAsBool("ENABLE_METRICS", true),
			MetricsPrefix:         getEnv("METRICS_PREFIX", "servicepro"),
			TrackResponseTime:     getEnvAsBool("TRACK_RESPONSE_TIME", true),
			TrackRequestSize:      getEnvAsBool("TRACK_REQUEST_SIZE", true),
			TrackResponseSize:     getEnvAsBool("TRACK_RESPONSE_SIZE", true),
			LogFormat:             getEnv("LOG_FORMAT", "json"),
			IncludeUserInfo:       getEnvAsBool("INCLUDE_USER_INFO", true),
			IncludeIPInfo:         getEnvAsBool("INCLUDE_IP_INFO", true),
			LogToFile:             getEnvAsBool("LOG_TO_FILE", true),
			LogFilePath:           getEnv("LOG_FILE_PATH", "logs/access.log"),
			LogToStdout:           getEnvAsBool("LOG_TO_STDOUT", true),
			LogRotation: LogRotationConfig{
				Enabled:    getEnvAsBool("LOG_ROTATION_ENABLED", true),
				MaxSize:    getEnvAsInt("LOG_MAX_SIZE", 100), // 100MB
				MaxBackups: getEnvAsInt("LOG_MAX_BACKUPS", 10),
				MaxAge:     getEnvAsInt("LOG_MAX_AGE", 30), // 30 days
				Compress:   getEnvAsBool("LOG_COMPRESS", true),
			},
			EnableAuditTrail:  getEnvAsBool("ENABLE_AUDIT_TRAIL", true),
			AuditTrailPath:    getEnv("AUDIT_TRAIL_PATH", "logs/audit.log"),
			AuditSensitiveOps: getEnvAsSlice("AUDIT_SENSITIVE_OPS", []string{"delete", "update_permissions", "update_role"}),
			AuditByResource:   make(map[string]bool),
		},
		ResourceAccess: ResourceAccessConfig{
			Enabled:              getEnvAsBool("RESOURCE_ACCESS_ENABLED", true),
			EnableOwnershipCheck: getEnvAsBool("ENABLE_OWNERSHIP_CHECK", true),
			EnableHierarchyCheck: getEnvAsBool("ENABLE_HIERARCHY_CHECK", true),
			Resources:            GetDefaultResourceRules(),
			SensitiveResources:   getEnvAsSlice("SENSITIVE_RESOURCES", []string{"users", "roles", "permissions"}),
			AuditSensitiveAccess: getEnvAsBool("AUDIT_SENSITIVE_ACCESS", true),
		},
	}
}

// GetDefaultResourceRules returns default resource access rules
func GetDefaultResourceRules() map[string]ResourceRule {
	return map[string]ResourceRule{
		"users": {
			Name: "users",
			RequiredPermissions: map[string][]string{
				"list":   {"users.list"},
				"read":   {"users.read"},
				"create": {"users.create"},
				"update": {"users.update"},
				"delete": {"users.delete"},
			},
			OwnerOnlyActions: []string{"update", "delete"},
			AdminOverride:    true,
			MinHierarchy:     0,
		},
		"roles": {
			Name: "roles",
			RequiredPermissions: map[string][]string{
				"list":   {"roles.list"},
				"read":   {"roles.read"},
				"create": {"roles.create"},
				"update": {"roles.update"},
				"delete": {"roles.delete"},
			},
			OwnerOnlyActions: []string{},
			AdminOverride:    false,
			MinHierarchy:     80, // Admin level required
		},
		"permissions": {
			Name: "permissions",
			RequiredPermissions: map[string][]string{
				"list":   {"permissions.list"},
				"read":   {"permissions.read"},
				"assign": {"permissions.assign"},
			},
			OwnerOnlyActions: []string{},
			AdminOverride:    false,
			MinHierarchy:     80, // Admin level required
		},
	}
}

// IsIPAllowed checks if an IP address is allowed based on whitelist/blacklist
func (rp *RouteProtectionConfig) IsIPAllowed(ip string) bool {
	// If whitelist is defined, IP must be in whitelist
	if len(rp.IPWhitelist) > 0 {
		return contains(rp.IPWhitelist, ip)
	}

	// Check if IP is in blacklist
	if len(rp.IPBlacklist) > 0 {
		return !contains(rp.IPBlacklist, ip)
	}

	// If no restrictions, allow
	return true
}

// IsCountryAllowed checks if a country is allowed
func (rp *RouteProtectionConfig) IsCountryAllowed(country string) bool {
	// If allowed countries list is defined, country must be in list
	if len(rp.AllowedCountries) > 0 {
		return contains(rp.AllowedCountries, country)
	}

	// Check if country is in blocklist
	if len(rp.BlockedCountries) > 0 {
		return !contains(rp.BlockedCountries, country)
	}

	// If no restrictions, allow
	return true
}

// ShouldFilterField checks if a field should be filtered from logs
func (al *AccessLoggingConfig) ShouldFilterField(fieldName string) bool {
	if !al.FilterSensitiveData {
		return false
	}

	// Check headers
	for _, header := range al.SensitiveHeaders {
		if matchPattern(header, fieldName) {
			return true
		}
	}

	// Check params
	for _, param := range al.SensitiveParams {
		if matchPattern(param, fieldName) {
			return true
		}
	}

	return false
}

// ShouldAuditResource checks if access to a resource should be audited
func (al *AccessLoggingConfig) ShouldAuditResource(resourceName string) bool {
	if !al.EnableAuditTrail {
		return false
	}

	if audit, exists := al.AuditByResource[resourceName]; exists {
		return audit
	}

	return false
}

// ShouldAuditOperation checks if an operation should be audited
func (al *AccessLoggingConfig) ShouldAuditOperation(operation string) bool {
	if !al.EnableAuditTrail {
		return false
	}

	return contains(al.AuditSensitiveOps, operation)
}

// GetResourceRule returns the access rule for a resource
func (ra *ResourceAccessConfig) GetResourceRule(resourceName string) (ResourceRule, bool) {
	rule, exists := ra.Resources[resourceName]
	return rule, exists
}

// IsSensitiveResource checks if a resource is marked as sensitive
func (ra *ResourceAccessConfig) IsSensitiveResource(resourceName string) bool {
	return contains(ra.SensitiveResources, resourceName)
}

// Helper functions

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// matchPattern checks if a field name matches a pattern (case-insensitive)
func matchPattern(pattern, fieldName string) bool {
	// Simple case-insensitive contains check
	// Could be enhanced with regex if needed
	matched, _ := regexp.MatchString("(?i)"+pattern, fieldName)
	return matched
}

// getEnvAsBool helper function (add to config.go if not exists)
func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	return valueStr == "true" || valueStr == "1" || valueStr == "yes"
}

// getEnvAsInt64 helper function
func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value := int64(getEnvAsInt(key, int(defaultValue)))
	return value
}

// getEnvAsSlice helper function (parses comma-separated values)
func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}

	// Split by comma and trim spaces
	var result []string
	for _, item := range regexp.MustCompile(`\s*,\s*`).Split(valueStr, -1) {
		if item != "" {
			result = append(result, item)
		}
	}

	if len(result) == 0 {
		return defaultValue
	}

	return result
}
