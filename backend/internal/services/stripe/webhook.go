package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"

	"github.com/javaknight1/servicepro/backend/config"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// MaxWebhookBodyBytes caps the (unauthenticated) webhook body before it is
// read. Stripe events are far smaller.
const MaxWebhookBodyBytes = 1 << 20 // 1 MiB

// ErrWebhookRetryLater marks failures Stripe should retry (non-400 response).
var ErrWebhookRetryLater = errors.New("webhook should be retried")

// WebhookHandler handles incoming Stripe webhooks.
//
// Replay protection is the signature timestamp tolerance enforced by
// webhook.ConstructEvent (5 minutes on the Stripe-Signature t=, which Stripe
// refreshes on every retry). There is deliberately NO check on the event's
// creation time: Stripe retries for days with the original Created value,
// and rejecting those would drop payments after one failed delivery (F-10).
// Duplicate deliveries are filtered by a persistent EventStore.
type WebhookHandler struct {
	webhookSecret     string
	webhookSecretFile string // Optional: path to file containing webhook secret (for dynamic loading)
	eventProcessor    *EventProcessor
	mu                sync.RWMutex

	store EventStore
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(cfg *config.Config, eventProcessor *EventProcessor) *WebhookHandler {
	// Get webhook secret file path from environment (for Docker Compose with Stripe CLI)
	webhookSecretFile := os.Getenv("STRIPE_WEBHOOK_SECRET_FILE")

	handler := &WebhookHandler{
		webhookSecret:     cfg.Stripe.WebhookSecret,
		webhookSecretFile: webhookSecretFile,
		eventProcessor:    eventProcessor,
		store:             newMemoryEventStore(),
	}

	// Log webhook configuration status
	switch {
	case cfg.Stripe.WebhookSecret != "":
		logging.Info(context.Background(), "[Stripe] Webhook handler initialized with secret", map[string]any{"secret_length": len(cfg.Stripe.WebhookSecret)})
	case webhookSecretFile != "":
		logging.Info(context.Background(), "[Stripe] Webhook handler will read secret from file", map[string]any{"file": webhookSecretFile})
	default:
		logging.Warn(context.Background(), "[Stripe] Webhook secret is empty - webhooks will fail signature verification", nil)
	}

	return handler
}

// SetEventStore installs the persistent dedupe store (stripe_webhook_events).
// Without it deduplication is per process only.
func (h *WebhookHandler) SetEventStore(store EventStore) {
	if store != nil {
		h.store = store
	}
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
		logging.Error(ctx, "[Stripe] Webhook signature verification failed", map[string]any{"error": err})
		return &WebhookResponse{
			Received: false,
			Error:    "signature verification failed",
		}, fmt.Errorf("signature verification failed: %w", err)
	}

	logging.Info(ctx, "[Stripe] Webhook received", map[string]any{"event_id": event.ID, "event_type": event.Type})

	// Claim the event ID before processing (INSERT ... ON CONFLICT DO
	// NOTHING): duplicates are acknowledged, concurrent deliveries retried.
	claim, err := h.store.Claim(ctx, event.ID, string(event.Type))
	if err != nil {
		return &WebhookResponse{Received: false, EventID: event.ID, Error: "event store unavailable"},
			fmt.Errorf("%w: %v", ErrWebhookRetryLater, err)
	}
	switch claim {
	case ClaimDuplicate:
		logging.Info(ctx, "[Stripe] Event already processed", map[string]any{"event_id": event.ID})
		return &WebhookResponse{Received: true, EventID: event.ID}, nil
	case ClaimInProgress:
		return &WebhookResponse{Received: false, EventID: event.ID, Error: "event is being processed"}, ErrWebhookRetryLater
	}

	// Process the event
	if h.eventProcessor != nil {
		if err := h.eventProcessor.ProcessEvent(ctx, event); err != nil {
			logging.Error(ctx, "[Stripe] Failed to process event", map[string]any{"event_id": event.ID, "event_type": event.Type, "error": err})
			_ = h.store.Release(context.WithoutCancel(ctx), event.ID)
			return &WebhookResponse{
				Received: false,
				EventID:  event.ID,
				Error:    "event processing failed",
			}, fmt.Errorf("%w: event processing failed: %v", ErrWebhookRetryLater, err)
		}
	}

	// Mark event as processed
	if err := h.store.MarkProcessed(context.WithoutCancel(ctx), event.ID); err != nil {
		logging.Error(ctx, "[Stripe] Failed to mark event processed", map[string]any{"event_id": event.ID, "error": err})
	}

	logging.Info(ctx, "[Stripe] Event processed successfully", map[string]any{"event_id": event.ID, "event_type": event.Type})

	return &WebhookResponse{
		Received: true,
		EventID:  event.ID,
	}, nil
}

// HandleHTTPWebhook processes an HTTP webhook request
func (h *WebhookHandler) HandleHTTPWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logging.Info(ctx, "[Stripe] Webhook HTTP request received", map[string]any{"remote_addr": r.RemoteAddr})

	// Read body (capped: the endpoint is unauthenticated until the
	// signature is checked).
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxWebhookBodyBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			h.sendErrorResponse(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		logging.Error(ctx, "[Stripe] Failed to read webhook body", map[string]any{"error": err})
		h.sendErrorResponse(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	logging.Info(ctx, "[Stripe] Webhook body length", map[string]any{"bytes": len(body)})

	// Get signature
	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		logging.Warn(ctx, "[Stripe] Missing Stripe-Signature header", nil)
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
		// Signature/format problems are permanent (400); processing
		// failures and in-flight duplicates ask Stripe to retry (503).
		statusCode := http.StatusBadRequest
		switch {
		case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
			statusCode = http.StatusRequestTimeout
		case errors.Is(err, ErrWebhookRetryLater):
			statusCode = http.StatusServiceUnavailable
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
				logging.Info(context.Background(), "[Stripe] Loaded webhook secret from file", map[string]any{"file": h.webhookSecretFile})
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

	return &event, nil
}

// sendJSONResponse sends a JSON response
func (h *WebhookHandler) sendJSONResponse(w http.ResponseWriter, statusCode int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logging.Error(context.Background(), "[Stripe] Failed to encode response", map[string]any{"error": err})
	}
}

// sendErrorResponse sends an error response
func (h *WebhookHandler) sendErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	h.sendJSONResponse(w, statusCode, &WebhookResponse{
		Received: false,
		Error:    message,
	})
}
