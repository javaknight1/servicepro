package services

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	stripeClient "github.com/javaknight1/servicepro/backend/internal/services/stripe"
)

type fakePlanLookup struct {
	tiers map[uuid.UUID]string
	err   error
}

func (f fakePlanLookup) ActiveTierName(_ context.Context, tenantID uuid.UUID) (string, error) {
	return f.tiers[tenantID], f.err
}

// T210: the invoice application fee comes from the tenant's plan; every
// plan maps to the D10 fees today, and a lookup failure falls back to them.
func TestPlatformFeesFor_UsesTenantPlan(t *testing.T) {
	ctx := context.Background()
	solo, pro := uuid.New(), uuid.New()
	plans := fakePlanLookup{tiers: map[uuid.UUID]string{solo: "solo", pro: "pro"}}

	for _, id := range []uuid.UUID{solo, pro, uuid.New()} {
		fees := platformFeesFor(ctx, plans, id)
		assert.Equal(t, stripeClient.DefaultPlatformFees.CardBps, fees.CardBps)
		assert.Equal(t, int64(1000), fees.ApplicationFee(500000, "usd", stripeClient.PayMethodBank), "1%% capped at $10")
		assert.Equal(t, int64(384), fees.ApplicationFee(96000, "usd", stripeClient.PayMethodCard), "0.4%%")
	}
	assert.Equal(t, stripeClient.DefaultPlatformFees.BankBps, platformFeesFor(ctx, nil, solo).BankBps)
	assert.Equal(t, stripeClient.DefaultPlatformFees.BankBps,
		platformFeesFor(ctx, fakePlanLookup{err: errors.New("db down")}, solo).BankBps)
}
