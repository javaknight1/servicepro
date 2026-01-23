package sentry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/errortracking"
)

func init() {
	errortracking.RegisterProvider(errortracking.ProviderSentry, func(ctx context.Context, cfg *config.Config) (errortracking.Client, error) {
		sentryCfg := &Config{
			DSN:              cfg.Sentry.DSN,
			Environment:      cfg.Server.Env,
			Release:          cfg.Sentry.Release,
			TracesSampleRate: cfg.Sentry.TracesSampleRate,
			Debug:            cfg.Sentry.Debug,
		}
		return NewClient(sentryCfg, cfg)
	})
}

// Config holds configuration for the Sentry error tracking client
type Config struct {
	// DSN is the Sentry DSN (required)
	DSN string

	// Environment is the deployment environment
	Environment string

	// Release is the release version
	Release string

	// TracesSampleRate is the sample rate for tracing (0.0 to 1.0)
	TracesSampleRate float64

	// Debug enables debug mode
	Debug bool

	// AttachStacktrace attaches stack traces to all messages
	AttachStacktrace bool
}

// Client implements the errortracking.Client interface using Sentry
type Client struct {
	config    *Config
	appConfig *config.Config
}

// NewClient creates a new Sentry error tracking client
func NewClient(cfg *Config, appCfg *config.Config) (*Client, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("Sentry DSN is required")
	}

	sampleRate := cfg.TracesSampleRate
	if sampleRate <= 0 {
		sampleRate = 0.1 // Default 10% sampling
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		TracesSampleRate: sampleRate,
		Debug:            cfg.Debug,
		AttachStacktrace: cfg.AttachStacktrace,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Sentry: %w", err)
	}

	return &Client{
		config:    cfg,
		appConfig: appCfg,
	}, nil
}

// CaptureException implements errortracking.Client
func (c *Client) CaptureException(ctx context.Context, err error) errortracking.EventID {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	eventID := hub.CaptureException(err)
	if eventID == nil {
		return ""
	}
	return errortracking.EventID(*eventID)
}

// CaptureMessage implements errortracking.Client
func (c *Client) CaptureMessage(ctx context.Context, message string) errortracking.EventID {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	eventID := hub.CaptureMessage(message)
	if eventID == nil {
		return ""
	}
	return errortracking.EventID(*eventID)
}

// CaptureEvent implements errortracking.Client
func (c *Client) CaptureEvent(ctx context.Context, event *errortracking.Event) errortracking.EventID {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	sentryEvent := &sentry.Event{
		Message:     event.Message,
		Level:       toSentryLevel(event.Level),
		Timestamp:   event.Timestamp,
		Tags:        event.Tags,
		Extra:       event.Extra,
		Fingerprint: event.Fingerprint,
		Transaction: event.Transaction,
	}

	if event.Error != nil {
		sentryEvent.Exception = []sentry.Exception{
			{
				Type:  fmt.Sprintf("%T", event.Error),
				Value: event.Error.Error(),
			},
		}
	}

	if event.User != nil {
		sentryEvent.User = sentry.User{
			ID:        event.User.ID,
			Email:     event.User.Email,
			Username:  event.User.Username,
			IPAddress: event.User.IPAddress,
		}
	}

	if event.Request != nil {
		sentryEvent.Request = &sentry.Request{
			Method:  event.Request.Method,
			URL:     event.Request.URL,
			Data:    fmt.Sprintf("%v", event.Request.Data),
			Headers: event.Request.Headers,
			Env:     event.Request.Env,
		}
	}

	eventID := hub.CaptureEvent(sentryEvent)
	if eventID == nil {
		return ""
	}
	return errortracking.EventID(*eventID)
}

// AddBreadcrumb implements errortracking.Client
func (c *Client) AddBreadcrumb(ctx context.Context, breadcrumb *errortracking.Breadcrumb) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	hub.AddBreadcrumb(&sentry.Breadcrumb{
		Type:      breadcrumb.Type,
		Category:  breadcrumb.Category,
		Message:   breadcrumb.Message,
		Data:      breadcrumb.Data,
		Level:     toSentryLevel(breadcrumb.Level),
		Timestamp: breadcrumb.Timestamp,
	}, nil)
}

// ConfigureScope implements errortracking.Client
func (c *Client) ConfigureScope(ctx context.Context, f func(errortracking.Scope)) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	hub.ConfigureScope(func(sentryScope *sentry.Scope) {
		f(&sentryScoreWrapper{scope: sentryScope})
	})
}

// WithScope implements errortracking.Client
func (c *Client) WithScope(ctx context.Context, f func(context.Context)) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
	} else {
		hub = hub.Clone()
	}

	hub.WithScope(func(scope *sentry.Scope) {
		newCtx := sentry.SetHubOnContext(ctx, hub)
		f(newCtx)
	})
}

// SetUser implements errortracking.Client
func (c *Client) SetUser(ctx context.Context, user *errortracking.User) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{
			ID:        user.ID,
			Email:     user.Email,
			Username:  user.Username,
			IPAddress: user.IPAddress,
		})
	})
}

