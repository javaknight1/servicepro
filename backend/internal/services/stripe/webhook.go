package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"

	"github.com/javaknight1/servicepro/backend/config"
)

// Default webhook tolerance for timestamp validation
const defaultWebhookTolerance = 5 * time.Minute

// WebhookHandler handles incoming Stripe webhooks
type WebhookHandler struct {
	webhookSecret     string
	webhookSecretFile string // Optional: path to file containing webhook secret (for dynamic loading)
	webhookTolerance  time.Duration
	eventProcessor    *EventProcessor
	mu                sync.RWMutex

	// Event tracking for idempotency
	processedEvents map[string]time.Time
	eventMu         sync.RWMutex
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(cfg *config.Config, eventProcessor *EventProcessor) *WebhookHandler {
	// Get webhook secret file path from environment (for Docker Compose with Stripe CLI)
	webhookSecretFile := os.Getenv("STRIPE_WEBHOOK_SECRET_FILE")

	handler := &WebhookHandler{
		webhookSecret:     cfg.Stripe.WebhookSecret,
		webhookSecretFile: webhookSecretFile,
		webhookTolerance:  defaultWebhookTolerance,
		eventProcessor:    eventProcessor,
		processedEvents:   make(map[string]time.Time),
	}

	// Log webhook configuration status
	if cfg.Stripe.WebhookSecret != "" {
		log.Printf("[Stripe] Webhook handler initialized with secret (length=%d)", len(cfg.Stripe.WebhookSecret))
	} else if webhookSecretFile != "" {
		log.Printf("[Stripe] Webhook handler will read secret from file: %s", webhookSecretFile)
	} else {
		log.Printf("[Stripe] WARNING: Webhook secret is empty - webhooks will fail signature verification")
	}

	// Start cleanup goroutine for processed events
	go handler.cleanupProcessedEvents()

	return handler
}

// WebhookRequest represents an incoming webhook request
type WebhookRequest struct {
	Payload   []byte
	Signature string
}

// WebhookResponse represents the response to a webhook
type WebhookResponse struct {
	Received bool   `json:"received"`
	EventID  string `json:"event_id,omitempty"`
	Error    string `json:"error,omitempty"`
}

// HandleWebhook processes an incoming Stripe webhook
func (h *WebhookHandler) HandleWebhook(ctx context.Context, req *WebhookRequest) (*WebhookResponse, error) {
	if req == nil {
		return nil, errors.New("webhook request cannot be nil")
	}

	if len(req.Payload) == 0 {
		return nil, errors.New("webhook payload is empty")
	}

	if req.Signature == "" {
		return nil, errors.New("webhook signature is missing")
	}

	// Verify webhook signature
	event, err := h.verifySignature(req.Payload, req.Signature)
	if err != nil {
		log.Printf("[Stripe] Webhook signature verification failed: %v", err)
		return &WebhookResponse{
			Received: false,
			Error:    "signature verification failed",
		}, fmt.Errorf("signature verification failed: %w", err)
	}

	log.Printf("[Stripe] Webhook received: id=%s, type=%s", event.ID, event.Type)

	// Check if event was already processed (idempotency)
	if h.wasEventProcessed(event.ID) {
		log.Printf("[Stripe] Event already processed: id=%s", event.ID)
		return &WebhookResponse{
			Received: true,
			EventID:  event.ID,
		}, nil
	}

	// Process the event
	if h.eventProcessor != nil {
		if err := h.eventProcessor.ProcessEvent(ctx, event); err != nil {
			log.Printf("[Stripe] Failed to process event: id=%s, type=%s, error=%v",
				event.ID, event.Type, err)
			return &WebhookResponse{
				Received: false,
				EventID:  event.ID,
				Error:    "event processing failed",
			}, fmt.Errorf("event processing failed: %w", err)
		}
	}

	// Mark event as processed
	h.markEventProcessed(event.ID)

	log.Printf("[Stripe] Event processed successfully: id=%s, type=%s", event.ID, event.Type)

	return &WebhookResponse{
		Received: true,
		EventID:  event.ID,
	}, nil
}

// HandleHTTPWebhook processes an HTTP webhook request
func (h *WebhookHandler) HandleHTTPWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	log.Printf("[Stripe] Webhook HTTP request received from %s", r.RemoteAddr)

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[Stripe] Failed to read webhook body: %v", err)
		h.sendErrorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	log.Printf("[Stripe] Webhook body length: %d bytes", len(body))

	// Get signature
	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		log.Printf("[Stripe] Missing Stripe-Signature header")
		h.sendErrorResponse(w, http.StatusBadRequest, "missing signature")
		return
	}

	// Process webhook
	req := &WebhookRequest{
		Payload:   body,
		Signature: signature,
	}

	response, err := h.HandleWebhook(ctx, req)
	if err != nil {
		// Determine status code based on error
		statusCode := http.StatusBadRequest
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			statusCode = http.StatusRequestTimeout
		}

		h.sendJSONResponse(w, statusCode, response)
		return
	}

	h.sendJSONResponse(w, http.StatusOK, response)
}

