package analytics

// Provider represents an analytics service provider
type Provider string

const (
	ProviderPostHog Provider = "posthog"
	ProviderMock    Provider = "mock"
)

// Event represents an analytics event to be tracked
type Event struct {
	// Name is the event name (e.g., "payment_received")
	Name string

	// DistinctID is the user or entity identifier
	DistinctID string

	// Properties is additional event data
	Properties map[string]any
}
