package analytics

import "context"

// Client defines the interface for analytics operations.
// Implementations can send events to various backends like PostHog.
type Client interface {
	// CaptureEvent sends an analytics event
	CaptureEvent(ctx context.Context, event *Event) error

	// IdentifyUser sets user properties for analytics
	IdentifyUser(ctx context.Context, userID string, properties map[string]any) error

	// Close releases resources and flushes pending events
	Close() error
}
