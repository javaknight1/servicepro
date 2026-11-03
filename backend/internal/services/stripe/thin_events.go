package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/stripe/stripe-go/v87/webhook"

	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

// ThinEvent is the part of a v2 "thin" event notification ServicePro reads.
// Thin events carry no object snapshot: handlers re-fetch the object.
type ThinEvent struct {
	ID            string `json:"id"`
	Object        string `json:"object"`
	Type          string `json:"type"`
	Livemode      bool   `json:"livemode"`
	RelatedObject *struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	} `json:"related_object"`
}

// ThinEventFunc processes one verified, claimed thin event.
type ThinEventFunc func(ctx context.Context, event *ThinEvent) error

// ThinEventHandler receives v2 thin events from a Stripe event destination
// (v2.core.account[...] for Connect, T169): signature-verified with its own
// secret and deduplicated through the same EventStore as v1 webhooks.
type ThinEventHandler struct {
	secret  string
	store   EventStore
	process ThinEventFunc
}

// NewThinEventHandler creates the handler. process is called once per event.
func NewThinEventHandler(secret string, process ThinEventFunc) *ThinEventHandler {
	return &ThinEventHandler{secret: secret, store: newMemoryEventStore(), process: process}
}

// SetEventStore installs the persistent dedupe store.
func (h *ThinEventHandler) SetEventStore(store EventStore) {
	if store != nil {
		h.store = store
	}
}

// Handle verifies, claims and processes one delivery. Errors wrapping
// ErrWebhookRetryLater should be answered non-2xx so Stripe retries.
func (h *ThinEventHandler) Handle(ctx context.Context, payload []byte, signature string) (*ThinEvent, error) {
	if h.secret == "" {
		return nil, errors.New("webhook secret not configured")
	}
	if signature == "" {
		return nil, errors.New("webhook signature is missing")
	}
	// Same scheme as v1 (Stripe-Signature, 5-minute tolerance).
	if err := webhook.ValidatePayload(payload, signature, h.secret); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}
	var ev ThinEvent
	if err := json.Unmarshal(payload, &ev); err != nil || ev.ID == "" || ev.Type == "" {
		return nil, errors.New("malformed event notification")
	}

	claim, err := h.store.Claim(ctx, ev.ID, ev.Type)
	if err != nil {
		return &ev, fmt.Errorf("%w: %w", ErrWebhookRetryLater, err)
	}
	switch claim {
	case ClaimDuplicate:
		return &ev, nil
	case ClaimInProgress:
		return &ev, ErrWebhookRetryLater
	}

	if h.process != nil {
		if err := h.process(ctx, &ev); err != nil {
			_ = h.store.Release(context.WithoutCancel(ctx), ev.ID)
			return &ev, fmt.Errorf("%w: %w", ErrWebhookRetryLater, err)
		}
	}
	if err := h.store.MarkProcessed(context.WithoutCancel(ctx), ev.ID); err != nil {
		logging.Error(ctx, "[Stripe] Failed to mark thin event processed", map[string]any{"event_id": ev.ID, "error": err})
	}
	return &ev, nil
}

// HandleHTTPWebhook is the HTTP entry point.
func (h *ThinEventHandler) HandleHTTPWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxWebhookBodyBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeWebhookJSON(w, http.StatusRequestEntityTooLarge, &WebhookResponse{Error: "request body too large"})
			return
		}
		writeWebhookJSON(w, http.StatusBadRequest, &WebhookResponse{Error: "failed to read request body"})
		return
	}
	ev, err := h.Handle(ctx, body, r.Header.Get("Stripe-Signature"))
	resp := &WebhookResponse{Received: err == nil}
	if ev != nil {
		resp.EventID = ev.ID
	}
	if err != nil {
		status := http.StatusBadRequest
		resp.Error = "invalid event"
		if errors.Is(err, ErrWebhookRetryLater) {
			status = http.StatusServiceUnavailable
			resp.Error = "event processing failed"
		}
		logging.Warn(ctx, "[Stripe] Thin event rejected", map[string]any{"error": err})
		writeWebhookJSON(w, status, resp)
		return
	}
	writeWebhookJSON(w, http.StatusOK, resp)
}

func writeWebhookJSON(w http.ResponseWriter, status int, resp *WebhookResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}

// IsAccountEvent reports whether a thin event is about a v2 account
// (v2.core.account.updated, v2.core.account[requirements].updated,
// v2.core.account[configuration.merchant].capability_status_updated, ...).
func IsAccountEvent(ev *ThinEvent) bool {
	return ev != nil && strings.HasPrefix(ev.Type, "v2.core.account") &&
		ev.RelatedObject != nil && ev.RelatedObject.Type == "v2.core.account" && ev.RelatedObject.ID != ""
}
