# Health Checks

System health monitoring and availability tracking.

## Overview

The health check system provides:

- Component health monitoring
- Kubernetes probe endpoints
- Automated alerting
- Availability metrics

## Endpoints

| Endpoint         | Purpose                    |
| ---------------- | -------------------------- |
| `/health`        | Full health report         |
| `/healthz`       | Kubernetes liveness probe  |
| `/readyz`        | Kubernetes readiness probe |
| `/health/uptime` | Availability statistics    |

## Health Response

```json
{
  "status": "healthy",
  "version": "1.0.0",
  "environment": "production",
  "uptime_seconds": 86400,
  "checks": {
    "database": {
      "status": "healthy",
      "duration_ms": 2,
      "details": {
        "open_connections": 10,
        "in_use": 3
      }
    },
    "redis": {
      "status": "healthy",
      "duration_ms": 1
    }
  }
}
```

## Status Types

| Status    | Description                 |
| --------- | --------------------------- |
| healthy   | All checks passing          |
| degraded  | Non-critical checks failing |
| unhealthy | Critical checks failing     |

## Built-in Checks

- **Database** - PostgreSQL connection and pool stats
- **Redis** - Redis connection and pool stats
- **Memory** - Go runtime memory usage
- **HTTP** - External HTTP endpoint availability
- **TCP** - TCP connectivity checks

## Configuration

```bash
# Enable checks
HEALTH_CHECK_DATABASE=true
HEALTH_CHECK_REDIS=true
HEALTH_CHECK_MEMORY=true

# Intervals
HEALTH_INTERVAL_DEFAULT=30s
HEALTH_INTERVAL_DATABASE=15s

# Notifications
HEALTH_NOTIFICATIONS_ENABLED=true
HEALTH_NOTIFICATIONS_MIN_INTERVAL=5m
```

## Alerting

Supports multiple notification channels:

- Slack
- PagerDuty
- Webhook
- Email

## Kubernetes Integration

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 15

readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
```
