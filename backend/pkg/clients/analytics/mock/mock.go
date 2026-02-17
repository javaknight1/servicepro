package mock

import (
	"context"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/analytics"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

func init() {
	analytics.RegisterProvider(analytics.ProviderMock, func(ctx context.Context, cfg *config.Config) (analytics.Client, error) {
		return NewClient(cfg.Analytics.Debug), nil
	})
}

// Client implements analytics.Client for development/testing
type Client struct {
	debug bool
}

// NewClient creates a new mock analytics client
func NewClient(debug bool) *Client {
	return &Client{debug: debug}
}

// CaptureEvent implements analytics.Client
func (c *Client) CaptureEvent(ctx context.Context, event *analytics.Event) error {
	if c.debug {
		logging.Info(ctx, "[ANALYTICS-MOCK] Event captured", map[string]any{
			"event":       event.Name,
			"distinct_id": event.DistinctID,
			"properties":  event.Properties,
		})
	}
	return nil
}

// IdentifyUser implements analytics.Client
func (c *Client) IdentifyUser(ctx context.Context, userID string, properties map[string]any) error {
	if c.debug {
		logging.Info(ctx, "[ANALYTICS-MOCK] User identified", map[string]any{
			"user_id":    userID,
			"properties": properties,
		})
	}
	return nil
}

// Close implements analytics.Client
func (c *Client) Close() error {
	return nil
}

// Ensure Client implements analytics.Client
var _ analytics.Client = (*Client)(nil)
