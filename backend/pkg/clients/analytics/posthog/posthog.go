package posthog

import (
	"context"

	ph "github.com/posthog/posthog-go"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/analytics"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

func init() {
	analytics.RegisterProvider(analytics.ProviderPostHog, func(ctx context.Context, cfg *config.Config) (analytics.Client, error) {
		return NewClient(cfg)
	})
}

// Client implements analytics.Client using PostHog
type Client struct {
	client ph.Client
}

// NewClient creates a new PostHog analytics client
func NewClient(cfg *config.Config) (*Client, error) {
	host := cfg.Analytics.PostHogHost
	if host == "" {
		host = "https://us.i.posthog.com"
	}

	client, err := ph.NewWithConfig(cfg.Analytics.PostHogAPIKey, ph.Config{
		Endpoint: host,
	})
	if err != nil {
		return nil, err
	}

	return &Client{client: client}, nil
}

// CaptureEvent implements analytics.Client
func (c *Client) CaptureEvent(ctx context.Context, event *analytics.Event) error {
	props := ph.NewProperties()
	for k, v := range event.Properties {
		props.Set(k, v)
	}

	return c.client.Enqueue(ph.Capture{
		DistinctId: event.DistinctID,
		Event:      event.Name,
		Properties: props,
	})
}

// IdentifyUser implements analytics.Client
func (c *Client) IdentifyUser(ctx context.Context, userID string, properties map[string]any) error {
	props := ph.NewProperties()
	for k, v := range properties {
		props.Set(k, v)
	}

	return c.client.Enqueue(ph.Identify{
		DistinctId: userID,
		Properties: props,
	})
}

// Close implements analytics.Client
func (c *Client) Close() error {
	logging.Info(context.Background(), "[ANALYTICS] Closing PostHog client", nil)
	return c.client.Close()
}

// Ensure Client implements analytics.Client
var _ analytics.Client = (*Client)(nil)
