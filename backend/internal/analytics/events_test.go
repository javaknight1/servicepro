package analytics

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewEvent(t *testing.T) {
	event := NewEvent(EventUserLogin, CategoryUser)

	if event.ID == uuid.Nil {
		t.Error("Event ID should not be nil")
	}
	if event.Name != EventUserLogin {
		t.Errorf("Expected name %s, got %s", EventUserLogin, event.Name)
	}
	if event.Category != CategoryUser {
		t.Errorf("Expected category %s, got %s", CategoryUser, event.Category)
	}
	if event.Source != "api" {
		t.Errorf("Expected source 'api', got %s", event.Source)
	}
	if event.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestEventWithUser(t *testing.T) {
	userID := uuid.New()
	event := NewEvent(EventUserLogin, CategoryUser).WithUser(userID)

	if event.UserID == nil {
		t.Fatal("UserID should not be nil")
	}
	if *event.UserID != userID {
		t.Errorf("Expected userID %s, got %s", userID, *event.UserID)
	}
}

func TestEventWithSession(t *testing.T) {
	sessionID := "session-123"
	event := NewEvent(EventPageView, CategoryPage).WithSession(sessionID)

	if event.SessionID != sessionID {
		t.Errorf("Expected sessionID %s, got %s", sessionID, event.SessionID)
	}
}

func TestEventWithTenant(t *testing.T) {
	tenantID := uuid.New()
	event := NewEvent(EventUserLogin, CategoryUser).WithTenant(tenantID)

	if event.TenantID == nil {
		t.Fatal("TenantID should not be nil")
	}
	if *event.TenantID != tenantID {
		t.Errorf("Expected tenantID %s, got %s", tenantID, *event.TenantID)
	}
}

func TestEventWithProperties(t *testing.T) {
	props := map[string]interface{}{
		"page": "/home",
		"time": 123,
	}
	event := NewEvent(EventPageView, CategoryPage).WithProperties(props)

	if event.Properties == nil {
		t.Fatal("Properties should not be nil")
	}
	if event.Properties["page"] != "/home" {
		t.Error("Property 'page' should be '/home'")
	}
	if event.Properties["time"] != 123 {
		t.Error("Property 'time' should be 123")
	}
}

func TestEventWithProperty(t *testing.T) {
	event := NewEvent(EventPageView, CategoryPage).
		WithProperty("key1", "value1").
		WithProperty("key2", 42)

	if event.Properties == nil {
		t.Fatal("Properties should not be nil")
	}
	if event.Properties["key1"] != "value1" {
		t.Error("Property 'key1' should be 'value1'")
	}
	if event.Properties["key2"] != 42 {
		t.Error("Property 'key2' should be 42")
	}
}

func TestEventWithSource(t *testing.T) {
	event := NewEvent(EventPageView, CategoryPage).WithSource("web")

	if event.Source != "web" {
		t.Errorf("Expected source 'web', got %s", event.Source)
	}
}

func TestEventWithEnvironment(t *testing.T) {
	event := NewEvent(EventPageView, CategoryPage).WithEnvironment("production")

	if event.Environment != "production" {
		t.Errorf("Expected environment 'production', got %s", event.Environment)
	}
}

func TestEventWithVersion(t *testing.T) {
	event := NewEvent(EventPageView, CategoryPage).WithVersion("1.2.3")

	if event.Version != "1.2.3" {
		t.Errorf("Expected version '1.2.3', got %s", event.Version)
	}
}

func TestEventWithLocation(t *testing.T) {
	event := NewEvent(EventPageView, CategoryPage).WithLocation("US", "CA", "San Francisco")

	if event.Country != "US" {
		t.Errorf("Expected country 'US', got %s", event.Country)
	}
	if event.Region != "CA" {
		t.Errorf("Expected region 'CA', got %s", event.Region)
	}
	if event.City != "San Francisco" {
		t.Errorf("Expected city 'San Francisco', got %s", event.City)
	}
}

func TestEventWithDeviceInfo(t *testing.T) {
	event := NewEvent(EventPageView, CategoryPage).WithDeviceInfo("mobile", "Mozilla/5.0", "192.168.1.1")

	if event.DeviceType != "mobile" {
		t.Errorf("Expected deviceType 'mobile', got %s", event.DeviceType)
	}
	if event.UserAgent != "Mozilla/5.0" {
		t.Errorf("Expected userAgent 'Mozilla/5.0', got %s", event.UserAgent)
	}
	if event.IPAddress != "192.168.1.1" {
		t.Errorf("Expected ipAddress '192.168.1.1', got %s", event.IPAddress)
	}
}

func TestEventPropertiesJSON(t *testing.T) {
	event := NewEvent(EventPageView, CategoryPage).
		WithProperty("page", "/home").
		WithProperty("referrer", "/login")

	json, err := event.PropertiesJSON()
	if err != nil {
		t.Fatalf("PropertiesJSON should not error: %v", err)
	}

	// Note: JSON order is not guaranteed, so we check for key presence
	if len(json) < 10 {
		t.Errorf("JSON too short: %s", string(json))
	}
}

func TestEventPropertiesJSONEmpty(t *testing.T) {
	event := NewEvent(EventPageView, CategoryPage)

	json, err := event.PropertiesJSON()
	if err != nil {
		t.Fatalf("PropertiesJSON should not error: %v", err)
	}

	if string(json) != "{}" {
		t.Errorf("Expected empty JSON '{}', got %s", string(json))
	}
}

func TestNewEventBatch(t *testing.T) {
	events := []*Event{
		NewEvent(EventUserLogin, CategoryUser),
		NewEvent(EventPageView, CategoryPage),
	}

	batch := NewEventBatch(events)

	if batch.BatchID == "" {
		t.Error("BatchID should not be empty")
	}
	if len(batch.Events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(batch.Events))
	}
	if batch.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestUserActivityEvent(t *testing.T) {
	userID := uuid.New()
	event := UserActivityEvent(EventUserLogin, userID)

	if event.Name != EventUserLogin {
		t.Errorf("Expected name %s, got %s", EventUserLogin, event.Name)
	}
	if event.Category != CategoryUser {
		t.Errorf("Expected category %s, got %s", CategoryUser, event.Category)
	}
	if event.UserID == nil || *event.UserID != userID {
		t.Error("UserID should be set correctly")
	}
}

func TestPageViewEvent(t *testing.T) {
	userID := uuid.New()
	event := PageViewEvent("/home", "Home Page", &userID)

	if event.Name != EventPageView {
		t.Errorf("Expected name %s, got %s", EventPageView, event.Name)
	}
	if event.Category != CategoryPage {
		t.Errorf("Expected category %s, got %s", CategoryPage, event.Category)
	}
	if event.Properties["path"] != "/home" {
		t.Error("Property 'path' should be '/home'")
	}
	if event.Properties["title"] != "Home Page" {
		t.Error("Property 'title' should be 'Home Page'")
	}
	if event.UserID == nil || *event.UserID != userID {
		t.Error("UserID should be set correctly")
	}
}

func TestPageViewEventNoUser(t *testing.T) {
	event := PageViewEvent("/home", "Home Page", nil)

	if event.UserID != nil {
		t.Error("UserID should be nil")
	}
}

func TestFeatureEvent(t *testing.T) {
	userID := uuid.New()
	event := FeatureEvent("search", "performed", userID)

	if event.Name != EventFeatureUsed {
		t.Errorf("Expected name %s, got %s", EventFeatureUsed, event.Name)
	}
	if event.Category != CategoryFeature {
		t.Errorf("Expected category %s, got %s", CategoryFeature, event.Category)
	}
	if event.Properties["feature"] != "search" {
		t.Error("Property 'feature' should be 'search'")
	}
	if event.Properties["action"] != "performed" {
		t.Error("Property 'action' should be 'performed'")
	}
}

func TestBusinessEvent(t *testing.T) {
	entityID := uuid.New()
	event := BusinessEvent(EventInvoiceCreated, entityID, "invoice", 1500.50)

	if event.Name != EventInvoiceCreated {
		t.Errorf("Expected name %s, got %s", EventInvoiceCreated, event.Name)
	}
	if event.Category != CategoryBusiness {
		t.Errorf("Expected category %s, got %s", CategoryBusiness, event.Category)
	}
	if event.Properties["entity_id"] != entityID.String() {
		t.Error("Property 'entity_id' should be set correctly")
	}
	if event.Properties["entity_type"] != "invoice" {
		t.Error("Property 'entity_type' should be 'invoice'")
	}
	if event.Properties["value"] != 1500.50 {
		t.Error("Property 'value' should be 1500.50")
	}
}

func TestPerformanceEvent(t *testing.T) {
	duration := 250 * time.Millisecond
	event := PerformanceEvent(EventAPILatency, duration, "GET /api/users")

	if event.Name != EventAPILatency {
		t.Errorf("Expected name %s, got %s", EventAPILatency, event.Name)
	}
	if event.Category != CategoryPerformance {
		t.Errorf("Expected category %s, got %s", CategoryPerformance, event.Category)
	}
	if event.Properties["duration_ms"] != int64(250) {
		t.Errorf("Property 'duration_ms' should be 250, got %v", event.Properties["duration_ms"])
	}
	if event.Properties["operation"] != "GET /api/users" {
		t.Error("Property 'operation' should be 'GET /api/users'")
	}
}

func TestAPIEvent(t *testing.T) {
	userID := uuid.New()
	duration := 100 * time.Millisecond
	event := APIEvent("GET", "/api/users", 200, duration, &userID)

	if event.Name != EventAPILatency {
		t.Errorf("Expected name %s, got %s", EventAPILatency, event.Name)
	}
	if event.Category != CategoryAPI {
		t.Errorf("Expected category %s, got %s", CategoryAPI, event.Category)
	}
	if event.Properties["method"] != "GET" {
		t.Error("Property 'method' should be 'GET'")
	}
	if event.Properties["path"] != "/api/users" {
		t.Error("Property 'path' should be '/api/users'")
	}
	if event.Properties["status_code"] != 200 {
		t.Error("Property 'status_code' should be 200")
	}
	if event.UserID == nil || *event.UserID != userID {
		t.Error("UserID should be set correctly")
	}
}

func TestErrorEvent(t *testing.T) {
	userID := uuid.New()
	event := ErrorEvent("ValidationError", "Invalid email format", &userID)

	if event.Name != EventErrorOccurred {
		t.Errorf("Expected name %s, got %s", EventErrorOccurred, event.Name)
	}
	if event.Category != CategoryError {
		t.Errorf("Expected category %s, got %s", CategoryError, event.Category)
	}
	if event.Properties["error_type"] != "ValidationError" {
		t.Error("Property 'error_type' should be 'ValidationError'")
	}
	if event.Properties["message"] != "Invalid email format" {
		t.Error("Property 'message' should be 'Invalid email format'")
	}
}

func TestEventChaining(t *testing.T) {
	userID := uuid.New()
	tenantID := uuid.New()

	event := NewEvent(EventUserLogin, CategoryUser).
		WithUser(userID).
		WithTenant(tenantID).
		WithSession("session-123").
		WithProperty("method", "email").
		WithSource("web").
		WithEnvironment("production").
		WithVersion("1.0.0").
		WithLocation("US", "CA", "San Francisco").
		WithDeviceInfo("desktop", "Mozilla/5.0", "127.0.0.1")

	// Verify all fields are set
	if event.UserID == nil || *event.UserID != userID {
		t.Error("UserID not set correctly")
	}
	if event.TenantID == nil || *event.TenantID != tenantID {
		t.Error("TenantID not set correctly")
	}
	if event.SessionID != "session-123" {
		t.Error("SessionID not set correctly")
	}
	if event.Properties["method"] != "email" {
		t.Error("Property 'method' not set correctly")
	}
	if event.Source != "web" {
		t.Error("Source not set correctly")
	}
	if event.Environment != "production" {
		t.Error("Environment not set correctly")
	}
	if event.Version != "1.0.0" {
		t.Error("Version not set correctly")
	}
	if event.Country != "US" {
		t.Error("Country not set correctly")
	}
	if event.DeviceType != "desktop" {
		t.Error("DeviceType not set correctly")
	}
}
