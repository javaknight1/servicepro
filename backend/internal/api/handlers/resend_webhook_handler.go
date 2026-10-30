package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/javaknight1/servicepro/backend/internal/api/apierror"
	"github.com/javaknight1/servicepro/backend/internal/models"
	"github.com/javaknight1/servicepro/backend/pkg/clients/logging"
)

const (
	resendWebhookMaxBody   = 256 << 10
	resendWebhookTolerance = 5 * time.Minute
)

// EmailSuppressor records an address that must not be emailed again.
type EmailSuppressor interface {
	Suppress(ctx context.Context, email, reason, eventID string) error
}

// ResendWebhookHandler receives Resend's signed delivery events (T159).
type ResendWebhookHandler struct {
	secret     []byte
	suppressor EmailSuppressor
	now        func() time.Time
}

// NewResendWebhookHandler takes the endpoint's signing secret ("whsec_…").
func NewResendWebhookHandler(secret string, suppressor EmailSuppressor) (*ResendWebhookHandler, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(secret, "whsec_"))
	if err != nil || len(key) == 0 {
		return nil, errors.New("RESEND_WEBHOOK_SECRET must be the whsec_… signing secret")
	}
	return &ResendWebhookHandler{secret: key, suppressor: suppressor, now: time.Now}, nil
}

type resendEvent struct {
	Type string `json:"type"`
	Data struct {
		EmailID string   `json:"email_id"`
		To      []string `json:"to"`
		Bounce  *struct {
			Type string `json:"type"`
		} `json:"bounce,omitempty"`
	} `json:"data"`
}

// HandleWebhook verifies the Svix signature and suppresses bounced or
// complained recipients.
// @Summary Resend delivery webhook
// @Description Signed (Svix) Resend events. email.bounced (permanent) and email.complained add the recipients to the suppression list; other events are acknowledged and ignored.
// @Tags Webhooks
// @Accept json
// @Produce json
// @Success 204
// @Failure 400 {object} models.ErrorResponse "invalid signature or payload"
// @Router /api/v1/webhooks/resend [post]
func (h *ResendWebhookHandler) HandleWebhook(c *gin.Context) {
	body, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, resendWebhookMaxBody))
	if err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "invalid_payload", Message: "Unreadable webhook body"})
		return
	}
	if err := verifySvixSignature(h.secret, c.GetHeader("svix-id"), c.GetHeader("svix-timestamp"),
		c.GetHeader("svix-signature"), body, h.now(), resendWebhookTolerance); err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "invalid_signature", Message: "Webhook signature verification failed"})
		return
	}

	var ev resendEvent
	if err := json.Unmarshal(body, &ev); err != nil {
		apierror.WriteResponse(c, http.StatusBadRequest, models.ErrorResponse{Error: "invalid_payload", Message: "Malformed webhook payload"})
		return
	}

	reason := ""
	switch ev.Type {
	case "email.bounced":
		// Transient bounces (mailbox full, etc.) may succeed later.
		if ev.Data.Bounce == nil || !strings.EqualFold(ev.Data.Bounce.Type, "Transient") {
			reason = "bounced"
		}
	case "email.complained":
		reason = "complained"
	}
	if reason != "" {
		ctx := c.Request.Context()
		for _, addr := range ev.Data.To {
			if err := h.suppressor.Suppress(ctx, addr, reason, ev.Data.EmailID); err != nil {
				// 500 makes Resend retry; Suppress is idempotent.
				apierror.Internal(c, err, "Could not record the suppression")
				return
			}
		}
		logging.Info(ctx, "[RESEND-WEBHOOK] Suppressed recipients", map[string]any{"type": ev.Type, "count": len(ev.Data.To)})
	}
	c.Status(http.StatusNoContent)
}

// verifySvixSignature checks a Svix-style signature: base64(HMAC-SHA256(key,
// "<id>.<timestamp>.<body>")) must match one "v1,<sig>" entry, and the
// timestamp must be within tolerance (replay protection).
func verifySvixSignature(key []byte, id, timestamp, signatures string, body []byte, now time.Time, tolerance time.Duration) error {
	if id == "" || timestamp == "" || signatures == "" {
		return errors.New("missing signature headers")
	}
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return errors.New("bad timestamp")
	}
	if d := now.Sub(time.Unix(ts, 0)); d > tolerance || d < -tolerance {
		return errors.New("timestamp outside tolerance")
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(id + "." + timestamp + "."))
	mac.Write(body)
	expected := mac.Sum(nil)
	for _, part := range strings.Fields(signatures) {
		version, sig, ok := strings.Cut(part, ",")
		if !ok || version != "v1" {
			continue
		}
		got, err := base64.StdEncoding.DecodeString(sig)
		if err == nil && hmac.Equal(got, expected) {
			return nil
		}
	}
	return errors.New("no matching signature")
}
