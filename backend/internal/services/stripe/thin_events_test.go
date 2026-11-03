package stripe

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stripe/stripe-go/v87/webhook"
)

const thinSecret = "whsec_thin_test"

func thinPayload(id, typ, relatedID string) []byte {
	return []byte(fmt.Sprintf(`{"id":%q,"object":"v2.core.event","type":%q,"livemode":false,"created":"2026-11-02T10:00:00.000Z","related_object":{"id":%q,"type":"v2.core.account","url":"/v2/core/accounts/%s"}}`,
		id, typ, relatedID, relatedID))
}

func deliverThin(h *ThinEventHandler, payload []byte, secret string) *httptest.ResponseRecorder {
	signed := webhook.GenerateTestSignedPayload(&webhook.UnsignedPayload{Payload: payload, Secret: secret})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/stripe/webhooks/v2", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", signed.Header)
	w := httptest.NewRecorder()
	h.HandleHTTPWebhook(w, req)
	return w
}

// T169: v2 account events are verified, deduplicated and retried on failure.
func TestThinEvents_VerifyDedupeRetry(t *testing.T) {
	var seen []string
	fail := true
	h := NewThinEventHandler(thinSecret, func(_ context.Context, ev *ThinEvent) error {
		if !IsAccountEvent(ev) {
			return nil
		}
		if fail {
			fail = false
			return errors.New("stripe down")
		}
		seen = append(seen, ev.RelatedObject.ID)
		return nil
	})
	p := thinPayload("evt_v2_1", "v2.core.account[requirements].updated", "acct_123")

	assert.Equal(t, http.StatusServiceUnavailable, deliverThin(h, p, thinSecret).Code, "failure asks Stripe to retry")
	assert.Equal(t, http.StatusOK, deliverThin(h, p, thinSecret).Code)
	assert.Equal(t, http.StatusOK, deliverThin(h, p, thinSecret).Code, "duplicate is acknowledged")
	assert.Equal(t, []string{"acct_123"}, seen, "processed exactly once")

	bad := deliverThin(h, thinPayload("evt_v2_2", "v2.core.account.updated", "acct_123"), "whsec_wrong")
	assert.Equal(t, http.StatusBadRequest, bad.Code, "wrong signature is rejected")
	require.Len(t, seen, 1)
}

func TestIsAccountEvent(t *testing.T) {
	ev := &ThinEvent{Type: "v2.core.account[configuration.merchant].capability_status_updated"}
	assert.False(t, IsAccountEvent(ev), "no related object")
	ev.RelatedObject = &struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}{ID: "acct_1", Type: "v2.core.account"}
	assert.True(t, IsAccountEvent(ev))
	ev.Type = "v2.billing.meter.error_report_triggered"
	assert.False(t, IsAccountEvent(ev))
}