// SetTag implements errortracking.Client
func (c *Client) SetTag(ctx context.Context, key, value string) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag(key, value)
	})
}

// SetExtra implements errortracking.Client
func (c *Client) SetExtra(ctx context.Context, key string, value any) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetExtra(key, value)
	})
}

// HTTPMiddleware implements errortracking.Client
func (c *Client) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hub := sentry.CurrentHub().Clone()
		ctx := sentry.SetHubOnContext(r.Context(), hub)

		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetRequest(r)
		})

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

				hub.RecoverWithContext(ctx, err)
				panic(rec) // Re-panic after capturing
			}
		}()

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Recover implements errortracking.Client
func (c *Client) Recover(ctx context.Context) {
	c.RecoverWithContext(ctx, nil)
}

// RecoverWithContext implements errortracking.Client
func (c *Client) RecoverWithContext(ctx context.Context, extra map[string]any) {
	if rec := recover(); rec != nil {
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			hub = sentry.CurrentHub()
		}

		if extra != nil {
			hub.ConfigureScope(func(scope *sentry.Scope) {
				for k, v := range extra {
					scope.SetExtra(k, v)
				}
			})
		}

		hub.RecoverWithContext(ctx, rec)
	}
}

// Flush implements errortracking.Client
func (c *Client) Flush() error {
	sentry.Flush(5 * time.Second)
	return nil
}

// Close implements errortracking.Client
func (c *Client) Close() error {
	sentry.Flush(5 * time.Second)
	return nil
}

// sentryScoreWrapper wraps a Sentry scope to implement errortracking.Scope
type sentryScoreWrapper struct {
	scope *sentry.Scope
}

func (s *sentryScoreWrapper) SetTag(key, value string) {
	s.scope.SetTag(key, value)
}

func (s *sentryScoreWrapper) SetTags(tags map[string]string) {
	s.scope.SetTags(tags)
}

func (s *sentryScoreWrapper) RemoveTag(key string) {
	s.scope.RemoveTag(key)
}

func (s *sentryScoreWrapper) SetExtra(key string, value any) {
	s.scope.SetExtra(key, value)
}

func (s *sentryScoreWrapper) SetExtras(extras map[string]any) {
	s.scope.SetExtras(extras)
}

func (s *sentryScoreWrapper) RemoveExtra(key string) {
	s.scope.RemoveExtra(key)
}

func (s *sentryScoreWrapper) SetUser(user *errortracking.User) {
	s.scope.SetUser(sentry.User{
		ID:        user.ID,
		Email:     user.Email,
		Username:  user.Username,
		IPAddress: user.IPAddress,
	})
}

func (s *sentryScoreWrapper) SetRequest(request *errortracking.Request) {
	// Sentry scope doesn't have SetRequest with our type
	// We'd need to convert to http.Request
}

func (s *sentryScoreWrapper) SetFingerprint(fingerprint []string) {
	s.scope.SetFingerprint(fingerprint)
}

func (s *sentryScoreWrapper) SetLevel(level errortracking.Level) {
	s.scope.SetLevel(toSentryLevel(level))
}

func (s *sentryScoreWrapper) SetTransaction(name string) {
	// Sentry scope doesn't have SetTransaction - transactions are set via spans
	s.scope.SetTag("transaction", name)
}

func (s *sentryScoreWrapper) AddBreadcrumb(breadcrumb *errortracking.Breadcrumb) {
	s.scope.AddBreadcrumb(&sentry.Breadcrumb{
		Type:      breadcrumb.Type,
		Category:  breadcrumb.Category,
		Message:   breadcrumb.Message,
		Data:      breadcrumb.Data,
		Level:     toSentryLevel(breadcrumb.Level),
		Timestamp: breadcrumb.Timestamp,
	}, 100)
}

func (s *sentryScoreWrapper) ClearBreadcrumbs() {
	s.scope.ClearBreadcrumbs()
}

func (s *sentryScoreWrapper) Clear() {
	s.scope.Clear()
}

func (s *sentryScoreWrapper) Clone() errortracking.Scope {
	return &sentryScoreWrapper{scope: s.scope.Clone()}
}

// toSentryLevel converts our Level to Sentry's Level
func toSentryLevel(level errortracking.Level) sentry.Level {
	switch level {
	case errortracking.LevelDebug:
		return sentry.LevelDebug
	case errortracking.LevelInfo:
		return sentry.LevelInfo
	case errortracking.LevelWarning:
		return sentry.LevelWarning
	case errortracking.LevelError:
		return sentry.LevelError
	case errortracking.LevelFatal:
		return sentry.LevelFatal
	default:
		return sentry.LevelInfo
	}
}

// Ensure Client implements errortracking.Client
var _ errortracking.Client = (*Client)(nil)

// Ensure sentryScoreWrapper implements errortracking.Scope
var _ errortracking.Scope = (*sentryScoreWrapper)(nil)
