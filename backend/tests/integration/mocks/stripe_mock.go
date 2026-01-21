//go:build integration

package mocks

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// Stripe Mock Server
// =============================================================================

// StripeMock provides a mock Stripe API server
type StripeMock struct {
	server         *httptest.Server
	mu             sync.Mutex
	customers      map[string]*StripeCustomer
	paymentIntents map[string]*StripePaymentIntent
	charges        map[string]*StripeCharge
	webhookSecret  string
	webhookEvents  []WebhookEvent
}

// StripeCustomer represents a Stripe customer
type StripeCustomer struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Created int64  `json:"created"`
}

// StripePaymentIntent represents a Stripe payment intent
type StripePaymentIntent struct {
	ID            string `json:"id"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
	Status        string `json:"status"`
	CustomerID    string `json:"customer"`
	ClientSecret  string `json:"client_secret"`
	ChargeID      string `json:"latest_charge,omitempty"`
	PaymentMethod string `json:"payment_method,omitempty"`
	Created       int64  `json:"created"`
}

// StripeCharge represents a Stripe charge
type StripeCharge struct {
	ID              string `json:"id"`
	Amount          int64  `json:"amount"`
	Currency        string `json:"currency"`
	Status          string `json:"status"`
	PaymentIntentID string `json:"payment_intent"`
	CustomerID      string `json:"customer"`
	ReceiptURL      string `json:"receipt_url"`
	Created         int64  `json:"created"`
}

// WebhookEvent represents a Stripe webhook event
type WebhookEvent struct {
	Type string
	Data interface{}
}

// NewStripeMock creates a new Stripe mock server
func NewStripeMock(t *testing.T) *StripeMock {
	t.Helper()

	mock := &StripeMock{
		customers:      make(map[string]*StripeCustomer),
		paymentIntents: make(map[string]*StripePaymentIntent),
		charges:        make(map[string]*StripeCharge),
		webhookSecret:  "whsec_test_" + uuid.New().String()[:16],
	}

	mux := http.NewServeMux()

	// Customer endpoints
	mux.HandleFunc("/v1/customers", mock.handleCustomers)
	mux.HandleFunc("/v1/customers/", mock.handleCustomer)

	// Payment Intent endpoints
	mux.HandleFunc("/v1/payment_intents", mock.handlePaymentIntents)
	mux.HandleFunc("/v1/payment_intents/", mock.handlePaymentIntent)

	// Charge endpoints
	mux.HandleFunc("/v1/charges", mock.handleCharges)
	mux.HandleFunc("/v1/charges/", mock.handleCharge)

	// Refund endpoints
	mux.HandleFunc("/v1/refunds", mock.handleRefunds)

	mock.server = httptest.NewServer(mux)
	return mock
}

// URL returns the mock server URL
func (m *StripeMock) URL() string {
	return m.server.URL
}

// Close closes the mock server
func (m *StripeMock) Close() {
	m.server.Close()
}

// WebhookSecret returns the webhook signing secret
func (m *StripeMock) WebhookSecret() string {
	return m.webhookSecret
}

// =============================================================================
// Customer Handlers
// =============================================================================

func (m *StripeMock) handleCustomers(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch r.Method {
	case http.MethodPost:
		// Create customer
		r.ParseForm()
		customer := &StripeCustomer{
			ID:      "cus_" + uuid.New().String()[:14],
			Email:   r.FormValue("email"),
			Name:    r.FormValue("name"),
			Created: time.Now().Unix(),
		}
		m.customers[customer.ID] = customer
		jsonResponse(w, http.StatusOK, customer)

	case http.MethodGet:
		// List customers
		var customers []*StripeCustomer
		for _, c := range m.customers {
			customers = append(customers, c)
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"object": "list",
			"data":   customers,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *StripeMock) handleCustomer(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := strings.TrimPrefix(r.URL.Path, "/v1/customers/")
	customer, ok := m.customers[id]
	if !ok {
		jsonError(w, http.StatusNotFound, "customer", "Customer not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		jsonResponse(w, http.StatusOK, customer)

	case http.MethodPost:
		r.ParseForm()
		if email := r.FormValue("email"); email != "" {
			customer.Email = email
		}
		if name := r.FormValue("name"); name != "" {
			customer.Name = name
		}
		jsonResponse(w, http.StatusOK, customer)

	case http.MethodDelete:
		delete(m.customers, id)
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"id":      id,
			"object":  "customer",
			"deleted": true,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// =============================================================================
// Payment Intent Handlers
// =============================================================================

func (m *StripeMock) handlePaymentIntents(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch r.Method {
	case http.MethodPost:
		r.ParseForm()

		amount := int64(0)
		fmt.Sscanf(r.FormValue("amount"), "%d", &amount)

		pi := &StripePaymentIntent{
			ID:           "pi_" + uuid.New().String()[:14],
			Amount:       amount,
			Currency:     r.FormValue("currency"),
			Status:       "requires_payment_method",
			CustomerID:   r.FormValue("customer"),
			ClientSecret: "pi_secret_" + uuid.New().String()[:16],
			Created:      time.Now().Unix(),
		}
		m.paymentIntents[pi.ID] = pi
		jsonResponse(w, http.StatusOK, pi)

	case http.MethodGet:
		var intents []*StripePaymentIntent
		for _, pi := range m.paymentIntents {
			intents = append(intents, pi)
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"object": "list",
			"data":   intents,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *StripeMock) handlePaymentIntent(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := strings.TrimPrefix(r.URL.Path, "/v1/payment_intents/")
	parts := strings.Split(path, "/")
	id := parts[0]

	pi, ok := m.paymentIntents[id]
	if !ok {
		jsonError(w, http.StatusNotFound, "payment_intent", "Payment intent not found")
		return
	}

	// Handle confirm action
	if len(parts) > 1 && parts[1] == "confirm" {
		pi.Status = "succeeded"

		// Create associated charge
		charge := &StripeCharge{
			ID:              "ch_" + uuid.New().String()[:14],
			Amount:          pi.Amount,
			Currency:        pi.Currency,
			Status:          "succeeded",
			PaymentIntentID: pi.ID,
			CustomerID:      pi.CustomerID,
			ReceiptURL:      "https://pay.stripe.com/receipts/" + uuid.New().String()[:16],
			Created:         time.Now().Unix(),
		}
		m.charges[charge.ID] = charge
		pi.ChargeID = charge.ID

		jsonResponse(w, http.StatusOK, pi)
		return
	}

	// Handle cancel action
	if len(parts) > 1 && parts[1] == "cancel" {
		pi.Status = "canceled"
		jsonResponse(w, http.StatusOK, pi)
		return
	}

	switch r.Method {
	case http.MethodGet:
		jsonResponse(w, http.StatusOK, pi)

	case http.MethodPost:
		r.ParseForm()
		if pm := r.FormValue("payment_method"); pm != "" {
			pi.PaymentMethod = pm
			pi.Status = "requires_confirmation"
		}
		jsonResponse(w, http.StatusOK, pi)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// =============================================================================
// Charge Handlers
// =============================================================================

func (m *StripeMock) handleCharges(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch r.Method {
	case http.MethodGet:
		var charges []*StripeCharge
		for _, c := range m.charges {
			charges = append(charges, c)
		}
		jsonResponse(w, http.StatusOK, map[string]interface{}{
			"object": "list",
			"data":   charges,
		})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *StripeMock) handleCharge(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := strings.TrimPrefix(r.URL.Path, "/v1/charges/")
	charge, ok := m.charges[id]
	if !ok {
		jsonError(w, http.StatusNotFound, "charge", "Charge not found")
		return
	}

	jsonResponse(w, http.StatusOK, charge)
}

// =============================================================================
// Refund Handlers
// =============================================================================

func (m *StripeMock) handleRefunds(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	chargeID := r.FormValue("charge")
	amount := int64(0)
	fmt.Sscanf(r.FormValue("amount"), "%d", &amount)

	charge, ok := m.charges[chargeID]
	if !ok {
		jsonError(w, http.StatusNotFound, "charge", "Charge not found")
		return
	}

	if amount == 0 {
		amount = charge.Amount
	}

	refund := map[string]interface{}{
		"id":       "re_" + uuid.New().String()[:14],
		"amount":   amount,
		"currency": charge.Currency,
		"charge":   chargeID,
		"status":   "succeeded",
		"created":  time.Now().Unix(),
	}

	jsonResponse(w, http.StatusOK, refund)
}

// =============================================================================
// Test Helpers
// =============================================================================

// CreateCustomer creates a test customer
func (m *StripeMock) CreateCustomer(email, name string) *StripeCustomer {
	m.mu.Lock()
	defer m.mu.Unlock()

	customer := &StripeCustomer{
		ID:      "cus_" + uuid.New().String()[:14],
		Email:   email,
		Name:    name,
		Created: time.Now().Unix(),
	}
	m.customers[customer.ID] = customer
	return customer
}

// CreatePaymentIntent creates a test payment intent
func (m *StripeMock) CreatePaymentIntent(amount int64, customerID string) *StripePaymentIntent {
	m.mu.Lock()
	defer m.mu.Unlock()

	pi := &StripePaymentIntent{
		ID:           "pi_" + uuid.New().String()[:14],
		Amount:       amount,
		Currency:     "usd",
		Status:       "requires_payment_method",
		CustomerID:   customerID,
		ClientSecret: "pi_secret_" + uuid.New().String()[:16],
		Created:      time.Now().Unix(),
	}
	m.paymentIntents[pi.ID] = pi
	return pi
}

// ConfirmPaymentIntent marks a payment intent as succeeded
func (m *StripeMock) ConfirmPaymentIntent(piID string) *StripeCharge {
	m.mu.Lock()
	defer m.mu.Unlock()

	pi, ok := m.paymentIntents[piID]
	if !ok {
		return nil
	}

	pi.Status = "succeeded"

	charge := &StripeCharge{
		ID:              "ch_" + uuid.New().String()[:14],
		Amount:          pi.Amount,
		Currency:        pi.Currency,
		Status:          "succeeded",
		PaymentIntentID: pi.ID,
		CustomerID:      pi.CustomerID,
		ReceiptURL:      "https://pay.stripe.com/receipts/" + uuid.New().String()[:16],
		Created:         time.Now().Unix(),
	}
	m.charges[charge.ID] = charge
	pi.ChargeID = charge.ID

	return charge
}

// AddWebhookEvent adds a webhook event for later verification
func (m *StripeMock) AddWebhookEvent(eventType string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.webhookEvents = append(m.webhookEvents, WebhookEvent{
		Type: eventType,
		Data: data,
	})
}

// GetWebhookEvents returns all recorded webhook events
func (m *StripeMock) GetWebhookEvents() []WebhookEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.webhookEvents
}

// =============================================================================
// JSON Helpers
// =============================================================================

func jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, status int, errType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"type":    errType,
			"message": message,
		},
	})
}
