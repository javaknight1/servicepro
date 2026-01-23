package mock

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/errortracking"
)

func init() {
	errortracking.RegisterProvider(errortracking.ProviderMock, func(ctx context.Context, cfg *config.Config) (errortracking.Client, error) {
		mockCfg := &Config{
			PrintToStdout: true,
			StoreEvents:   false,
		}
		return NewClient(mockCfg, cfg), nil
	})
}

// Config holds configuration for the mock error tracking client
type Config struct {
	// PrintToStdout prints events to stdout (default: true)
	PrintToStdout bool

	// StoreEvents stores events in memory for testing (default: false)
	StoreEvents bool
}

// Client implements the errortracking.Client interface for testing and development
type Client struct {
	config      *Config
	appConfig   *config.Config
	mu          sync.RWMutex
	events      []*errortracking.Event
	breadcrumbs []*errortracking.Breadcrumb
	tags        map[string]string
	extras      map[string]any
	user        *errortracking.User
}

// NewClient creates a new mock error tracking client
func NewClient(cfg *Config, appCfg *config.Config) *Client {
	if cfg == nil {
		cfg = &Config{
			PrintToStdout: true,
			StoreEvents:   false,
		}
	}

	c := &Client{
		config:      cfg,
		appConfig:   appCfg,
		events:      make([]*errortracking.Event, 0),
		breadcrumbs: make([]*errortracking.Breadcrumb, 0),
		tags:        make(map[string]string),
		extras:      make(map[string]any),
	}

	return c
}

// CaptureException implements errortracking.Client
func (c *Client) CaptureException(ctx context.Context, err error) errortracking.EventID {
	event := &errortracking.Event{
		Error:     err,
		Level:     errortracking.LevelError,
		Timestamp: time.Now(),
		Tags:      c.cloneTags(),
		Extra:     c.cloneExtras(),
		User:      c.user,
	}

	return c.captureEvent(event)
}

// CaptureMessage implements errortracking.Client
func (c *Client) CaptureMessage(ctx context.Context, message string) errortracking.EventID {
	event := &errortracking.Event{
		Message:   message,
		Level:     errortracking.LevelInfo,
		Timestamp: time.Now(),
		Tags:      c.cloneTags(),
		Extra:     c.cloneExtras(),
		User:      c.user,
	}

	return c.captureEvent(event)
}

// CaptureEvent implements errortracking.Client
func (c *Client) CaptureEvent(ctx context.Context, event *errortracking.Event) errortracking.EventID {
	// Merge in current scope
	if event.Tags == nil {
		event.Tags = c.cloneTags()
	} else {
		for k, v := range c.tags {
			if _, exists := event.Tags[k]; !exists {
				event.Tags[k] = v
			}
		}
	}

	if event.Extra == nil {
		event.Extra = c.cloneExtras()
	} else {
		for k, v := range c.extras {
			if _, exists := event.Extra[k]; !exists {
				event.Extra[k] = v
			}
		}
	}

	if event.User == nil {
		event.User = c.user
	}

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	return c.captureEvent(event)
}

func (c *Client) captureEvent(event *errortracking.Event) errortracking.EventID {
	eventID := generateEventID()

	if c.config.StoreEvents {
		c.mu.Lock()
		c.events = append(c.events, event)
		c.mu.Unlock()
	}

	if c.config.PrintToStdout {
		c.printEvent(event, eventID)
	}

	return eventID
}

// AddBreadcrumb implements errortracking.Client
func (c *Client) AddBreadcrumb(ctx context.Context, breadcrumb *errortracking.Breadcrumb) {
	if breadcrumb.Timestamp.IsZero() {
		breadcrumb.Timestamp = time.Now()
	}

	c.mu.Lock()
	c.breadcrumbs = append(c.breadcrumbs, breadcrumb)
	c.mu.Unlock()

	if c.config.PrintToStdout {
		log.Printf("[BREADCRUMB] %s: %s (category=%s, data=%v)",
			breadcrumb.Type, breadcrumb.Message, breadcrumb.Category, breadcrumb.Data)
	}
}

