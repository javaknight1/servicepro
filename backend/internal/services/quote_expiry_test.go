package services

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/javaknight1/servicepro/backend/internal/models"
)

// Audit 7.M4: nothing expired quotes. The quote-expiry job moves sent and
// viewed quotes past valid_until to expired, for every tenant, idempotently.
func TestExpireQuotes(t *testing.T) {
	f := setupIsolation(t)
	past, future := time.Now().Add(-time.Hour), time.Now().Add(72*time.Hour)

	mk := func(ctx context.Context, customerID uuid.UUID, status models.QuoteStatus, validUntil time.Time) uuid.UUID {
		q := &models.Quote{ID: uuid.New(), QuoteNumber: "Q-" + uuid.NewString()[:8], CustomerID: customerID,
			Status: status, ValidUntil: future, CreatedBy: f.userA}
		require.NoError(t, f.db.WithContext(ctx).Create(q).Error)
		require.NoError(t, f.db.WithContext(ctx).Exec("UPDATE quotes SET valid_until = ? WHERE id = ?", validUntil, q.ID).Error)
		return q.ID
	}
	sentA := mk(f.ctxA, f.custA.ID, models.QuoteStatusSent, past)
	viewedB := mk(f.ctxB, f.custB.ID, models.QuoteStatusViewed, past)
	draft := mk(f.ctxA, f.custA.ID, models.QuoteStatusDraft, past)
	accepted := mk(f.ctxA, f.custA.ID, models.QuoteStatusAccepted, past)
	stillValid := mk(f.ctxA, f.custA.ID, models.QuoteStatusSent, future)

	n, err := ExpireQuotes(context.Background(), f.db, time.Now())
	require.NoError(t, err)
	assert.Equal(t, int64(2), n)

	status := func(ctx context.Context, id uuid.UUID) models.QuoteStatus {
		var q models.Quote
		require.NoError(t, f.db.WithContext(ctx).First(&q, "id = ?", id).Error)
		return q.Status
	}
	assert.Equal(t, models.QuoteStatusExpired, status(f.ctxA, sentA))
	assert.Equal(t, models.QuoteStatusExpired, status(f.ctxB, viewedB))
	assert.Equal(t, models.QuoteStatusDraft, status(f.ctxA, draft))
	assert.Equal(t, models.QuoteStatusAccepted, status(f.ctxA, accepted))
	assert.Equal(t, models.QuoteStatusSent, status(f.ctxA, stillValid))

	n, err = ExpireQuotes(context.Background(), f.db, time.Now())
	require.NoError(t, err)
	assert.Zero(t, n, "re-running changes nothing")
}
