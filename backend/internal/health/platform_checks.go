package health

import (
	"context"

	"github.com/javaknight1/servicepro/backend/pkg/database"
)

// Redis modes reported by the "redis" check.
const (
	RedisModeConnected        = database.RedisModeConnected        // real Redis reachable at boot
	RedisModeInMemoryFallback = database.RedisModeInMemoryFallback // configured but unreachable at boot; using in-process store
	RedisModeNotConfigured    = database.RedisModeNotConfigured    // REDIS_URL unset; using in-process store
)

// SchemaStateProvider reports the migration state of the database
// (implemented by *database.SchemaMonitor).
type SchemaStateProvider interface {
	State() database.SchemaState
}

// SchemaCheck reports whether the database schema matches this build.
//
//	ok / ahead    -> healthy (ahead is normal mid expand/contract)
//	behind / dirty -> unhealthy (critical: /health/ready returns 503)
//	unknown        -> degraded (the database check reports connectivity)
func SchemaCheck(provider SchemaStateProvider) Check {
	return func(ctx context.Context) CheckResult {
		state := provider.State()
		result := CheckResult{
			Name:    "schema",
			Message: state.Status,
			Details: map[string]interface{}{
				"state":    state.Status,
				"applied":  state.Applied,
				"expected": state.Expected,
			},
		}
		switch state.Status {
		case database.SchemaOK, database.SchemaAhead:
			result.Status = StatusHealthy
		case database.SchemaBehind, database.SchemaDirty:
			result.Status = StatusUnhealthy
		default:
			result.Status = StatusDegraded
			result.Error = state.Error
		}
		return result
	}
}

// RedisFallbackCheck reports Redis when no shared Redis is in use. It never
// fails readiness: the app runs on an in-process store in that case.
func RedisFallbackCheck(mode string) Check {
	return func(ctx context.Context) CheckResult {
		result := CheckResult{
			Name:    "redis",
			Message: mode,
			Details: map[string]interface{}{"state": mode},
		}
		if mode == RedisModeNotConfigured {
			result.Status = StatusHealthy
		} else {
			// Configured but unreachable at boot: still serving, but state
			// that should be shared (rate limits, tokens) is per-instance.
			result.Status = StatusDegraded
		}
		return result
	}
}
