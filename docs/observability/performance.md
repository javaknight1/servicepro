# Performance Monitoring

API and application performance monitoring.

## Overview

Performance monitoring tracks:

- API response times
- Database query performance
- Memory and CPU usage
- Request throughput

## Metrics Collected

### API Metrics

| Metric              | Description                     |
| ------------------- | ------------------------------- |
| request_duration_ms | Request processing time         |
| request_count       | Total request count             |
| error_rate          | Percentage of 4xx/5xx responses |
| requests_per_second | Request throughput              |

### Database Metrics

| Metric               | Description                 |
| -------------------- | --------------------------- |
| query_duration_ms    | Query execution time        |
| connection_pool_size | Active connections          |
| connection_wait_time | Time waiting for connection |

### Application Metrics

| Metric            | Description                   |
| ----------------- | ----------------------------- |
| memory_heap_bytes | Heap memory usage             |
| goroutine_count   | Active goroutines             |
| gc_pause_ms       | Garbage collection pause time |

## Middleware

```go
// Request timing middleware
func TimingMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()
        c.Next()
        duration := time.Since(start)

        // Record metric
        metrics.RecordRequestDuration(c.Request.URL.Path, duration)
    }
}
```

## Database Query Monitoring

```go
// GORM callback for query timing
db.Callback().Query().After("gorm:query").Register("metrics", func(d *gorm.DB) {
    duration := d.Statement.Context.Value("query_start")
    metrics.RecordQueryDuration(duration)
})
```

## Performance Targets

| Endpoint Type   | Target p50 | Target p95 |
| --------------- | ---------- | ---------- |
| List endpoints  | <100ms     | <200ms     |
| Single resource | <50ms      | <100ms     |
| Create/Update   | <100ms     | <250ms     |
| Complex queries | <200ms     | <500ms     |

## Slow Query Detection

```bash
# Log slow queries
DB_SLOW_QUERY_THRESHOLD=100ms
DB_LOG_SLOW_QUERIES=true
```

## Frontend Performance

```typescript
// Track page load time
const observer = new PerformanceObserver((list) => {
  for (const entry of list.getEntries()) {
    analytics.track('page_performance', 'performance', {
      page: window.location.pathname,
      load_time: entry.loadEventEnd - entry.startTime,
    });
  }
});
observer.observe({ entryTypes: ['navigation'] });
```

## Configuration

```bash
# Performance monitoring
PERF_MONITORING_ENABLED=true
PERF_SAMPLE_RATE=0.1

# Slow query logging
DB_SLOW_QUERY_THRESHOLD=100ms

# Memory limits
MEMORY_WARNING_THRESHOLD=80
MEMORY_CRITICAL_THRESHOLD=90
```

## Optimization Tips

1. **Use indexes** - Add indexes for frequently queried columns
2. **Pagination** - Always paginate list endpoints
3. **Caching** - Cache frequently accessed data in Redis
4. **Connection pooling** - Configure appropriate pool sizes
5. **Async operations** - Use goroutines for non-blocking I/O