// getWebhookSecret returns the webhook secret, reading from file if necessary
func (h *WebhookHandler) getWebhookSecret() string {
	h.mu.RLock()
	secret := h.webhookSecret
	h.mu.RUnlock()

	// If we have a secret in memory, use it
	if secret != "" {
		return secret
	}

	// Try to read from file if configured
	if h.webhookSecretFile != "" {
		data, err := os.ReadFile(h.webhookSecretFile)
		if err == nil {
			secret = strings.TrimSpace(string(data))
			if secret != "" {
				// Cache it for future use
				h.mu.Lock()
				h.webhookSecret = secret
				h.mu.Unlock()
				log.Printf("[Stripe] Loaded webhook secret from file: %s", h.webhookSecretFile)
				return secret
			}
		}
	}

	return ""
}

// verifySignature verifies the webhook signature and returns the parsed event
func (h *WebhookHandler) verifySignature(payload []byte, signature string) (*stripe.Event, error) {
	// Get the webhook secret (from memory or file)
	secret := h.getWebhookSecret()
	if secret == "" {
		return nil, errors.New("webhook secret not configured")
	}

	// Construct event with signature verification
	// Use ConstructEventWithOptions to handle API version mismatches between
	// Stripe CLI/dashboard and the stripe-go library version
	event, err := webhook.ConstructEventWithOptions(
		payload,
		signature,
		secret,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	// Additional timestamp check for replay attacks
	if time.Since(time.Unix(event.Created, 0)) > h.webhookTolerance {
		return nil, fmt.Errorf("webhook timestamp too old: event created at %v",
			time.Unix(event.Created, 0))
	}

	return &event, nil
}

// wasEventProcessed checks if an event was already processed
func (h *WebhookHandler) wasEventProcessed(eventID string) bool {
	h.eventMu.RLock()
	defer h.eventMu.RUnlock()

	_, exists := h.processedEvents[eventID]
	return exists
}

// markEventProcessed marks an event as processed
func (h *WebhookHandler) markEventProcessed(eventID string) {
	h.eventMu.Lock()
	defer h.eventMu.Unlock()

	h.processedEvents[eventID] = time.Now()
}

// cleanupProcessedEvents removes old processed events (runs every hour)
func (h *WebhookHandler) cleanupProcessedEvents() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		h.eventMu.Lock()

		// Remove events older than 24 hours
		cutoff := time.Now().Add(-24 * time.Hour)
		for eventID, processedAt := range h.processedEvents {
			if processedAt.Before(cutoff) {
				delete(h.processedEvents, eventID)
			}
		}

		h.eventMu.Unlock()

		log.Printf("[Stripe] Cleaned up old processed events, current count: %d",
			len(h.processedEvents))
	}
}

// sendJSONResponse sends a JSON response
func (h *WebhookHandler) sendJSONResponse(w http.ResponseWriter, statusCode int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[Stripe] Failed to encode response: %v", err)
	}
}

// sendErrorResponse sends an error response
func (h *WebhookHandler) sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	h.sendJSONResponse(w, statusCode, &WebhookResponse{
		Received: false,
		Error:    message,
	})
}

// TestWebhookSignature tests webhook signature verification (for testing only)
func (h *WebhookHandler) TestWebhookSignature(payload []byte, signature string) error {
	_, err := h.verifySignature(payload, signature)
	return err
}

// GetProcessedEventsCount returns the number of processed events in memory
func (h *WebhookHandler) GetProcessedEventsCount() int {
	h.eventMu.RLock()
	defer h.eventMu.RUnlock()
	return len(h.processedEvents)
}

// ClearProcessedEvents clears all processed events (for testing)
func (h *WebhookHandler) ClearProcessedEvents() {
	h.eventMu.Lock()
	defer h.eventMu.Unlock()
	h.processedEvents = make(map[string]time.Time)
}