// ConfigureScope implements errortracking.Client
func (c *Client) ConfigureScope(ctx context.Context, f func(errortracking.Scope)) {
	scope := &mockScope{client: c}
	f(scope)
}

// WithScope implements errortracking.Client
func (c *Client) WithScope(ctx context.Context, f func(context.Context)) {
	// Create a new client with cloned state
	cloned := &Client{
		config:      c.config,
		appConfig:   c.appConfig,
		events:      c.events,
		breadcrumbs: make([]*errortracking.Breadcrumb, len(c.breadcrumbs)),
		tags:        c.cloneTags(),
		extras:      c.cloneExtras(),
		user:        c.user,
	}
	copy(cloned.breadcrumbs, c.breadcrumbs)

	// Run function with the scoped context
	f(context.WithValue(ctx, errortracking.ContextKeyScope, cloned))
}

// SetUser implements errortracking.Client
func (c *Client) SetUser(ctx context.Context, user *errortracking.User) {
	c.mu.Lock()
	c.user = user
	c.mu.Unlock()
}

// SetTag implements errortracking.Client
func (c *Client) SetTag(ctx context.Context, key, value string) {
	c.mu.Lock()
	c.tags[key] = value
	c.mu.Unlock()
}

// SetExtra implements errortracking.Client
func (c *Client) SetExtra(ctx context.Context, key string, value any) {
	c.mu.Lock()
	c.extras[key] = value
	c.mu.Unlock()
}

// HTTPMiddleware implements errortracking.Client
func (c *Client) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add request breadcrumb
		c.AddBreadcrumb(r.Context(), &errortracking.Breadcrumb{
			Type:     "http",
			Category: "http.request",
			Message:  fmt.Sprintf("%s %s", r.Method, r.URL.Path),
			Data: map[string]any{
				"method": r.Method,
				"url":    r.URL.String(),
			},
		})

		// Recover from panics
		defer func() {
			if rec := recover(); rec != nil {
				var err error
				switch v := rec.(type) {
				case error:
					err = v
				case string:
					err = fmt.Errorf("%s", v)
				default:
					err = fmt.Errorf("%v", v)
				}

				c.CaptureException(r.Context(), err)
				panic(rec) // Re-panic after capturing
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// Recover implements errortracking.Client
func (c *Client) Recover(ctx context.Context) {
	c.RecoverWithContext(ctx, nil)
}

// RecoverWithContext implements errortracking.Client
func (c *Client) RecoverWithContext(ctx context.Context, extra map[string]any) {
	if rec := recover(); rec != nil {
		var err error
		switch v := rec.(type) {
		case error:
			err = v
		case string:
			err = fmt.Errorf("%s", v)
		default:
			err = fmt.Errorf("%v", v)
		}

		event := &errortracking.Event{
			Error:     err,
			Level:     errortracking.LevelFatal,
			Timestamp: time.Now(),
			Tags:      c.cloneTags(),
			Extra:     c.cloneExtras(),
			User:      c.user,
		}

		// Add extra context
		for k, v := range extra {
			event.Extra[k] = v
		}

		c.captureEvent(event)
	}
}

// Flush implements errortracking.Client
func (c *Client) Flush() error {
	return nil
}

// Close implements errortracking.Client
func (c *Client) Close() error {
	return nil
}

func (c *Client) cloneTags() map[string]string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tags := make(map[string]string, len(c.tags))
	for k, v := range c.tags {
		tags[k] = v
	}
	return tags
}

func (c *Client) cloneExtras() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	extras := make(map[string]any, len(c.extras))
	for k, v := range c.extras {
		extras[k] = v
	}
	return extras
}

