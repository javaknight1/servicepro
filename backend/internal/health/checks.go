package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Status represents the health status of a component
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
	StatusUnknown   Status = "unknown"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Name        string                 `json:"name"`
	Status      Status                 `json:"status"`
	Message     string                 `json:"message,omitempty"`
	Duration    time.Duration          `json:"duration_ms"`
	Timestamp   time.Time              `json:"timestamp"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Error       string                 `json:"error,omitempty"`
	LastSuccess *time.Time             `json:"last_success,omitempty"`
	LastFailure *time.Time             `json:"last_failure,omitempty"`
}

// MarshalJSON customizes JSON serialization for CheckResult
func (r CheckResult) MarshalJSON() ([]byte, error) {
	type Alias CheckResult
	return json.Marshal(&struct {
		DurationMs int64 `json:"duration_ms"`
		*Alias
	}{
		DurationMs: r.Duration.Milliseconds(),
		Alias:      (*Alias)(&r),
	})
}

// Check is a function that performs a health check
type Check func(ctx context.Context) CheckResult

// CheckConfig holds configuration for a health check
type CheckConfig struct {
	Name     string
	Timeout  time.Duration
	Interval time.Duration
	Critical bool // If true, failure marks overall health as unhealthy
	Check    Check
}

// HealthReport represents the overall health status
type HealthReport struct {
	Status      Status                 `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Version     string                 `json:"version,omitempty"`
	Environment string                 `json:"environment,omitempty"`
	Uptime      time.Duration          `json:"uptime_seconds"`
	Checks      map[string]CheckResult `json:"checks"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// MarshalJSON customizes JSON serialization for HealthReport
func (r HealthReport) MarshalJSON() ([]byte, error) {
	type Alias HealthReport
	return json.Marshal(&struct {
		UptimeSeconds int64 `json:"uptime_seconds"`
		*Alias
	}{
		UptimeSeconds: int64(r.Uptime.Seconds()),
		Alias:         (*Alias)(&r),
	})
}

// Checker manages health checks
type Checker struct {
	mu          sync.RWMutex
	checks      map[string]CheckConfig
	results     map[string]CheckResult
	startTime   time.Time
	version     string
	environment string
	callbacks   []StatusCallback
}

// StatusCallback is called when health status changes
type StatusCallback func(name string, oldStatus, newStatus Status)

// NewChecker creates a new health checker
func NewChecker(version, environment string) *Checker {
	return &Checker{
		checks:      make(map[string]CheckConfig),
		results:     make(map[string]CheckResult),
		startTime:   time.Now(),
		version:     version,
		environment: environment,
	}
}

// Register registers a new health check
func (c *Checker) Register(config CheckConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.Interval == 0 {
		config.Interval = 30 * time.Second
	}

	c.checks[config.Name] = config
	c.results[config.Name] = CheckResult{
		Name:      config.Name,
		Status:    StatusUnknown,
		Timestamp: time.Now(),
	}
}

// Unregister removes a health check
func (c *Checker) Unregister(name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.checks, name)
	delete(c.results, name)
}

// OnStatusChange registers a callback for status changes
func (c *Checker) OnStatusChange(callback StatusCallback) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.callbacks = append(c.callbacks, callback)
}

// RunCheck runs a specific health check
func (c *Checker) RunCheck(ctx context.Context, name string) CheckResult {
	c.mu.RLock()
	config, exists := c.checks[name]
	c.mu.RUnlock()

	if !exists {
		return CheckResult{
			Name:      name,
			Status:    StatusUnknown,
			Message:   "Check not found",
			Timestamp: time.Now(),
		}
	}

	// Create timeout context
	checkCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	start := time.Now()
	result := config.Check(checkCtx)
	result.Name = name
	result.Duration = time.Since(start)
	result.Timestamp = time.Now()

	// Update last success/failure times
	c.mu.Lock()
	oldResult := c.results[name]
	if result.Status == StatusHealthy {
		now := time.Now()
		result.LastSuccess = &now
		result.LastFailure = oldResult.LastFailure
	} else {
		now := time.Now()
		result.LastFailure = &now
		result.LastSuccess = oldResult.LastSuccess
	}

	// Notify callbacks if status changed
	if oldResult.Status != result.Status {
		for _, callback := range c.callbacks {
			go callback(name, oldResult.Status, result.Status)
		}
	}

	c.results[name] = result
	c.mu.Unlock()

	return result
}

// RunAllChecks runs all registered health checks
func (c *Checker) RunAllChecks(ctx context.Context) map[string]CheckResult {
	c.mu.RLock()
	checkNames := make([]string, 0, len(c.checks))
	for name := range c.checks {
		checkNames = append(checkNames, name)
	}
	c.mu.RUnlock()

	results := make(map[string]CheckResult)
	var wg sync.WaitGroup

	resultChan := make(chan CheckResult, len(checkNames))

	for _, name := range checkNames {
		wg.Add(1)
		go func(checkName string) {
			defer wg.Done()
			result := c.RunCheck(ctx, checkName)
			resultChan <- result
		}(name)
	}

	wg.Wait()
	close(resultChan)

	for result := range resultChan {
		results[result.Name] = result
	}

	return results
}

// GetReport generates a health report
func (c *Checker) GetReport(ctx context.Context) HealthReport {
	results := c.RunAllChecks(ctx)

	// Determine overall status
	overallStatus := StatusHealthy
	for name, result := range results {
		c.mu.RLock()
		config := c.checks[name]
		c.mu.RUnlock()

		if result.Status == StatusUnhealthy && config.Critical {
			overallStatus = StatusUnhealthy
			break
		}
		if result.Status == StatusDegraded || (result.Status == StatusUnhealthy && !config.Critical) {
			if overallStatus == StatusHealthy {
				overallStatus = StatusDegraded
			}
		}
	}

	return HealthReport{
		Status:      overallStatus,
		Timestamp:   time.Now(),
		Version:     c.version,
		Environment: c.environment,
		Uptime:      time.Since(c.startTime),
		Checks:      results,
	}
}

// GetCachedResults returns the most recent cached results
func (c *Checker) GetCachedResults() map[string]CheckResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	results := make(map[string]CheckResult, len(c.results))
	for k, v := range c.results {
		results[k] = v
	}
	return results
}

// IsHealthy returns true if all critical checks are healthy
func (c *Checker) IsHealthy(ctx context.Context) bool {
	report := c.GetReport(ctx)
	return report.Status == StatusHealthy
}

// IsReady returns true if the service is ready to accept traffic
func (c *Checker) IsReady(ctx context.Context) bool {
	report := c.GetReport(ctx)
	return report.Status != StatusUnhealthy
}

// Standard health checks

// DatabaseCheck creates a database connectivity check
func DatabaseCheck(db *sql.DB) Check {
	return func(ctx context.Context) CheckResult {
		result := CheckResult{
			Name:   "database",
			Status: StatusHealthy,
		}

		err := db.PingContext(ctx)
		if err != nil {
			result.Status = StatusUnhealthy
			result.Error = err.Error()
			result.Message = "Database connection failed"
			return result
		}

		// Get connection pool stats
		stats := db.Stats()
		result.Details = map[string]interface{}{
			"open_connections":    stats.OpenConnections,
			"in_use":              stats.InUse,
			"idle":                stats.Idle,
			"max_open":            stats.MaxOpenConnections,
			"wait_count":          stats.WaitCount,
			"wait_duration_ms":    stats.WaitDuration.Milliseconds(),
			"max_idle_closed":     stats.MaxIdleClosed,
			"max_lifetime_closed": stats.MaxLifetimeClosed,
		}

		// Check for connection pool exhaustion
		if stats.MaxOpenConnections > 0 {
			utilizationPct := float64(stats.InUse) / float64(stats.MaxOpenConnections) * 100
			result.Details["utilization_pct"] = utilizationPct

			if utilizationPct > 90 {
				result.Status = StatusDegraded
				result.Message = fmt.Sprintf("Connection pool utilization high: %.1f%%", utilizationPct)
			}
		}

		if result.Status == StatusHealthy {
			result.Message = "Database connection healthy"
		}

		return result
	}
}

// RedisCheck creates a Redis connectivity check
func RedisCheck(client *redis.Client) Check {
	return func(ctx context.Context) CheckResult {
		result := CheckResult{
			Name:   "redis",
			Status: StatusHealthy,
		}

		// Ping Redis
		pong, err := client.Ping(ctx).Result()
		if err != nil {
			result.Status = StatusUnhealthy
			result.Error = err.Error()
			result.Message = "Redis connection failed"
			return result
		}

		result.Details = map[string]interface{}{
			"response": pong,
		}

		// Get pool stats
		stats := client.PoolStats()
		result.Details["pool_hits"] = stats.Hits
		result.Details["pool_misses"] = stats.Misses
		result.Details["pool_timeouts"] = stats.Timeouts
		result.Details["total_conns"] = stats.TotalConns
		result.Details["idle_conns"] = stats.IdleConns
		result.Details["stale_conns"] = stats.StaleConns

		result.Message = "Redis connection healthy"
		return result
	}
}

// HTTPCheck creates an HTTP endpoint check
func HTTPCheck(name, url string, expectedStatus int) Check {
	return func(ctx context.Context) CheckResult {
		result := CheckResult{
			Name:   name,
			Status: StatusHealthy,
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			result.Status = StatusUnhealthy
			result.Error = err.Error()
			result.Message = "Failed to create request"
			return result
		}

		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			result.Status = StatusUnhealthy
			result.Error = err.Error()
			result.Message = "HTTP request failed"
			return result
		}
		defer resp.Body.Close()

		result.Details = map[string]interface{}{
			"url":         url,
			"status_code": resp.StatusCode,
		}

		if resp.StatusCode != expectedStatus {
			result.Status = StatusUnhealthy
			result.Message = fmt.Sprintf("Expected status %d, got %d", expectedStatus, resp.StatusCode)
			return result
		}

		result.Message = "HTTP endpoint healthy"
		return result
	}
}

// TCPCheck creates a TCP connectivity check
func TCPCheck(name, host string, port int) Check {
	return func(ctx context.Context) CheckResult {
		result := CheckResult{
			Name:   name,
			Status: StatusHealthy,
		}

		address := fmt.Sprintf("%s:%d", host, port)

		var d net.Dialer
		conn, err := d.DialContext(ctx, "tcp", address)
		if err != nil {
			result.Status = StatusUnhealthy
			result.Error = err.Error()
			result.Message = fmt.Sprintf("TCP connection to %s failed", address)
			return result
		}
		defer conn.Close()

		result.Details = map[string]interface{}{
			"address": address,
		}
		result.Message = fmt.Sprintf("TCP connection to %s successful", address)

		return result
	}
}

// DiskSpaceCheck creates a disk space check
func DiskSpaceCheck(path string, warningThreshold, criticalThreshold float64) Check {
	return func(ctx context.Context) CheckResult {
		result := CheckResult{
			Name:   "disk_space",
			Status: StatusHealthy,
		}

		// Note: This is a simplified version. In production, you'd use
		// syscall.Statfs on Unix or similar for Windows
		result.Details = map[string]interface{}{
			"path":               path,
			"warning_threshold":  warningThreshold,
			"critical_threshold": criticalThreshold,
		}

		// Placeholder - actual implementation would check disk space
		result.Message = "Disk space check placeholder"

		return result
	}
}

// MemoryCheck creates a memory usage check
func MemoryCheck(warningThreshold, criticalThreshold float64) Check {
	return func(ctx context.Context) CheckResult {
		result := CheckResult{
			Name:   "memory",
			Status: StatusHealthy,
		}

		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		allocMB := float64(m.Alloc) / 1024 / 1024
		sysMB := float64(m.Sys) / 1024 / 1024
		heapAllocMB := float64(m.HeapAlloc) / 1024 / 1024
		heapSysMB := float64(m.HeapSys) / 1024 / 1024

		result.Details = map[string]interface{}{
			"alloc_mb":      allocMB,
			"sys_mb":        sysMB,
			"heap_alloc_mb": heapAllocMB,
			"heap_sys_mb":   heapSysMB,
			"heap_objects":  m.HeapObjects,
			"gc_cycles":     m.NumGC,
			"goroutines":    runtime.NumGoroutine(),
		}

		// Check thresholds based on heap utilization
		if heapSysMB > 0 {
			utilizationPct := heapAllocMB / heapSysMB * 100
			result.Details["utilization_pct"] = utilizationPct

			if utilizationPct >= criticalThreshold {
				result.Status = StatusUnhealthy
				result.Message = fmt.Sprintf("Memory utilization critical: %.1f%%", utilizationPct)
			} else if utilizationPct >= warningThreshold {
				result.Status = StatusDegraded
				result.Message = fmt.Sprintf("Memory utilization warning: %.1f%%", utilizationPct)
			}
		}

		if result.Status == StatusHealthy {
			result.Message = "Memory usage healthy"
		}

		return result
	}
}

// CustomCheck creates a custom health check from a function
func CustomCheck(name string, checkFn func(ctx context.Context) error) Check {
	return func(ctx context.Context) CheckResult {
		result := CheckResult{
			Name:   name,
			Status: StatusHealthy,
		}

		if err := checkFn(ctx); err != nil {
			result.Status = StatusUnhealthy
			result.Error = err.Error()
			result.Message = "Check failed"
			return result
		}

		result.Message = "Check passed"
		return result
	}
}
