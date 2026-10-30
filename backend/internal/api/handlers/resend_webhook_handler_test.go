package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingSuppressor struct{ got []string }

func (r *recordingSuppressor) Suppress(_ context.Context, email, reason, _ string) error {
	r.got = append(r.got, reason+":"+email)
	return nil
}

var testWebhookKey = []byte("super-secret-signing-key")

func sign(id string, ts time.Time, body string) (string, string) {
	t := strconv.FormatInt(ts.Unix(), 10)
	mac := hmac.New(sha256.New, testWebhookKey)
	mac.Write([]byte(id + "." + t + "." + body))
	return t, "v1," + base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func postWebhook(t *testing.T, h *ResendWebhookHandler, body, id, ts, sig string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/webhooks/resend", h.HandleWebhook)
	req := httptest.NewRequest(http.MethodPost, "/webhooks/resend", bytes.NewBufferString(body))
	req.Header.Set("svix-id", id)
	req.Header.Set("svix-timestamp", ts)
	req.Header.Set("svix-signature", sig)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func newTestWebhook(t *testing.T, now time.Time) (*ResendWebhookHandler, *recordingSuppressor) {
	t.Helper()
	s := &recordingSuppressor{}
	h, err := NewResendWebhookHandler("whsec_"+base64.StdEncoding.EncodeToString(testWebhookKey), s)
	require.NoError(t, err)
	h.now = func() time.Time { return now }
	return h, s
}

// T159: signed bounce and complaint events suppress the recipients.
func TestResendWebhook_SuppressesBouncesAndComplaints(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	h, s := newTestWebhook(t, now)

	body := `{"type":"email.bounced","data":{"email_id":"e1","to":["a@example.test"],"bounce":{"type":"Permanent"}}}`
	ts, sig := sign("msg_1", now, body)
	assert.Equal(t, http.StatusNoContent, postWebhook(t, h, body, "msg_1", ts, sig).Code)

	body = `{"type":"email.complained","data":{"email_id":"e2","to":["b@example.test"]}}`
	ts, sig = sign("msg_2", now, body)
	assert.Equal(t, http.StatusNoContent, postWebhook(t, h, body, "msg_2", ts, sig).Code)

	assert.Equal(t, []string{"bounced:a@example.test", "complained:b@example.test"}, s.got)
}

func TestResendWebhook_IgnoresTransientBouncesAndOtherEvents(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	h, s := newTestWebhook(t, now)
	for i, body := range []string{
		`{"type":"email.bounced","data":{"to":["a@example.test"],"bounce":{"type":"Transient"}}}`,
		`{"type":"email.delivered","data":{"to":["a@example.test"]}}`,
	} {
		id := "msg_" + strconv.Itoa(i)
		ts, sig := sign(id, now, body)
		assert.Equal(t, http.StatusNoContent, postWebhook(t, h, body, id, ts, sig).Code)
	}
	assert.Empty(t, s.got)
}

func TestResendWebhook_RejectsBadSignatures(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	h, s := newTestWebhook(t, now)
	body := `{"type":"email.complained","data":{"to":["a@example.test"]}}`

	ts, sig := sign("msg_1", now, body)
	assert.Equal(t, http.StatusBadRequest, postWebhook(t, h, body+" ", "msg_1", ts, sig).Code, "tampered body")
	assert.Equal(t, http.StatusBadRequest, postWebhook(t, h, body, "msg_other", ts, sig).Code, "wrong id")

	oldTS, oldSig := sign("msg_1", now.Add(-10*time.Minute), body)
	assert.Equal(t, http.StatusBadRequest, postWebhook(t, h, body, "msg_1", oldTS, oldSig).Code, "replayed/stale")

	assert.Equal(t, http.StatusBadRequest, postWebhook(t, h, body, "", "", "").Code, "unsigned")
	assert.Empty(t, s.got)
}

func TestNewResendWebhookHandler_RequiresSecret(t *testing.T) {
	_, err := NewResendWebhookHandler("", &recordingSuppressor{})
	assert.Error(t, err)
}