func (c *Client) printEvent(event *errortracking.Event, eventID errortracking.EventID) {
	var msg string
	if event.Error != nil {
		msg = event.Error.Error()
	} else {
		msg = event.Message
	}

	log.Printf("[ERROR_TRACKING] %s (event_id=%s, level=%s, tags=%v, extra=%v)",
		msg, eventID, event.Level, event.Tags, event.Extra)
}

// GetEvents returns all stored events (for testing)
func (c *Client) GetEvents() []*errortracking.Event {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]*errortracking.Event, len(c.events))
	copy(result, c.events)
	return result
}

// ClearEvents clears all stored events (for testing)
func (c *Client) ClearEvents() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = make([]*errortracking.Event, 0)
}

// mockScope implements errortracking.Scope
type mockScope struct {
	client *Client
}

func (s *mockScope) SetTag(key, value string) {
	s.client.mu.Lock()
	s.client.tags[key] = value
	s.client.mu.Unlock()
}

func (s *mockScope) SetTags(tags map[string]string) {
	s.client.mu.Lock()
	for k, v := range tags {
		s.client.tags[k] = v
	}
	s.client.mu.Unlock()
}

func (s *mockScope) RemoveTag(key string) {
	s.client.mu.Lock()
	delete(s.client.tags, key)
	s.client.mu.Unlock()
}

func (s *mockScope) SetExtra(key string, value any) {
	s.client.mu.Lock()
	s.client.extras[key] = value
	s.client.mu.Unlock()
}

func (s *mockScope) SetExtras(extras map[string]any) {
	s.client.mu.Lock()
	for k, v := range extras {
		s.client.extras[k] = v
	}
	s.client.mu.Unlock()
}

func (s *mockScope) RemoveExtra(key string) {
	s.client.mu.Lock()
	delete(s.client.extras, key)
	s.client.mu.Unlock()
}

func (s *mockScope) SetUser(user *errortracking.User) {
	s.client.mu.Lock()
	s.client.user = user
	s.client.mu.Unlock()
}

func (s *mockScope) SetRequest(request *errortracking.Request) {
	// Mock doesn't store request info
}

func (s *mockScope) SetFingerprint(fingerprint []string) {
	// Mock doesn't handle fingerprinting
}

func (s *mockScope) SetLevel(level errortracking.Level) {
	// Mock doesn't track default level
}

func (s *mockScope) SetTransaction(name string) {
	// Mock doesn't track transaction
}

func (s *mockScope) AddBreadcrumb(breadcrumb *errortracking.Breadcrumb) {
	s.client.mu.Lock()
	s.client.breadcrumbs = append(s.client.breadcrumbs, breadcrumb)
	s.client.mu.Unlock()
}

func (s *mockScope) ClearBreadcrumbs() {
	s.client.mu.Lock()
	s.client.breadcrumbs = make([]*errortracking.Breadcrumb, 0)
	s.client.mu.Unlock()
}

func (s *mockScope) Clear() {
	s.client.mu.Lock()
	s.client.tags = make(map[string]string)
	s.client.extras = make(map[string]any)
	s.client.user = nil
	s.client.breadcrumbs = make([]*errortracking.Breadcrumb, 0)
	s.client.mu.Unlock()
}

func (s *mockScope) Clone() errortracking.Scope {
	return &mockScope{
		client: &Client{
			config:      s.client.config,
			appConfig:   s.client.appConfig,
			events:      s.client.events,
			breadcrumbs: s.client.breadcrumbs,
			tags:        s.client.cloneTags(),
			extras:      s.client.cloneExtras(),
			user:        s.client.user,
		},
	}
}

// generateEventID generates a random event ID
func generateEventID() errortracking.EventID {
	b := make([]byte, 16)
	rand.Read(b)
	return errortracking.EventID(hex.EncodeToString(b))
}

// Ensure Client implements errortracking.Client
var _ errortracking.Client = (*Client)(nil)

// Ensure mockScope implements errortracking.Scope
var _ errortracking.Scope = (*mockScope)(nil)
