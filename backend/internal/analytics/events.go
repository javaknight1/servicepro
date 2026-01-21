package analytics

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventCategory represents the category of an analytics event
type EventCategory string

const (
	CategoryUser        EventCategory = "user"
	CategoryPage        EventCategory = "page"
	CategoryFeature     EventCategory = "feature"
	CategoryBusiness    EventCategory = "business"
	CategoryPerformance EventCategory = "performance"
	CategoryError       EventCategory = "error"
	CategoryAPI         EventCategory = "api"
)

// EventName represents standard event names
type EventName string

// User events
const (
	EventUserSignup       EventName = "user_signup"
	EventUserLogin        EventName = "user_login"
	EventUserLogout       EventName = "user_logout"
	EventUserProfileView  EventName = "user_profile_view"
	EventUserSettingsEdit EventName = "user_settings_edit"
)

// Page events
const (
	EventPageView   EventName = "page_view"
	EventPageScroll EventName = "page_scroll"
	EventPageExit   EventName = "page_exit"
)

// Feature events
const (
	EventFeatureUsed     EventName = "feature_used"
	EventButtonClick     EventName = "button_click"
	EventFormSubmit      EventName = "form_submit"
	EventSearchPerformed EventName = "search_performed"
	EventFilterApplied   EventName = "filter_applied"
	EventExportGenerated EventName = "export_generated"
)

// Business events
const (
	EventCustomerCreated EventName = "customer_created"
	EventCustomerUpdated EventName = "customer_updated"
	EventJobScheduled    EventName = "job_scheduled"
	EventJobCompleted    EventName = "job_completed"
	EventJobCancelled    EventName = "job_cancelled"
	EventQuoteSent       EventName = "quote_sent"
	EventQuoteAccepted   EventName = "quote_accepted"
	EventQuoteDeclined   EventName = "quote_declined"
	EventInvoiceCreated  EventName = "invoice_created"
	EventInvoiceSent     EventName = "invoice_sent"
	EventPaymentReceived EventName = "payment_received"
	EventPaymentFailed   EventName = "payment_failed"
)

// Performance events
const (
	EventAPILatency    EventName = "api_latency"
	EventPageLoadTime  EventName = "page_load_time"
	EventDatabaseQuery EventName = "database_query"
	EventCacheHit      EventName = "cache_hit"
	EventCacheMiss     EventName = "cache_miss"
)

// Error events
const (
	EventErrorOccurred EventName = "error_occurred"
	EventErrorResolved EventName = "error_resolved"
)

