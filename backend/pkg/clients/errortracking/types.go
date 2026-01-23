package errortracking

import (
	"time"
)

// Provider represents an error tracking service provider
type Provider string

const (
	ProviderSentry Provider = "sentry"
	ProviderMock   Provider = "mock"
)

// Level represents error severity level
type Level string

const (
	LevelDebug   Level = "debug"
	LevelInfo    Level = "info"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
	LevelFatal   Level = "fatal"
)

// Event represents an error or message to be tracked
type Event struct {
	// Message is the event message
	Message string

	// Level is the severity level
	Level Level

	// Error is the error to capture (optional)
	Error error

	// Tags are key-value pairs for filtering
	Tags map[string]string

	// Extra is additional context data
	Extra map[string]any

	// User is the user associated with the event
	User *User

	// Request is HTTP request context
	Request *Request

	// Fingerprint groups similar events
	Fingerprint []string

	// Timestamp is when the event occurred
	Timestamp time.Time

	// Transaction is the transaction name
	Transaction string
}

// User represents user information for error context
type User struct {
	ID        string
	Email     string
	Username  string
	IPAddress string
	Segment   string
	Data      map[string]any
}

// Request represents HTTP request context
type Request struct {
	Method  string
	URL     string
	Query   string
	Headers map[string]string
	Data    any
	Env     map[string]string
}

// EventID is a unique identifier for a captured event
type EventID string

// Breadcrumb represents a single breadcrumb
type Breadcrumb struct {
	// Type is the breadcrumb type (default, http, navigation, etc.)
	Type string

	// Category is a dotted string for categorization
	Category string

	// Message is the breadcrumb message
	Message string

	// Data is additional data
	Data map[string]any

	// Level is the breadcrumb level
	Level Level

	// Timestamp is when the breadcrumb was recorded
	Timestamp time.Time
}

// Scope represents the current scope with tags, extras, and user info
type Scope interface {
	// SetTag sets a tag
	SetTag(key, value string)

	// SetTags sets multiple tags
	SetTags(tags map[string]string)

	// RemoveTag removes a tag
	RemoveTag(key string)

	// SetExtra sets extra context data
	SetExtra(key string, value any)

	// SetExtras sets multiple extra context data
	SetExtras(extras map[string]any)

	// RemoveExtra removes extra context data
	RemoveExtra(key string)

	// SetUser sets user information
	SetUser(user *User)

	// SetRequest sets request information
	SetRequest(request *Request)

	// SetFingerprint sets the event fingerprint
	SetFingerprint(fingerprint []string)

	// SetLevel sets the level
	SetLevel(level Level)

	// SetTransaction sets the transaction name
	SetTransaction(name string)

	// AddBreadcrumb adds a breadcrumb
	AddBreadcrumb(breadcrumb *Breadcrumb)

	// ClearBreadcrumbs clears all breadcrumbs
	ClearBreadcrumbs()

	// Clear clears the scope
	Clear()

	// Clone creates a copy of the scope
	Clone() Scope
}
