# Error Tracking

Error monitoring and alerting for ServicePro.

## Overview

Error tracking captures and reports:

- Application exceptions
- API errors
- Frontend JavaScript errors
- Validation failures

## Backend Error Handling

```go
// Centralized error handling middleware
func ErrorMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()

        if len(c.Errors) > 0 {
            err := c.Errors.Last()
            // Log error with context
            logger.Error("request error",
                "error", err.Error(),
                "path", c.Request.URL.Path,
                "user_id", c.GetString("user_id"),
            )
        }
    }
}
```

## Error Response Format

```json
{
  "error": "error_code",
  "message": "Human-readable error message",
  "details": {
    "field": "Additional context"
  },
  "request_id": "unique-request-id"
}
```

## Error Codes

| Code             | HTTP Status | Description               |
| ---------------- | ----------- | ------------------------- |
| validation_error | 400         | Request validation failed |
| unauthorized     | 401         | Authentication required   |
| forbidden        | 403         | Insufficient permissions  |
| not_found        | 404         | Resource not found        |
| conflict         | 409         | Resource conflict         |
| rate_limit       | 429         | Too many requests         |
| internal_error   | 500         | Server error              |

## Frontend Error Boundary

```tsx
import { ErrorBoundary } from '../components/ErrorBoundary';

<ErrorBoundary
  fallback={<ErrorPage />}
  onError={(error, info) => {
    // Log to error tracking service
    errorService.capture(error, info);
  }}
>
  <App />
</ErrorBoundary>;
```

## Logging

### Log Levels

| Level | Usage                      |
| ----- | -------------------------- |
| DEBUG | Development debugging      |
| INFO  | Normal operation events    |
| WARN  | Potential issues           |
| ERROR | Errors requiring attention |
| FATAL | Critical failures          |

### Structured Logging

```go
logger.Error("failed to create customer",
    "error", err,
    "customer_email", email,
    "user_id", userID,
)
```

## Configuration

```bash
LOG_LEVEL=info
LOG_FORMAT=json
ERROR_REPORTING_ENABLED=true
```
