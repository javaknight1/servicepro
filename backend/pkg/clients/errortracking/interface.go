package errortracking

import (
	"context"
	"net/http"
)

// Client defines the interface for error tracking operations.
// Implementations can send errors to various backends like Sentry.
type Client interface {
	// CaptureException captures an error
	CaptureException(ctx context.Context, err error) EventID

	// CaptureMessage captures a message
	CaptureMessage(ctx context.Context, message string) EventID

	// CaptureEvent captures a full event
	CaptureEvent(ctx context.Context, event *Event) EventID

	// AddBreadcrumb adds a breadcrumb to the current scope
	AddBreadcrumb(ctx context.Context, breadcrumb *Breadcrumb)

	// ConfigureScope allows modifying the current scope
	ConfigureScope(ctx context.Context, f func(Scope))

	// WithScope creates a new scope for the duration of the function
	WithScope(ctx context.Context, f func(context.Context))

	// SetUser sets the user for the current scope
	SetUser(ctx context.Context, user *User)

	// SetTag sets a tag for the current scope
	SetTag(ctx context.Context, key, value string)

	// SetExtra sets extra context for the current scope
	SetExtra(ctx context.Context, key string, value any)

	// HTTPMiddleware returns middleware for capturing panics and enriching context
	HTTPMiddleware(next http.Handler) http.Handler

	// Recover captures a panic and re-panics (for use with defer)
	Recover(ctx context.Context)

	// RecoverWithContext captures a panic with additional context
	RecoverWithContext(ctx context.Context, extra map[string]any)

	// Flush waits for all events to be sent
	Flush() error

	// Close releases resources and flushes events
	Close() error
}

// ContextKey is the type for context keys used by the errortracking package
type ContextKey string

const (
	// ContextKeyScope is the context key for the current scope
	ContextKeyScope ContextKey = "errortracking_scope"

	// ContextKeyHub is the context key for the hub (Sentry-specific concept)
	ContextKeyHub ContextKey = "errortracking_hub"
)
