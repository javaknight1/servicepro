package stripe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/javaknight1/servicepro/backend/config"
)

const testWebhookSecret = "whsec_test_secret"

func newTestWebhook(t *testing.T, handler EventHandler) *WebhookHandler {
	t.Helper()
	cfg := &config.Config{}
	cfg.Stripe.WebhookSecret = testWebhookSecret
	p := NewEventProcessor()
	p.RegisterHandler("checkout.session.completed", handler)
	return NewWebhookHandler(cfg, p)
}

func deliver(h *WebhookHandler, eventID string, created time.Time) *httptest.ResponseRecorder {
	payload := []byte(fmt.Sprintf(`{"id":%q,"object":"event","type":"checkout.session.completed","created":%d,"api_version":"2023-10-16","data":{"object":{}}}`,
		eventID, created.Unix()))
	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{Payload: payload, Secret: testWebhookSecret})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stripe/webhooks", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", signed.Header)
	w := httptest.NewRecorder()
	h.HandleHTTPWebhook(w, req)
	return w
}

func TestWebhook_AcceptsRetriesOfOldEvents(t *testing.T) {
	calls := 0
	h := newTestWebhook(t, func(context.Context, *stripe.Event) error { calls++; return nil })

	// Stripe retries keep the original `created`; only the signature
	// timestamp is fresh. Such retries must be processed.
	w := deliver(h, "evt_old", time.Now().Add(-48*time.Hour))
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
	assert.Equal(t, 1, calls)
}

func TestWebhook_DeduplicatesAndRetriesFailures(t *testing.T) {
	calls := 0
	fail := true
	h := newTestWebhook(t, func(context.Context, *stripe.Event) error {
		calls++
		if fail {
			return errors.New("db down")
		}
		return nil
	})

	w := deliver(h, "evt_1", time.Now())
	assert.Equal(t, http.StatusServiceUnavailable, w.Code, "processing failure asks Stripe to retry")

	fail = false
	assert.Equal(t, http.StatusOK, deliver(h, "evt_1", time.Now()).Code, "retry is processed")
	assert.Equal(t, http.StatusOK, deliver(h, "evt_1", time.Now()).Code, "duplicate is acknowledged")
	assert.Equal(t, 2, calls, "processed once successfully, never twice")
}

func TestWebhook_BodyCap(t *testing.T) {
	h := newTestWebhook(t, func(context.Context, *stripe.Event) error { return nil })
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stripe/webhooks", strings.NewReader(strings.Repeat("a", MaxWebhookBodyBytes+1)))
	req.Header.Set("Stripe-Signature", "t=1,v1=00")
	w := httptest.NewRecorder()
	h.HandleHTTPWebhook(w, req)
	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestWebhook_RejectsBadSignature(t *testing.T) {
	h := newTestWebhook(t, func(context.Context, *stripe.Event) error { return nil })
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stripe/webhooks", strings.NewReader(`{"id":"evt_x"}`))
	req.Header.Set("Stripe-Signature", "t=1,v1=deadbeef")
	w := httptest.NewRecorder()
	h.HandleHTTPWebhook(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGormEventStore(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&StripeWebhookEvent{}))
	store := NewGormEventStore(db)
	ctx := context.Background()

	res, err := store.Claim(ctx, "evt_a", "x")
	require.NoError(t, err)
	assert.Equal(t, ClaimProcess, res)

	res, err = store.Claim(ctx, "evt_a", "x")
	require.NoError(t, err)
	assert.Equal(t, ClaimInProgress, res, "concurrent delivery while processing")

	require.NoError(t, store.Release(ctx, "evt_a"))
	res, err = store.Claim(ctx, "evt_a", "x")
	require.NoError(t, err)
	assert.Equal(t, ClaimProcess, res, "released claim can be retried")

	require.NoError(t, store.MarkProcessed(ctx, "evt_a"))
	res, err = store.Claim(ctx, "evt_a", "x")
	require.NoError(t, err)
	assert.Equal(t, ClaimDuplicate, res)

	// A stale "processing" claim (crashed instance) can be taken over.
	_, err = store.Claim(ctx, "evt_b", "x")
	require.NoError(t, err)
	require.NoError(t, db.Model(&StripeWebhookEvent{}).Where("event_id = ?", "evt_b").
		Update("received_at", time.Now().Add(-time.Hour)).Error)
	res, err = store.Claim(ctx, "evt_b", "x")
	require.NoError(t, err)
	assert.Equal(t, ClaimProcess, res)
}