// Event represents an analytics event
type Event struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	Name        EventName              `json:"name" db:"name"`
	Category    EventCategory          `json:"category" db:"category"`
	UserID      *uuid.UUID             `json:"user_id,omitempty" db:"user_id"`
	SessionID   string                 `json:"session_id,omitempty" db:"session_id"`
	TenantID    *uuid.UUID             `json:"tenant_id,omitempty" db:"tenant_id"`
	Properties  map[string]interface{} `json:"properties,omitempty" db:"properties"`
	Timestamp   time.Time              `json:"timestamp" db:"timestamp"`
	Source      string                 `json:"source" db:"source"`
	Version     string                 `json:"version,omitempty" db:"version"`
	Environment string                 `json:"environment" db:"environment"`
	IPAddress   string                 `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent   string                 `json:"user_agent,omitempty" db:"user_agent"`
	Referrer    string                 `json:"referrer,omitempty" db:"referrer"`
	DeviceType  string                 `json:"device_type,omitempty" db:"device_type"`
	Country     string                 `json:"country,omitempty" db:"country"`
	Region      string                 `json:"region,omitempty" db:"region"`
	City        string                 `json:"city,omitempty" db:"city"`
}

// NewEvent creates a new analytics event
func NewEvent(name EventName, category EventCategory) *Event {
	return &Event{
		ID:        uuid.New(),
		Name:      name,
		Category:  category,
		Timestamp: time.Now().UTC(),
		Source:    "api",
	}
}

// WithUser sets the user ID for the event
func (e *Event) WithUser(userID uuid.UUID) *Event {
	e.UserID = &userID
	return e
}

// WithSession sets the session ID for the event
func (e *Event) WithSession(sessionID string) *Event {
	e.SessionID = sessionID
	return e
}

// WithTenant sets the tenant ID for the event
func (e *Event) WithTenant(tenantID uuid.UUID) *Event {
	e.TenantID = &tenantID
	return e
}

// WithProperties sets the properties for the event
func (e *Event) WithProperties(props map[string]interface{}) *Event {
	e.Properties = props
	return e
}

// WithProperty adds a single property to the event
func (e *Event) WithProperty(key string, value interface{}) *Event {
	if e.Properties == nil {
		e.Properties = make(map[string]interface{})
	}
	e.Properties[key] = value
	return e
}

// WithSource sets the source of the event
func (e *Event) WithSource(source string) *Event {
	e.Source = source
	return e
}

// WithEnvironment sets the environment for the event
func (e *Event) WithEnvironment(env string) *Event {
	e.Environment = env
	return e
}

// WithVersion sets the app version for the event
func (e *Event) WithVersion(version string) *Event {
	e.Version = version
	return e
}

// WithLocation sets the location information for the event
func (e *Event) WithLocation(country, region, city string) *Event {
	e.Country = country
	e.Region = region
	e.City = city
	return e
}

// WithDeviceInfo sets device information for the event
func (e *Event) WithDeviceInfo(deviceType, userAgent, ipAddress string) *Event {
	e.DeviceType = deviceType
	e.UserAgent = userAgent
	e.IPAddress = ipAddress
	return e
}

// PropertiesJSON returns the properties as JSON bytes
func (e *Event) PropertiesJSON() ([]byte, error) {
	if e.Properties == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(e.Properties)
}

// EventBatch represents a batch of events for bulk processing
type EventBatch struct {
	Events    []*Event  `json:"events"`
	BatchID   string    `json:"batch_id"`
	Timestamp time.Time `json:"timestamp"`
}

// NewEventBatch creates a new event batch
func NewEventBatch(events []*Event) *EventBatch {
	return &EventBatch{
		Events:    events,
		BatchID:   uuid.New().String(),
		Timestamp: time.Now().UTC(),
	}
}

// EventTracker is the interface for tracking events
type EventTracker interface {
	Track(ctx context.Context, event *Event) error
	TrackBatch(ctx context.Context, batch *EventBatch) error
	Flush(ctx context.Context) error
}

// EventStore is the interface for storing events
type EventStore interface {
	Save(ctx context.Context, event *Event) error
	SaveBatch(ctx context.Context, events []*Event) error
	Query(ctx context.Context, query EventQuery) ([]*Event, error)
	Count(ctx context.Context, query EventQuery) (int64, error)
	Delete(ctx context.Context, before time.Time) (int64, error)
}

// EventQuery represents a query for events
type EventQuery struct {
	Names      []EventName     `json:"names,omitempty"`
	Categories []EventCategory `json:"categories,omitempty"`
	UserID     *uuid.UUID      `json:"user_id,omitempty"`
	TenantID   *uuid.UUID      `json:"tenant_id,omitempty"`
	SessionID  string          `json:"session_id,omitempty"`
	StartTime  *time.Time      `json:"start_time,omitempty"`
	EndTime    *time.Time      `json:"end_time,omitempty"`
	Limit      int             `json:"limit,omitempty"`
	Offset     int             `json:"offset,omitempty"`
	OrderBy    string          `json:"order_by,omitempty"`
	OrderDir   string          `json:"order_dir,omitempty"`
}

// UserActivityEvent creates a user activity event
func UserActivityEvent(name EventName, userID uuid.UUID) *Event {
	return NewEvent(name, CategoryUser).WithUser(userID)
}

// PageViewEvent creates a page view event
func PageViewEvent(path, title string, userID *uuid.UUID) *Event {
	event := NewEvent(EventPageView, CategoryPage).
		WithProperty("path", path).
		WithProperty("title", title)
	if userID != nil {
		event.WithUser(*userID)
	}
	return event
}

// FeatureEvent creates a feature usage event
func FeatureEvent(feature string, action string, userID uuid.UUID) *Event {
	return NewEvent(EventFeatureUsed, CategoryFeature).
		WithUser(userID).
		WithProperty("feature", feature).
		WithProperty("action", action)
}

// BusinessEvent creates a business metrics event
func BusinessEvent(name EventName, entityID uuid.UUID, entityType string, value float64) *Event {
	return NewEvent(name, CategoryBusiness).
		WithProperty("entity_id", entityID.String()).
		WithProperty("entity_type", entityType).
		WithProperty("value", value)
}

// PerformanceEvent creates a performance metrics event
func PerformanceEvent(name EventName, duration time.Duration, operation string) *Event {
	return NewEvent(name, CategoryPerformance).
		WithProperty("duration_ms", duration.Milliseconds()).
		WithProperty("operation", operation)
}

// APIEvent creates an API usage event
func APIEvent(method, path string, statusCode int, duration time.Duration, userID *uuid.UUID) *Event {
	event := NewEvent(EventAPILatency, CategoryAPI).
		WithProperty("method", method).
		WithProperty("path", path).
		WithProperty("status_code", statusCode).
		WithProperty("duration_ms", duration.Milliseconds())
	if userID != nil {
		event.WithUser(*userID)
	}
	return event
}

// ErrorEvent creates an error event
func ErrorEvent(errorType, message string, userID *uuid.UUID) *Event {
	event := NewEvent(EventErrorOccurred, CategoryError).
		WithProperty("error_type", errorType).
		WithProperty("message", message)
	if userID != nil {
		event.WithUser(*userID)
	}
	return event
}
